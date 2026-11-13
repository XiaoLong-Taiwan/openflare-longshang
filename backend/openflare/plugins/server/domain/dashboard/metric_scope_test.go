package dashboard

import (
	"testing"
	"time"

	"Wavelet/openflare/plugins/server/domain/observability"
	"Wavelet/openflare/plugins/server/kernel/model"
	"Wavelet/openflare/plugins/server/kernel/repository"
)

func TestSnapshotMetricsIncludeZero(t *testing.T) {
	view := &OverviewView{}
	row := &NodeHealth{}
	cpu, memory := applyNodeSnapshotMetrics(row, &model.OpenFlareMetricSnapshot{MemoryTotalBytes: 100}, view, 0, 0)
	cpu, memory = applyNodeSnapshotMetrics(row, &model.OpenFlareMetricSnapshot{CPUUsagePercent: 80, MemoryUsedBytes: 50, MemoryTotalBytes: 100}, view, cpu, memory)
	if cpu != 2 || memory != 2 || view.Capacity.AverageCPUUsagePercent/float64(cpu) != 40 || view.Capacity.AverageMemoryUsagePercent/float64(memory) != 25 {
		t.Fatalf("applyNodeSnapshotMetrics(zero, loaded) = %+v, cpu=%d memory=%d; want averages 40/25 with two samples", view.Capacity, cpu, memory)
	}
	cpu, memory = applyNodeSnapshotMetrics(row, nil, view, cpu, memory)
	if cpu != 2 || memory != 2 {
		t.Errorf("applyNodeSnapshotMetrics(nil) counts = %d/%d, want 2/2", cpu, memory)
	}
}

func TestFallbackTrafficRates(t *testing.T) {
	traffic := &Traffic{EstimatedQPS: 99}
	points := []observability.TrafficTrendPoint{{RequestCount: 43200, ErrorCount: 432, Status4xxCount: 4320, Status5xxCount: 432, UniqueVisitorCount: 10}, {RequestCount: 43200, ErrorCount: 432, Status4xxCount: 4320, Status5xxCount: 432, UniqueVisitorCount: 10}}
	applyTrafficTotalsFromTrend(traffic, points)
	applyTrafficRates(traffic, points)
	if traffic.AverageQPS != 1 || traffic.EstimatedQPS != 1 || traffic.Status4xxRate != 10 || traffic.Status5xxRate != 1 || traffic.UniqueVisitors != 0 || traffic.ReportedNodes != 0 {
		t.Errorf("fallback traffic = %+v, want QPS 1, rates 10/1, no fabricated UV/node count", traffic)
	}
	applyTrafficTotalsFromTrend(traffic, nil)
	applyTrafficRates(traffic, nil)
	if traffic.AverageQPS != 0 || traffic.Status4xxRate != 0 || traffic.Status5xxRate != 0 {
		t.Errorf("empty fallback traffic = %+v, want zero QPS and rates", traffic)
	}
}

func TestSnapshotCoverage(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 30, 0, 0, time.UTC)
	capacity := &Capacity{FreshnessThresholdSeconds: 120}
	applySnapshotCoverage(capacity, nil, now)
	applySnapshotCoverage(capacity, &model.OpenFlareMetricSnapshot{CapturedAt: now.Add(-120 * time.Second)}, now)
	applySnapshotCoverage(capacity, &model.OpenFlareMetricSnapshot{CapturedAt: now.Add(-121 * time.Second)}, now)
	if capacity.ReportedNodes != 2 || capacity.FreshNodes != 1 || capacity.StaleNodes != 1 || capacity.MissingNodes != 1 || capacity.LatestSnapshotAt == nil || !capacity.LatestSnapshotAt.Equal(now.Add(-120*time.Second)) {
		t.Errorf("applySnapshotCoverage() = %+v, want reported/fresh/stale/missing 2/1/1/1 and latest at threshold", capacity)
	}
}

func TestRollingTrafficAndNetworkBoundaries(t *testing.T) {
	cleanup := setupDashboardTestDB(t)
	t.Cleanup(cleanup)
	ctx := t.Context()
	now := time.Date(2026, 11, 13, 12, 30, 0, 0, time.UTC)
	since := now.Add(-24 * time.Hour)
	logs := []*model.OpenFlareAccessLog{
		{NodeID: "edge", LoggedAt: since.Add(-time.Second), StatusCode: 500, BytesSent: 1000},
		{NodeID: "edge", LoggedAt: since.Add(time.Second), StatusCode: 404, BytesSent: 10, RequestLength: 5, RemoteAddr: "1.1.1.1"},
		{NodeID: "edge", LoggedAt: now.Add(-time.Second), StatusCode: 503, BytesSent: 20, RequestLength: 7, RemoteAddr: "1.1.1.1"},
		{NodeID: "edge", LoggedAt: now.Add(time.Second), StatusCode: 500, BytesSent: 1000},
	}
	if err := repository.InsertOpenFlareAccessLogsBatch(ctx, logs); err != nil {
		t.Fatal(err)
	}
	trends := observability.BuildNodeTrends(ctx, now, "", nil)
	traffic := &Traffic{}
	applyTrafficTotalsFromTrend(traffic, trends.Traffic24h)
	applyTrafficRates(traffic, trends.Traffic24h)
	if traffic.RequestCount != 2 || traffic.Status4xxRate != 50 || traffic.Status5xxRate != 50 {
		t.Errorf("rolling traffic = %+v, want two requests and 50/50 rates", traffic)
	}
	if trends.Traffic24h[0].RequestCount != 1 || trends.Traffic24h[24].RequestCount != 1 || trends.Network24h[0].BytesProvided != 10 || trends.Network24h[24].BytesProvided != 20 || trends.Network24h[0].BytesReceived != 5 || trends.Network24h[24].BytesReceived != 7 {
		t.Errorf("rolling boundaries traffic=%+v network=%+v, want only first/last in-window records", trends.Traffic24h, trends.Network24h)
	}
	if traffic.RequestCount != 2 || traffic.Status4xxRate != 50 || traffic.Status5xxRate != 50 {
		t.Errorf("partial-hour requests outside rolling window affected rates: %+v", traffic)
	}
	for i := range trends.Traffic24h {
		if !trends.Traffic24h[i].BucketStartedAt.Equal(trends.Network24h[i].BucketStartedAt) || !trends.Traffic24h[i].BucketStartedAt.Equal(trends.Capacity24h[i].BucketStartedAt) || !trends.Traffic24h[i].BucketStartedAt.Equal(trends.DiskIO24h[i].BucketStartedAt) {
			t.Errorf("trend bucket %d timestamps differ", i)
		}
	}
}
