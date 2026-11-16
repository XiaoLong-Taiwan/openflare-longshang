// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"Wavelet/openflare/plugins/server/kernel/model"
	"Wavelet/openflare/plugins/server/kernel/repository"
	"Wavelet/pkg/logger"

	"gorm.io/gorm"
)

const (
	memberLockStripeCount  = 64
	memberLastErrorColumn  = "last_error"
	memberSyncStatusColumn = "sync_status"
)

type managedKey struct {
	nodeID     uint
	recordType string
}

var memberLocks [memberLockStripeCount]sync.Mutex

// ReconcileMember makes one Cloudflare A record match the local desired state.
func ReconcileMember(ctx context.Context, memberID uint) error {
	lock := &memberLocks[memberID%memberLockStripeCount]
	lock.Lock()
	defer lock.Unlock()

	if err := repository.UpdateCFPointingMemberColumns(ctx, memberID, map[string]any{memberSyncStatusColumn: model.CFMemberSyncing, memberLastErrorColumn: ""}); err != nil {
		return err
	}
	if err := reconcileMember(ctx, memberID); err != nil {
		if updateErr := repository.UpdateCFPointingMemberColumns(ctx, memberID, map[string]any{memberSyncStatusColumn: model.CFMemberSyncError, memberLastErrorColumn: errSyncFailed}); updateErr != nil {
			logger.ErrorF(ctx, "[Cloudflare] persist member sync error failed: member_id=%d error=%v", memberID, updateErr)
		}
		return err
	}
	return nil
}

func reconcileMember(ctx context.Context, memberID uint) error {
	state, err := repository.GetCFPointingMemberContext(ctx, memberID)
	if err != nil {
		return err
	}
	targets, err := memberTargets(state)
	if err != nil {
		return err
	}
	connection, err := repository.GetCFConnection(ctx)
	if err != nil || connection.Status != model.CFConnectionStatusReady {
		return errors.New(errConnectionNotConfigured)
	}
	token, err := resolveToken(ctx, connection)
	if err != nil {
		return err
	}
	client := clientFactory(token)
	zoneID, err := memberZoneID(ctx, client, state)
	if err != nil {
		return err
	}
	managed, err := repository.ListCFPointingManagedRecords(ctx, memberID)
	if err != nil {
		return err
	}
	if len(managed) == 0 && state.Member.CFRecordID != "" {
		managed = append(managed, model.CFPointingManagedRecord{MemberID: memberID, NodeID: state.Group.ActiveNodeID, CFRecordID: state.Member.CFRecordID, DesiredIP: state.Member.DesiredIP})
	}
	managedByKey := make(map[managedKey]*model.CFPointingManagedRecord, len(managed))
	for i := range managed {
		recordType := managed[i].RecordType
		if recordType == "" {
			recordType = "A"
		}
		managedByKey[managedKey{nodeID: managed[i].NodeID, recordType: recordType}] = &managed[i]
	}
	targetKeys := make(map[managedKey]struct{}, len(targets))
	for _, target := range targets {
		for _, input := range memberRecordInputs(state, target) {
			targetKeys[managedKey{nodeID: target.Node.ID, recordType: input.Type}] = struct{}{}
		}
	}
	_, hadCustomTarget := managedByKey[managedKey{nodeID: 0, recordType: state.Group.RecordType}]
	cleanupFirst := state.Group.TargetMode == targetModeCustom || hadCustomTarget
	if cleanupFirst {
		if err := cleanupManagedRecords(ctx, client, zoneID, managed, targetKeys); err != nil {
			return err
		}
	}
	for _, target := range targets {
		for _, input := range memberRecordInputs(state, target) {
			key := managedKey{nodeID: target.Node.ID, recordType: input.Type}
			record, syncErr := syncManagedRecord(ctx, client, zoneID, input, managedByKey[key], memberID, target.Node.ID)
			if syncErr != nil {
				return syncErr
			}
			managedByKey[key] = record
		}
	}
	if !cleanupFirst {
		if err := cleanupManagedRecords(ctx, client, zoneID, managed, targetKeys); err != nil {
			return err
		}
	}
	primary := targets[0]
	primaryRecord := managedByKey[managedKey{nodeID: primary.Node.ID, recordType: "A"}]
	if primaryRecord == nil {
		primaryRecord = managedByKey[managedKey{nodeID: 0, recordType: state.Group.RecordType}]
	}
	return markMemberSynced(ctx, memberID, zoneID, primaryRecord.CFRecordID, primaryRecord.DesiredIP, primary.Node.ID)
}

func memberRecordInputs(state *repository.CFPointingMemberContext, target repository.CFPointingGroupNodeContext) []RecordInput {
	if state.Group.TargetMode == targetModeCustom {
		input := recordInput(state.Domain.Domain, state.Group.RecordType, state.Group.RecordContent, state.Member.Proxied)
		input.TTL = groupTTL(&state.Group)
		if input.Proxied {
			input.TTL = automaticTTL
		}
		return []RecordInput{input}
	}
	inputs := make([]RecordInput, 0, 2)
	if ip := strings.TrimSpace(target.Node.IP); net.ParseIP(ip) != nil && net.ParseIP(ip).To4() != nil {
		inputs = append(inputs, recordInput(state.Domain.Domain, "A", ip, state.Member.Proxied))
	}
	if ip := strings.TrimSpace(target.GroupNode.IPv6); net.ParseIP(ip) != nil && net.ParseIP(ip).To4() == nil {
		inputs = append(inputs, recordInput(state.Domain.Domain, "AAAA", ip, state.Member.Proxied))
	}
	for i := range inputs {
		inputs[i].TTL = groupTTL(&state.Group)
		if inputs[i].Proxied {
			inputs[i].TTL = automaticTTL
		}
	}
	return inputs
}

func memberTargets(state *repository.CFPointingMemberContext) ([]repository.CFPointingGroupNodeContext, error) {
	if !state.Group.Enabled {
		return nil, errors.New(errGroupDisabled)
	}
	if state.Group.TargetMode == targetModeCustom {
		if !validCustomRecordContent(state.Group.RecordType, state.Group.RecordContent) {
			return nil, errors.New(errRecordContentInvalid)
		}
		return []repository.CFPointingGroupNodeContext{{}}, nil
	}
	targets := lowestPriorityAvailableNodes(state.Nodes, time.Duration(state.Group.DebounceSeconds)*time.Second)
	if len(targets) == 0 {
		return nil, errors.New(errNoAvailableNodes)
	}
	return targets, nil
}

func memberZoneID(ctx context.Context, client Client, state *repository.CFPointingMemberContext) (string, error) {
	if state.Member.CFZoneID != "" {
		return state.Member.CFZoneID, nil
	}
	zone, err := client.FindZone(ctx, state.Zone.Domain)
	if err != nil {
		return "", err
	}
	return zone.ID, nil
}

func cleanupManagedRecords(ctx context.Context, client Client, zoneID string, managed []model.CFPointingManagedRecord, targets map[managedKey]struct{}) error {
	for i := range managed {
		recordType := managed[i].RecordType
		if recordType == "" {
			recordType = "A"
		}
		if _, keep := targets[managedKey{nodeID: managed[i].NodeID, recordType: recordType}]; keep {
			continue
		}
		if managed[i].CFRecordID != "" {
			if err := client.DeleteRecord(ctx, zoneID, managed[i].CFRecordID); err != nil && !isNotFoundError(err) {
				return err
			}
		}
		if managed[i].ID != 0 {
			if err := repository.DeleteCFPointingManagedRecord(ctx, &managed[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func syncManagedRecord(ctx context.Context, client Client, zoneID string, input RecordInput, record *model.CFPointingManagedRecord, memberID, nodeID uint) (*model.CFPointingManagedRecord, error) {
	if record != nil && record.CFRecordID != "" {
		_, err := client.GetRecord(ctx, zoneID, record.CFRecordID)
		if err == nil {
			updated, updateErr := client.UpdateARecord(ctx, zoneID, record.CFRecordID, input)
			if updateErr != nil {
				return nil, updateErr
			}
			record.CFRecordID, record.DesiredIP, record.RecordType = updated.ID, input.Content, input.Type
			return record, repository.SaveCFPointingManagedRecord(ctx, record)
		}
		if !isNotFoundError(err) {
			return nil, err
		}
	}
	created, err := client.CreateARecord(ctx, zoneID, input)
	if err != nil {
		return nil, err
	}
	if record == nil {
		record = &model.CFPointingManagedRecord{MemberID: memberID, NodeID: nodeID}
	}
	record.CFRecordID, record.DesiredIP, record.RecordType = created.ID, input.Content, input.Type
	if err := repository.SaveCFPointingManagedRecord(ctx, record); err != nil {
		if deleteErr := client.DeleteRecord(ctx, zoneID, created.ID); deleteErr != nil && !isNotFoundError(deleteErr) {
			logger.ErrorF(ctx, "failed to delete Cloudflare record %s after managed record save failed: %v", created.ID, deleteErr)
		}
		return nil, err
	}
	return record, nil
}

func lowestPriorityAvailableNodes(nodes []repository.CFPointingGroupNodeContext, debounce time.Duration) []repository.CFPointingGroupNodeContext {
	priority := 0
	found := false
	for _, node := range nodes {
		if !nodeAvailable(node.Node, debounce) {
			continue
		}
		if !found || node.GroupNode.Priority < priority {
			priority = node.GroupNode.Priority
			found = true
		}
	}
	if !found {
		return nil
	}
	available := make([]repository.CFPointingGroupNodeContext, 0, len(nodes))
	for _, node := range nodes {
		if nodeAvailable(node.Node, debounce) && node.GroupNode.Priority == priority {
			available = append(available, node)
		}
	}
	return available
}

func nodeAvailable(node model.OpenFlareNode, debounce time.Duration) bool {
	if node.OpenrestyStatus == "unhealthy" || node.LastSeenAt == nil || node.LastSeenAt.IsZero() {
		return false
	}
	age := time.Since(*node.LastSeenAt)
	if node.Status == "offline" {
		return debounce > 0 && age <= debounce
	}
	if node.Status != "online" {
		return false
	}
	return age <= time.Minute
}

func recordInput(name, recordType, ip string, proxied bool) RecordInput {
	ttl := 300
	if proxied {
		ttl = 1
	}
	return RecordInput{Type: recordType, Name: name, Content: ip, Proxied: proxied, TTL: ttl}
}

func markMemberSynced(ctx context.Context, memberID uint, zoneID, recordID, ip string, activeNodeID uint) error {
	now := time.Now()
	member, err := repository.GetCFPointingMemberByID(ctx, memberID)
	if err != nil {
		return err
	}
	updated, err := repository.CompleteCFPointingMemberSync(ctx, memberID, map[string]any{
		"cf_zone_id": zoneID, "cf_record_id": recordID, "desired_ip": ip,
		memberSyncStatusColumn: model.CFMemberSyncOK, memberLastErrorColumn: "", "synced_at": &now,
	})
	if err != nil {
		return err
	}
	if !updated {
		return nil
	}
	if err := repository.DB(ctx).Model(&model.CFPointingGroup{}).Where("id = ?", member.GroupID).Update("active_node_id", activeNodeID).Error; err != nil {
		return err
	}
	return nil
}

// DeleteManagedRecord deletes the cached or uniquely discoverable A record.
func DeleteManagedRecord(ctx context.Context, memberID uint) error {
	state, err := repository.GetCFPointingMemberContext(ctx, memberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	connection, err := repository.GetCFConnection(ctx)
	if err != nil {
		return err
	}
	token, err := resolveToken(ctx, connection)
	if err != nil {
		return err
	}
	client := clientFactory(token)
	zoneID := state.Member.CFZoneID
	if zoneID == "" {
		zone, findErr := client.FindZone(ctx, state.Zone.Domain)
		if findErr != nil {
			return findErr
		}
		zoneID = zone.ID
	}
	managed, err := repository.ListCFPointingManagedRecords(ctx, memberID)
	if err != nil {
		return err
	}
	if len(managed) == 0 && state.Member.CFRecordID != "" {
		managed = append(managed, model.CFPointingManagedRecord{MemberID: memberID, NodeID: state.Group.ActiveNodeID, CFRecordID: state.Member.CFRecordID})
	}
	for i := range managed {
		if managed[i].CFRecordID != "" {
			if deleteErr := client.DeleteRecord(ctx, zoneID, managed[i].CFRecordID); deleteErr != nil && !isNotFoundError(deleteErr) {
				return deleteErr
			}
		}
		if managed[i].ID != 0 {
			if deleteErr := repository.DeleteCFPointingManagedRecord(ctx, &managed[i]); deleteErr != nil {
				return deleteErr
			}
		}
	}
	return nil
}
