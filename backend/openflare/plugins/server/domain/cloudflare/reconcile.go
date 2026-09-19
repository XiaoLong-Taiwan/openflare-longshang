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
	if !state.Group.Enabled {
		return errors.New(errGroupDisabled)
	}
	targets := lowestPriorityAvailableNodes(state.Nodes)
	if len(targets) == 0 {
		return errors.New(errNoAvailableNodes)
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
		managed = append(managed, model.CFPointingManagedRecord{MemberID: memberID, NodeID: state.Group.ActiveNodeID, CFRecordID: state.Member.CFRecordID, DesiredIP: state.Member.DesiredIP})
	}
	managedByNode := make(map[uint]*model.CFPointingManagedRecord, len(managed))
	for i := range managed {
		managedByNode[managed[i].NodeID] = &managed[i]
	}
	targetNodeIDs := make(map[uint]struct{}, len(targets))
	for _, target := range targets {
		targetNodeIDs[target.Node.ID] = struct{}{}
		input := recordInput(state.Domain.Domain, strings.TrimSpace(target.Node.IP), state.Member.Proxied)
		record := managedByNode[target.Node.ID]
		if record != nil && record.CFRecordID != "" {
			if _, getErr := client.GetRecord(ctx, zoneID, record.CFRecordID); getErr == nil {
				updated, updateErr := client.UpdateARecord(ctx, zoneID, record.CFRecordID, input)
				if updateErr != nil {
					return updateErr
				}
				record.CFRecordID = updated.ID
				record.DesiredIP = input.Content
				if saveErr := repository.SaveCFPointingManagedRecord(ctx, record); saveErr != nil {
					return saveErr
				}
				continue
			} else if !isNotFoundError(getErr) {
				return getErr
			}
		}
		created, createErr := client.CreateARecord(ctx, zoneID, input)
		if createErr != nil {
			return createErr
		}
		if record == nil {
			record = &model.CFPointingManagedRecord{MemberID: memberID, NodeID: target.Node.ID}
		}
		record.CFRecordID = created.ID
		record.DesiredIP = input.Content
		if saveErr := repository.SaveCFPointingManagedRecord(ctx, record); saveErr != nil {
			if deleteErr := client.DeleteRecord(ctx, zoneID, created.ID); deleteErr != nil && !isNotFoundError(deleteErr) {
				logger.ErrorF(ctx, "failed to delete Cloudflare record %s after managed record save failed: %v", created.ID, deleteErr)
			}
			return saveErr
		}
	}
	for i := range managed {
		if _, keep := targetNodeIDs[managed[i].NodeID]; keep {
			continue
		}
		if managed[i].CFRecordID != "" {
			if deleteErr := client.DeleteRecord(ctx, zoneID, managed[i].CFRecordID); deleteErr != nil && !isNotFoundError(deleteErr) {
				return deleteErr
			}
		}
		if deleteErr := repository.DeleteCFPointingManagedRecord(ctx, &managed[i]); deleteErr != nil {
			return deleteErr
		}
	}
	primary := targets[0]
	primaryRecord := managedByNode[primary.Node.ID]
	if primaryRecord == nil {
		records, listErr := repository.ListCFPointingManagedRecords(ctx, memberID)
		if listErr != nil || len(records) == 0 {
			return listErr
		}
		primaryRecord = &records[0]
	}
	return markMemberSynced(ctx, memberID, zoneID, primaryRecord.CFRecordID, primaryRecord.DesiredIP, primary.Node.ID)
}

func lowestPriorityAvailableNodes(nodes []repository.CFPointingGroupNodeContext) []repository.CFPointingGroupNodeContext {
	priority := 0
	found := false
	for _, node := range nodes {
		if node.Node.Status != "online" || net.ParseIP(strings.TrimSpace(node.Node.IP)).To4() == nil {
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
		if node.Node.Status == "online" && net.ParseIP(strings.TrimSpace(node.Node.IP)).To4() != nil && node.GroupNode.Priority == priority {
			available = append(available, node)
		}
	}
	return available
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
