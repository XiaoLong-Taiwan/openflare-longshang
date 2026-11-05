package repository

import (
	"Wavelet/openflare/plugins/server/kernel/model"
	analyticsmodel "Wavelet/openflare/plugins/server/kernel/model/analytics"
	"testing"
	"time"
)

func TestDailyQueriesSurviveDetailDeletion(t *testing.T) {
	ctx, cleanup := setupOpenFlareAccessLogTestEnvironment(t)
	t.Cleanup(cleanup)
	day := time.Now().UTC().Truncate(24 * time.Hour)
	rows := []analyticsmodel.NodeAccessLog{
		{ID: 1, NodeID: "a", LoggedAt: day, StatusCode: 200, BytesSent: 100, RequestLength: 10},
		{ID: 2, NodeID: "a", LoggedAt: day.Add(time.Hour), StatusCode: 404, BytesSent: 200, RequestLength: 20},
		{ID: 3, NodeID: "b", LoggedAt: day.Add(2 * time.Hour), StatusCode: 503, BytesSent: 300, RequestLength: 30},
	}
	if err := DB(ctx).Create(rows).Error; err != nil {
		t.Fatal(err)
	}
	q := model.OpenFlareAccessLogQuery{Since: day, Until: day.Add(24*time.Hour - time.Nanosecond)}
	before, err := TrafficSummaryOpenFlareAccessLogs(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DeleteOpenFlareAccessLogsBefore(ctx, day.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	after, err := TrafficSummaryOpenFlareAccessLogs(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	if before != after || after.RequestCount != 3 || after.ErrorCount != 1 || after.BytesSent != 600 || after.RequestLength != 60 || after.NodeCount != 2 {
		t.Errorf("daily traffic before=%+v after=%+v, want stable 3 requests/1 error/600 sent/60 received/2 nodes", before, after)
	}
	status, err := ValueCountsOpenFlareAccessLogs(ctx, q, "status_code", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(status) != 3 {
		t.Errorf("status groups = %v, want 3 groups", status)
	}
	nodes, err := NodeAggregatesOpenFlareAccessLogs(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 || nodes[0].RequestCount != 2 || nodes[1].ErrorCount != 1 {
		t.Errorf("daily nodes = %+v, want a:2/0 b:1/1", nodes)
	}
}
