package cloudflare

import (
	"context"
	"errors"
	"time"

	"Wavelet/openflare/plugins/server/kernel/model"
	"Wavelet/openflare/plugins/server/kernel/repository"
	"Wavelet/pkg/logger"
)

func RunHealthMonitor(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := checkHealth(ctx); err != nil {
				logger.ErrorF(ctx, "[Cloudflare] health check failed: %v", err)
			}
		}
	}
}

func checkHealth(ctx context.Context) error {
	groups, err := repository.ListCFPointingGroups(ctx)
	if err != nil {
		return err
	}
	var failures []error
	for _, group := range groups {
		if !group.Enabled || group.TargetMode == targetModeCustom {
			continue
		}
		members, err := repository.ListCFPointingMembersByGroupID(ctx, group.ID)
		if err != nil {
			return err
		}
		for _, member := range members {
			if member.SyncStatus != model.CFMemberSyncOK {
				continue
			}
			state, err := repository.GetCFPointingMemberContext(ctx, member.ID)
			if err != nil {
				return err
			}
			targets, err := memberTargets(state)
			if err != nil {
				continue
			}
			records, err := repository.ListCFPointingManagedRecords(ctx, member.ID)
			if err != nil {
				return err
			}
			if !healthTargetsChanged(state, targets, records) {
				continue
			}
			claimed, err := repository.ClaimCFPointingMemberHealthSync(ctx, member.ID)
			if err != nil {
				return err
			}
			if !claimed {
				continue
			}
			if _, err := DispatchMemberSync(ctx, member.ID, "cloudflare_health_changed"); err != nil {
				if restoreErr := repository.UpdateCFPointingMemberColumns(ctx, member.ID, map[string]any{memberSyncStatusColumn: member.SyncStatus}); restoreErr != nil {
					logger.ErrorF(ctx, "[Cloudflare] restore member status failed: member_id=%d error=%v", member.ID, restoreErr)
				}
				failures = append(failures, err)
			}
		}
	}
	return errors.Join(failures...)
}

func healthTargetsChanged(state *repository.CFPointingMemberContext, targets []repository.CFPointingGroupNodeContext, records []model.CFPointingManagedRecord) bool {
	desired := make(map[managedKey]string)
	for _, target := range targets {
		for _, input := range memberRecordInputs(state, target) {
			desired[managedKey{nodeID: target.Node.ID, recordType: input.Type}] = input.Content
		}
	}
	if len(desired) != len(records) {
		return true
	}
	for _, record := range records {
		recordType := record.RecordType
		if recordType == "" {
			recordType = "A"
		}
		ip, ok := desired[managedKey{nodeID: record.NodeID, recordType: recordType}]
		if !ok || ip != record.DesiredIP {
			return true
		}
	}
	return false
}
