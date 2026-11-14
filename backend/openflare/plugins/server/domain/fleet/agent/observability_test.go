// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"testing"
	"time"

	"Wavelet/openflare/plugins/server/kernel/model"
	analyticsmodel "Wavelet/openflare/plugins/server/kernel/model/analytics"
	"Wavelet/openflare/plugins/server/kernel/repository/logstore"
	"Wavelet/pkg/idgen"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestBuildNodeAccessLogRecordsPreservesBytesSent(t *testing.T) {
	reportedAt := time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)

	records, err := buildNodeAccessLogRecords(context.Background(), "node-a", []NodeAccessLog{
		{
			LoggedAtUnix: reportedAt.Unix(),
			RemoteAddr:   "203.0.113.10",
			Host:         "api.example.com",
			Path:         "/v1/ping",
			StatusCode:   200,
			BytesSent:    4096,
		},
	}, nil, reportedAt)
	if err != nil {
		t.Fatalf("buildNodeAccessLogRecords() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected one access log record, got %d", len(records))
	}
	if records[0].BytesSent != 4096 {
		t.Fatalf("BytesSent = %d, want 4096", records[0].BytesSent)
	}
}

func TestPersistHeartbeatObservabilityWritesDurablyWithoutQueueHooks(t *testing.T) {
	if err := idgen.Init(1); err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(sqlite.Open("file:agent-heartbeat-durable?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&analyticsmodel.NodeMetricSnapshot{}, &analyticsmodel.NodeEdgeHealth{}, &analyticsmodel.NodeAccessLog{}, &analyticsmodel.NodeAccessDaily{}, &analyticsmodel.NodeAccessQuota{}, &analyticsmodel.NodeAccessBatch{}, &analyticsmodel.NodeAccessCheckpoint{}, &analyticsmodel.NodeAccessOutbox{}); err != nil {
		t.Fatal(err)
	}
	logstore.ResetForTest()
	logstore.SetDBResolver(func(context.Context) *gorm.DB { return db })
	logstore.SetConfigReader(func(_ context.Context, key string) (string, error) {
		if key == model.ConfigKeyLogDatabase {
			return "sqlite", nil
		}
		return "", nil
	})
	logstore.SetObservabilityHooks(logstore.ObservabilityHooks{
		QueueMetricSnapshot: func(analyticsmodel.NodeMetricSnapshot) bool { return false },
		QueueEdgeHealth:     func(analyticsmodel.NodeEdgeHealth) bool { return false },
	})
	t.Cleanup(func() {
		logstore.SetObservabilityHooks(logstore.ObservabilityHooks{})
		logstore.SetDBResolver(nil)
		logstore.ResetForTest()
	})
	if err := persistHeartbeatObservability(context.Background(), "node-a", NodePayload{
		HostMetrics: &NodeMetricSnapshot{CPUUsagePercent: 12.5},
		EdgeHealth:  &NodeEdgeHealth{Status: "healthy"},
	}, time.Now().UTC()); err != nil {
		t.Fatalf("persistHeartbeatObservability() error = %v", err)
	}
	var metricCount, edgeCount int64
	if err := db.Model(&analyticsmodel.NodeMetricSnapshot{}).Count(&metricCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&analyticsmodel.NodeEdgeHealth{}).Count(&edgeCount).Error; err != nil {
		t.Fatal(err)
	}
	if metricCount != 1 || edgeCount != 1 {
		t.Fatalf("durable counts metric=%d edge=%d, want 1/1", metricCount, edgeCount)
	}
}

func TestPersistHeartbeatObservabilityFailsWhenDurableWriteFails(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:agent-heartbeat-durable-failure?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	logstore.ResetForTest()
	logstore.SetDBResolver(func(context.Context) *gorm.DB { return db })
	logstore.SetConfigReader(func(_ context.Context, key string) (string, error) {
		if key == model.ConfigKeyLogDatabase {
			return "sqlite", nil
		}
		return "", nil
	})
	t.Cleanup(func() { logstore.SetDBResolver(nil); logstore.ResetForTest() })
	err = persistHeartbeatObservability(context.Background(), "node-a", NodePayload{
		HostMetrics: &NodeMetricSnapshot{CPUUsagePercent: 12.5},
	}, time.Now().UTC())
	if err == nil {
		t.Fatalf("persistHeartbeatObservability() error = %v, want schema/write error", err)
	}
}
