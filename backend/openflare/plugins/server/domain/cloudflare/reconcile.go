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
		if updateErr := repository.UpdateCFPointingMemberColumns(ctx, memberID, map[string]any{memberSyncStatusColumn: model.CFMemberSyncError, memberLastErrorColumn: err.Error()}); updateErr != nil {
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
	managedByNode := make(map[uint]*model.CFPointingManagedRecord, len(managed))
	for i := range managed {
		managedByNode[managed[i].NodeID] = &managed[i]
	}
	targetNodeIDs := make(map[uint]struct{}, len(targets))
	for _, target := range targets {
		targetNodeIDs[target.Node.ID] = struct{}{}
	}
	_, hadCustomTarget := managedByNode[0]
	cleanupFirst := state.Group.TargetMode == targetModeCustom || hadCustomTarget
	if cleanupFirst {
		if err := cleanupManagedRecords(ctx, client, zoneID, managed, targetNodeIDs); err != nil {
			return err
		}
	}
	for _, target := range targets {
		input := memberRecordInput(state, target)
		record, syncErr := syncManagedRecord(ctx, client, zoneID, input, managedByNode[target.Node.ID], memberID, target.Node.ID)
		if syncErr != nil {
			return syncErr
		}
		managedByNode[target.Node.ID] = record
	}
	if !cleanupFirst {
		if err := cleanupManagedRecords(ctx, client, zoneID, managed, targetNodeIDs); err != nil {
			return err
		}
	}
	primary := targets[0]
	primaryRecord := managedByNode[primary.Node.ID]
	return markMemberSynced(ctx, memberID, zoneID, primaryRecord.CFRecordID, primaryRecord.DesiredIP, primary.Node.ID)
}

func memberRecordInput(state *repository.CFPointingMemberContext, target repository.CFPointingGroupNodeContext) RecordInput {
	input := recordInput(state.Domain.Domain, strings.TrimSpace(target.Node.IP), state.Member.Proxied)
	if state.Group.TargetMode == targetModeCustom {
		input.Type, input.Content = state.Group.RecordType, state.Group.RecordContent
	}
	input.TTL = groupTTL(&state.Group)
	if input.Proxied {
		input.TTL = automaticTTL
	}
	return input
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
	targets := lowestPriorityAvailableNodes(state.Nodes)
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

func cleanupManagedRecords(ctx context.Context, client Client, zoneID string, managed []model.CFPointingManagedRecord, targets map[uint]struct{}) error {
	for i := range managed {
		if _, keep := targets[managed[i].NodeID]; keep {
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
			record.CFRecordID, record.DesiredIP = updated.ID, input.Content
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
	record.CFRecordID, record.DesiredIP = created.ID, input.Content
	if err := repository.SaveCFPointingManagedRecord(ctx, record); err != nil {
		if deleteErr := client.DeleteRecord(ctx, zoneID, created.ID); deleteErr != nil && !isNotFoundError(deleteErr) {
			logger.ErrorF(ctx, "failed to delete Cloudflare record %s after managed record save failed: %v", created.ID, deleteErr)
		}
		return nil, err
	}
	return record, nil
}

func lowestPriorityAvailableNodes(nodes []repository.CFPointingGroupNodeContext) []repository.CFPointingGroupNodeContext {
	priority := 0
	found := false
	for _, node := range nodes {
		if !nodeAvailable(node.Node) {
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
		if nodeAvailable(node.Node) && node.GroupNode.Priority == priority {
			available = append(available, node)
		}
	}
	return available
}

func nodeAvailable(node model.OpenFlareNode) bool {
	if node.Status != "online" || net.ParseIP(strings.TrimSpace(node.IP)).To4() == nil || node.LastSeenAt == nil || node.LastSeenAt.IsZero() {
		return false
	}
	return time.Since(*node.LastSeenAt) <= 60*time.Second
}

func recordInput(name, ip string, proxied bool) RecordInput {
	ttl := 300
	if proxied {
		ttl = 1
	}
	return RecordInput{Type: "A", Name: name, Content: ip, Proxied: proxied, TTL: ttl}
}

func markMemberSynced(ctx context.Context, memberID uint, zoneID, recordID, ip string, activeNodeID uint) error {
	now := time.Now()
	member, err := repository.GetCFPointingMemberByID(ctx, memberID)
	if err != nil {
		return err
	}
	if err = repository.UpdateCFPointingMemberColumns(ctx, memberID, map[string]any{
		"cf_zone_id": zoneID, "cf_record_id": recordID, "desired_ip": ip,
		memberSyncStatusColumn: model.CFMemberSyncOK, memberLastErrorColumn: "", "synced_at": &now,
	}); err != nil {
		return err
	}
	return repository.DB(ctx).Model(&model.CFPointingGroup{}).Where("id = ?", member.GroupID).Update("active_node_id", activeNodeID).Error
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
