package observability

import (
	"testing"
	"time"

	"Wavelet/openflare/plugins/server/kernel/model"
)

func TestRollingTrendIncludesBothPartialHoursAndZeroCPU(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 30, 0, 0, time.UTC)
	since := now.Add(-24 * time.Hour)
	points := BuildCapacityTrendPoints(now, []*model.OpenFlareMetricSnapshot{
		{NodeID: "a", CapturedAt: since.Add(-time.Second), CPUUsagePercent: 100},
		{NodeID: "a", CapturedAt: since, CPUUsagePercent: 0, MemoryTotalBytes: 100},
		{NodeID: "b", CapturedAt: since.Add(time.Second), CPUUsagePercent: 80, MemoryUsedBytes: 50, MemoryTotalBytes: 100},
		{NodeID: "a", CapturedAt: now, CPUUsagePercent: 20},
		{NodeID: "a", CapturedAt: now.Add(time.Second), CPUUsagePercent: 100},
	})
	if len(points) != 25 || !points[0].BucketStartedAt.Equal(since.Truncate(time.Hour)) || !points[24].BucketStartedAt.Equal(now.Truncate(time.Hour)) {
		t.Fatalf("BuildCapacityTrendPoints() boundaries = %+v, want 25 buckets from %v to %v", points, since.Truncate(time.Hour), now.Truncate(time.Hour))
	}
	if points[0].AverageCPUUsagePercent != 40 || points[0].AverageMemoryUsagePercent != 25 || points[24].AverageCPUUsagePercent != 20 {
		t.Errorf("BuildCapacityTrendPoints() first/last = %+v/%+v, want CPU 40/20 and first memory 25", points[0], points[24])
	}
}
