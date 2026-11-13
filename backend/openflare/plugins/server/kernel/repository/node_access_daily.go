package repository

import (
	"context"
	"sort"
	"strconv"
	"time"

	"Wavelet/openflare/plugins/server/kernel/model"
	analyticsmodel "Wavelet/openflare/plugins/server/kernel/model/analytics"
	"Wavelet/openflare/plugins/server/kernel/repository/logstore"
)

func dailySummaryQuery(q model.OpenFlareAccessLogQuery) bool {
	return q.RemoteAddr == "" && q.Host == "" && len(q.Hosts) == 0 && q.Path == "" &&
		(q.Since.IsZero() || q.Since.Equal(q.Since.UTC().Truncate(24*time.Hour))) &&
		(q.Until.IsZero() || q.Until.Equal(q.Until.UTC().Truncate(24*time.Hour).Add(24*time.Hour-time.Nanosecond)))
}

func dailyRows(ctx context.Context, q model.OpenFlareAccessLogQuery) ([]analyticsmodel.NodeAccessDaily, error) {
	return logstore.SummarizeNodeAccessDaily(ctx, q)
}

func dailyTraffic(ctx context.Context, q model.OpenFlareAccessLogQuery) (model.OpenFlareAccessLogTrafficSummary, error) {
	rows, err := dailyRows(ctx, q)
	var out model.OpenFlareAccessLogTrafficSummary
	nodes := map[string]bool{}
	for _, row := range rows {
		out.RequestCount += int64(row.RequestCount)
		out.BytesSent += int64(row.BytesSent)
		out.RequestLength += int64(row.RequestLength)
		if row.StatusCode >= 500 {
			out.ErrorCount += int64(row.RequestCount)
		}
		if row.NodeID != "" {
			nodes[row.NodeID] = true
		}
	}
	out.NodeCount = int64(len(nodes))
	return out, err
}

func dailyStatusCounts(ctx context.Context, q model.OpenFlareAccessLogQuery, limit int) ([]model.OpenFlareAccessLogValueCount, error) {
	rows, err := dailyRows(ctx, q)
	if err != nil {
		return nil, err
	}
	counts := map[int32]int64{}
	for _, row := range rows {
		counts[row.StatusCode] += int64(row.RequestCount)
	}
	out := make([]model.OpenFlareAccessLogValueCount, 0, len(counts))
	for code, count := range counts {
		out = append(out, model.OpenFlareAccessLogValueCount{Value: strconv.Itoa(int(code)), Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Value < out[j].Value
		}
		return out[i].Count > out[j].Count
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func dailyNodes(ctx context.Context, q model.OpenFlareAccessLogQuery) ([]model.OpenFlareAccessLogNodeAggregate, error) {
	rows, err := dailyRows(ctx, q)
	if err != nil {
		return nil, err
	}
	byNode := map[string]model.OpenFlareAccessLogNodeAggregate{}
	for _, row := range rows {
		if row.NodeID == "" {
			continue
		}
		aggregate := byNode[row.NodeID]
		aggregate.NodeID = row.NodeID
		aggregate.RequestCount += int64(row.RequestCount)
		if row.StatusCode >= 500 {
			aggregate.ErrorCount += int64(row.RequestCount)
		}
		byNode[row.NodeID] = aggregate
	}
	out := make([]model.OpenFlareAccessLogNodeAggregate, 0, len(byNode))
	for _, aggregate := range byNode {
		out = append(out, aggregate)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NodeID < out[j].NodeID })
	return out, nil
}
