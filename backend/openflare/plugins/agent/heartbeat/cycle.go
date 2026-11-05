// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package heartbeat implements the periodic heartbeat cycle executed by the agent,
// including payload preparation, config sync, WAF IP group application, and observability buffering.
package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"Wavelet/openflare/plugins/agent/config"
	"Wavelet/openflare/plugins/agent/observability"
	"Wavelet/openflare/plugins/agent/protocol"
	"Wavelet/openflare/plugins/agent/state"
	"Wavelet/openflare/plugins/agent/updater"
	edgeheartbeat "Wavelet/openflare/share/edge/heartbeat"
	"Wavelet/openflare/share/edge/nodeip"
)

// SyncService is the interface used by Cycle to sync active configuration and WAF IP groups.
type SyncService interface {
	SyncOnStartup(ctx context.Context, target *protocol.ActiveConfigMeta) error
	SyncOnce(ctx context.Context, target *protocol.ActiveConfigMeta) error
	WAFIPGroupChecksums() (map[string]string, error)
	ApplyWAFIPGroups(ctx context.Context, groups []protocol.WAFIPGroup) error
}

// SettingsApplier is the interface used by Cycle to apply agent settings received from the server.
type SettingsApplier interface {
	Apply(settings *protocol.AgentSettings) (intervalChanged bool)
	RestartOpenrestyIfNeeded(ctx context.Context)
}

// Cycle holds the dependencies required to execute a single agent heartbeat cycle.
type Cycle struct {
	Config              *config.Config
	StateStore          *state.Store
	ObservabilityBuffer *state.ObservabilityBufferStore
	Heartbeat           API
	Sync                SyncService
	Updater             *updater.Service
	RecordSyncError     func(err error)
	lastCollection      time.Time
	nextAttempt         time.Time
	retryDelay          time.Duration
	byteLimit           int
	logLimit            int
	bufferLimit         int
	pendingRecords      []state.ObservabilityBufferRecord
	prepareError        error
}

// Perform executes one complete heartbeat cycle: sends the heartbeat, syncs config, and applies settings.
func (c *Cycle) Perform(ctx context.Context, nodeID string, startup bool, settings SettingsApplier) (bool, error) {
	payload, ackWindows := c.PrepareHeartbeatPayload(ctx, nodeID)
	if payload.SchemaVersion == 0 {
		return false, c.prepareError
	}
	heartbeatResult, err := c.Heartbeat.Heartbeat(ctx, payload)
	if err != nil {
		var reportErr interface {
			error
			StatusCode() int
			RetryAfter() time.Duration
		}
		if errors.As(err, &reportErr) {
			switch reportErr.StatusCode() {
			case http.StatusRequestEntityTooLarge:
				c.byteLimit = max(c.byteLimit/2, 64*1024)
				c.logLimit = max(c.logLimit/2, 1)
				c.bufferLimit = max(c.bufferLimit/2, 1)
				if c.byteLimit == 64*1024 && c.logLimit == 1 && c.bufferLimit == 1 {
					c.nextAttempt = time.Now().Add(time.Minute)
					slog.Error("agent report rejected at minimum batch size", "byte_limit", c.byteLimit)
				}
			case http.StatusTooManyRequests:
				c.retryDelay = min(max(c.retryDelay*2, 3*time.Second), time.Minute)
				c.nextAttempt = time.Now().Add(max(c.retryDelay, reportErr.RetryAfter()))
			}
		}
		return false, err
	}
	c.retryDelay = 0
	c.AckObservabilityWindows(ackWindows)
	if heartbeatResult == nil {
		heartbeatResult = &protocol.HeartbeatResult{}
	}
	mode := "periodic"
	if startup {
		mode = "startup"
	}
	slog.Debug("agent heartbeat succeeded", "mode", mode, "node_id", nodeID)

	var changed bool
	if settings != nil {
		changed = settings.Apply(heartbeatResult.AgentSettings)
	}
	c.ApplyWAFIPGroups(ctx, heartbeatResult.WAFIPGroups)
	if startup {
		if c.Sync != nil {
			if err = c.Sync.SyncOnStartup(ctx, heartbeatResult.ActiveConfig); err != nil {
				c.recordSyncError(err)
				slog.Error("agent startup sync failed", "error", err)
			} else {
				slog.Debug("agent startup sync completed")
			}
		}
	} else if c.Sync != nil {
		if err = c.Sync.SyncOnce(ctx, heartbeatResult.ActiveConfig); err != nil {
			c.recordSyncError(err)
			slog.Error("agent sync failed", "error", err)
		}
	}
	if settings != nil {
		settings.RestartOpenrestyIfNeeded(ctx)
	}
	edgeheartbeat.TryAutoUpdate(ctx, c.Updater, agentSettingsToAutoUpdate(heartbeatResult.AgentSettings), "agent")
	return changed, nil
}

// NodePayload builds and returns the full NodePayload to be sent in a heartbeat request.
func (c *Cycle) NodePayload(ctx context.Context, nodeID string) protocol.NodePayload {
	snapshot, _ := c.StateStore.Load()
	openrestyStatus := strings.TrimSpace(snapshot.OpenrestyStatus)
	if openrestyStatus == "" {
		openrestyStatus = protocol.OpenrestyStatusUnknown
	}
	profile := observability.BuildProfile(c.Config, c.StateStore)
	edgeSnapshot := observability.CollectEdgeHealth(ctx, c.Config)
	accessLogs := observability.CollectAccessLogs(c.Config, c.StateStore)
	metricSnapshot := observability.BuildSnapshot(c.Config, c.StateStore)
	edgeHealth := observability.BuildEdgeHealth(edgeSnapshot, openrestyStatus, snapshot.OpenrestyMessage)
	healthEvents := observability.BuildHealthEvents(snapshot)

	ip := c.Config.NodeIP
	if !c.Config.NodeIPConfigured {
		ip = nodeip.DetectWithContext(ctx)
	}

	payload := protocol.NodePayload{
		SchemaVersion:    2,
		NodeID:           nodeID,
		Name:             c.Config.NodeName,
		IP:               ip,
		Version:          c.Config.Version,
		ExtVersion:       c.Config.ExtVersion,
		CurrentVersion:   snapshot.CurrentVersion,
		LastError:        snapshot.LastError,
		OpenrestyStatus:  openrestyStatus,
		OpenrestyMessage: snapshot.OpenrestyMessage,
		Profile:          profile,
		HostMetrics:      metricSnapshot,
		EdgeHealth:       edgeHealth,
		AccessLogs:       accessLogs,
		HealthEvents:     healthEvents,
	}
	if c.Sync != nil {
		checksums, err := c.Sync.WAFIPGroupChecksums()
		if err != nil {
			slog.Debug("load local waf ip group checksums failed", "error", err)
		} else if len(checksums) > 0 {
			payload.WAFIPGroupChecksums = checksums
		}
	}
	return payload
}

// PrepareHeartbeatPayload constructs the heartbeat payload with buffered observability records and returns the window timestamps to acknowledge.
func (c *Cycle) PrepareHeartbeatPayload(ctx context.Context, nodeID string) (payload protocol.NodePayload, ack []int64) {
	c.prepareError = nil
	defer func() {
		if payload.SchemaVersion == 0 {
			return
		}
		encoded, err := json.Marshal(payload)
		oversizedRow := false
		for _, row := range payload.AccessLogs {
			data, rowErr := json.Marshal(row)
			oversizedRow = oversizedRow || rowErr != nil || len(data) > 16*1024
		}
		rows := len(payload.AccessLogs)
		for _, record := range payload.Buffered {
			rows += len(record.AccessLogs)
		}
		if err != nil || oversizedRow || len(encoded) > c.byteLimit || rows > c.logLimit || len(payload.Buffered) > c.bufferLimit || len(payload.HealthEvents) > 100 {
			c.prepareError = fmt.Errorf("agent report exceeds batch limits: bytes=%d limit=%d logs=%d encoding_error=%v", len(encoded), c.byteLimit, rows, err)
			slog.Error("agent report preparation failed", "error", c.prepareError)
			payload = protocol.NodePayload{}
			ack = nil
			c.pendingRecords = nil
		}
		if c.prepareError != nil && payload.SchemaVersion != 0 {
			slog.Error("agent report contains retained unsendable data", "error", c.prepareError)
		}
	}()
	if time.Now().Before(c.nextAttempt) || time.Since(c.lastCollection) < max(c.Config.HeartbeatInterval.Duration(), 3*time.Second) {
		return protocol.NodePayload{}, nil
	}
	c.lastCollection = time.Now()
	if c.byteLimit == 0 {
		c.byteLimit = 1024 * 1024
		c.logLimit = 1000
		c.bufferLimit = 4
	}
	payload = c.NodePayload(ctx, nodeID)
	c.pendingRecords = nil
	if c.ObservabilityBuffer == nil || (payload.HostMetrics == nil && payload.EdgeHealth == nil && len(payload.AccessLogs) == 0) {
		return payload, nil
	}
	now := time.Now().UTC()
	retainAfterUnix := now.Add(-time.Duration(c.Config.ObservabilityReplayMinutes) * time.Minute).Unix()
	windowStartedAtUnix := state.ObservabilityWindowStartedAt(payload.HostMetrics, payload.EdgeHealth)
	if windowStartedAtUnix <= 0 {
		return payload, nil
	}

	record := state.ObservabilityBufferRecord{
		WindowStartedAtUnix: windowStartedAtUnix,
		HostMetrics:         payload.HostMetrics,
		EdgeHealth:          payload.EdgeHealth,
		AccessLogs:          payload.AccessLogs,
		QueuedAtUnix:        now.Unix(),
	}
	if err := c.ObservabilityBuffer.Upsert(record, retainAfterUnix); err != nil {
		slog.Error("upsert observability buffer failed", "error", err)
		return payload, nil
	}
	records, err := c.ObservabilityBuffer.Replayable(0, retainAfterUnix)
	if err != nil {
		c.prepareError = fmt.Errorf("load replayable observability buffer: %w", err)
		slog.Error("load replayable observability buffer failed", "error", err)
		return protocol.NodePayload{}, nil
	}
	payload.HostMetrics = nil
	payload.EdgeHealth = nil
	payload.AccessLogs = nil
	ackWindows := make([]int64, 0, len(records)+1)
	buffered := make([]protocol.BufferedObservabilityRecord, 0, len(records))
	for _, item := range records {
		if item.WindowStartedAtUnix <= 0 {
			continue
		}
		if len(buffered) >= c.bufferLimit {
			break
		}
		selected := item
		selected.AccessLogs = nil
		buffered = append(buffered, protocol.BufferedObservabilityRecord{
			CapturedAtUnix: item.WindowStartedAtUnix,
			HostMetrics:    item.HostMetrics,
			EdgeHealth:     item.EdgeHealth,
		})
		payload.Buffered = buffered
		encoded, marshalErr := json.Marshal(payload)
		if marshalErr != nil || len(encoded) > c.byteLimit {
			c.prepareError = errors.New("buffered observability metadata exceeds report byte limit")
			slog.Error("agent buffered record cannot fit report", "window", item.WindowStartedAtUnix, "byte_limit", c.byteLimit)
			buffered = buffered[:len(buffered)-1]
			break
		}
		batchBytes := len(encoded)
		rowCount := len(payload.AccessLogs)
		for _, previous := range buffered {
			rowCount += len(previous.AccessLogs)
		}
		skipped := 0
		for _, row := range item.AccessLogs {
			rowJSON, rowErr := json.Marshal(row)
			if rowErr != nil || len(rowJSON) > 16*1024 {
				skipped++
				c.prepareError = errors.New("buffered access log exceeds 16 KiB single record limit")
				slog.Warn("agent buffered access log exceeds single record limit; retained", "window", item.WindowStartedAtUnix, "bytes", len(rowJSON))
				continue
			}
			if rowCount >= c.logLimit {
				break
			}
			index := len(buffered) - 1
			addedBytes := len(rowJSON) + 1
			if len(buffered[index].AccessLogs) == 0 {
				addedBytes = len(rowJSON) + len(`,"access_logs":[]`)
			}
			if batchBytes+addedBytes > c.byteLimit {
				if rowCount == 0 {
					c.prepareError = errors.New("single access log cannot fit report byte limit")
					slog.Error("agent access log cannot fit report", "byte_limit", c.byteLimit)
				}
				break
			}
			buffered[index].AccessLogs = append(buffered[index].AccessLogs, row)
			batchBytes += addedBytes
			rowCount++
		}
		selected.AccessLogs = buffered[len(buffered)-1].AccessLogs
		c.pendingRecords = append(c.pendingRecords, selected)
		ackWindows = append(ackWindows, item.WindowStartedAtUnix)
		if len(selected.AccessLogs)+skipped < len(item.AccessLogs) {
			break
		}
	}
	payload.Buffered = buffered
	return payload, ackWindows
}

// AckObservabilityWindows acknowledges the given observability window timestamps in the buffer store.
func (c *Cycle) AckObservabilityWindows(windowStartedAtUnix []int64) {
	if c.ObservabilityBuffer == nil || len(windowStartedAtUnix) == 0 {
		return
	}
	retainAfterUnix := time.Now().UTC().Add(-time.Duration(c.Config.ObservabilityReplayMinutes) * time.Minute).Unix()
	if err := c.ObservabilityBuffer.AckRecords(c.pendingRecords, retainAfterUnix); err != nil {
		slog.Error("ack observability buffer failed", "error", err)
	}
}

// ApplyWAFIPGroups applies the WAF IP groups received from the server via the SyncService.
func (c *Cycle) ApplyWAFIPGroups(ctx context.Context, groups []protocol.WAFIPGroup) {
	if len(groups) == 0 || c.Sync == nil {
		return
	}
	if err := c.Sync.ApplyWAFIPGroups(ctx, groups); err != nil {
		c.recordSyncError(err)
		slog.Error("agent apply waf ip groups failed", "error", err)
	}
}

func (c *Cycle) recordSyncError(err error) {
	if c.RecordSyncError != nil {
		c.RecordSyncError(err)
	}
}

// AgentSettingsToAutoUpdate converts AgentSettings to an AutoUpdateSettings value used by the edge heartbeat updater.
func AgentSettingsToAutoUpdate(settings *protocol.AgentSettings) *edgeheartbeat.AutoUpdateSettings {
	if settings == nil {
		return nil
	}
	return &edgeheartbeat.AutoUpdateSettings{
		AutoUpdate:    settings.AutoUpdate,
		UpdateNow:     settings.UpdateNow,
		UpdateRepo:    settings.UpdateRepo,
		UpdateChannel: settings.UpdateChannel,
		UpdateTag:     settings.UpdateTag,
	}
}

func agentSettingsToAutoUpdate(settings *protocol.AgentSettings) *edgeheartbeat.AutoUpdateSettings {
	return AgentSettingsToAutoUpdate(settings)
}
