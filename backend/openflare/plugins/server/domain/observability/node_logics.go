// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package observability

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"Wavelet/openflare/plugins/server/kernel/repository"
	"Wavelet/pkg/cache/ram"
	"Wavelet/pkg/util"
	"Wavelet/plugins/infra/cache"

	"Wavelet/openflare/plugins/server/kernel/model"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"gorm.io/gorm"
)

const (
	defaultObservabilityWindow           = 24 * time.Hour
	defaultObservabilityLimit            = 120
	maxObservabilityLimit                = 500
	defaultTrafficDistributionLimit      = 8
	nodeObservabilityCacheTTL            = 15 * time.Second
	nodeObservabilityCacheCapacity       = 1024
	nodeObservabilityInvalidationChannel = "wavelet:observability:node_invalidation"
)

var nodeObservabilityCache = newNodeObservabilityCache()

var (
	nodeObservabilityMeter       = otel.Meter("wavelet/openflare/observability")
	nodeObservabilityCacheHits   = mustInt64Counter(nodeObservabilityMeter, "node_observability_cache_hits")
	nodeObservabilityCacheMisses = mustInt64Counter(nodeObservabilityMeter, "node_observability_cache_misses")
	nodeObservabilityCacheItems  = mustInt64Gauge(nodeObservabilityMeter, "node_observability_cache_items")
	nodeObservabilityQueryTime   = mustFloat64Histogram(nodeObservabilityMeter, "node_observability_query_duration_ms")
)

type nodeObservabilityResponseCache struct {
	views      *ram.Cache[string, *NodeView]
	mu         sync.Mutex
	generation uint64
}

func newNodeObservabilityCache() *nodeObservabilityResponseCache {
	return &nodeObservabilityResponseCache{views: ram.MustNew[string, *NodeView](ram.Options{
		MaximumSize: nodeObservabilityCacheCapacity,
		TTL:         nodeObservabilityCacheTTL,
	})}
}

func mustInt64Counter(m metric.Meter, name string) metric.Int64Counter {
	instrument, _ := m.Int64Counter(name)
	return instrument
}

func mustInt64Gauge(m metric.Meter, name string) metric.Int64ObservableGauge {
	instrument, _ := m.Int64ObservableGauge(name, metric.WithInt64Callback(func(_ context.Context, observer metric.Int64Observer) error {
		observer.Observe(int64(nodeObservabilityCache.views.EstimatedSize()))
		return nil
	}))
	return instrument
}

// StartNodeCacheInvalidation subscribes to cache invalidations from other processes.
func StartNodeCacheInvalidation(ctx context.Context, client redis.UniversalClient) func() error {
	if client == nil {
		return func() error { return nil }
	}
	ctx, cancel := context.WithCancel(ctx)
	pubsub := client.Subscribe(ctx, nodeObservabilityInvalidationChannel)
	util.Go(func() {
		for message := range pubsub.Channel() {
			nodeObservabilityCache.invalidateNode(message.Payload)
		}
	})
	return func() error {
		cancel()
		return pubsub.Close()
	}
}

func invalidateNodeObservability(ctx context.Context, nodeID string) {
	nodeObservabilityCache.invalidateNode(nodeID)
	if cache.Redis != nil {
		_ = cache.Redis.Publish(ctx, nodeObservabilityInvalidationChannel, nodeID).Err()
	}
}

func mustFloat64Histogram(m metric.Meter, name string) metric.Float64Histogram {
	instrument, _ := m.Float64Histogram(name)
	return instrument
}

// NodeQuery filters node observability data.
type NodeQuery struct {
	Hours int `json:"hours"`
	Limit int `json:"limit"`
}

// NodeAnalytics groups node observability analytics.
type NodeAnalytics struct {
	Traffic       *TrafficWindowSummary `json:"traffic"`
	Distributions TrafficDistributions  `json:"distributions"`
	Health        HealthSummary         `json:"health"`
}

// NodeTrends groups node observability trend series.
type NodeTrends struct {
	Traffic24h  []TrafficTrendPoint  `json:"traffic_24h"`
	Capacity24h []CapacityTrendPoint `json:"capacity_24h"`
	Network24h  []NetworkTrendPoint  `json:"network_24h"`
	DiskIO24h   []DiskIOTrendPoint   `json:"disk_io_24h"`
}

// RelayDashboardSnapshot summarizes tunnel relay status.
type RelayDashboardSnapshot struct {
	TotalProxies     int              `json:"total_proxies"`
	OnlineProxies    int              `json:"online_proxies"`
	OfflineProxies   int              `json:"offline_proxies"`
	Proxies          []RelayProxyStat `json:"proxies"`
	TotalConnections int              `json:"total_connections"`
	ClientCounts     int              `json:"client_counts"`
}

// RelayProxyStat is a single relay proxy entry.
type RelayProxyStat struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	ClientVersion string `json:"client_version"`
	LastStartTime string `json:"last_start_time"`
	LastCloseTime string `json:"last_close_time"`
	ClientAddr    string `json:"client_addr"`
}

// NodeView is the node observability API response.
type NodeView struct {
	NodeID          string                            `json:"node_id"`
	Profile         *model.OpenFlareNodeSystemProfile `json:"profile"`
	MetricSnapshots []*NodeMetricSnapshotView         `json:"metric_snapshots"`
	HealthEvents    []*model.OpenFlareHealthEvent     `json:"health_events"`
	Analytics       NodeAnalytics                     `json:"analytics"`
	Trends          NodeTrends                        `json:"trends"`
	RelayDashboard  *RelayDashboardSnapshot           `json:"relay_dashboard,omitempty"`
}

// HealthEventCleanupResult reports health event cleanup outcome.
type HealthEventCleanupResult struct {
	NodeID       string `json:"node_id"`
	DeletedCount int64  `json:"deleted_count"`
}

// GetNodeObservability returns observability details for a node.
func GetNodeObservability(ctx context.Context, id uint, query NodeQuery) (*NodeView, error) {
	node, err := repository.GetOpenFlareNodeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(node.NodeID) == "" {
		return nil, gorm.ErrRecordNotFound
	}
	hours := normalizeObservabilityHours(query.Hours)
	limit := normalizeObservabilityLimit(query.Limit)
	key := nodeObservabilityCacheKey(node.NodeID, hours, limit, node.NodeType)
	return nodeObservabilityCache.getOrLoad(ctx, key, func() (*NodeView, error) {
		return queryNodeObservability(ctx, node, hours, limit)
	})
}

func (c *nodeObservabilityResponseCache) getOrLoad(ctx context.Context, key string, loader func() (*NodeView, error)) (*NodeView, error) {
	c.mu.Lock()
	if view, ok := c.views.GetIfPresent(key); ok {
		c.mu.Unlock()
		nodeObservabilityCacheHits.Add(ctx, 1)
		return view, nil
	}
	generation := c.generation
	c.mu.Unlock()
	nodeObservabilityCacheMisses.Add(ctx, 1)
	started := time.Now()
	view, err := loader()
	nodeObservabilityQueryTime.Record(ctx, float64(time.Since(started).Microseconds())/1000)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	if generation == c.generation {
		c.views.Set(key, view)
	}
	c.mu.Unlock()
	return view, nil
}

func (c *nodeObservabilityResponseCache) invalidateNode(nodeID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.generation++
	prefix := nodeID + ":"
	for key := range c.views.Keys() {
		if strings.HasPrefix(key, prefix) {
			c.views.Invalidate(key)
		}
	}
	c.views.CleanUp()
}

func queryNodeObservability(ctx context.Context, node *model.OpenFlareNode, hours, limit int) (*NodeView, error) {
	now := time.Now()
	since := now.Add(-time.Duration(hours) * time.Hour)

	profile, err := repository.GetOpenFlareNodeSystemProfile(ctx, node.NodeID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		profile = nil
	}

	snapshots, err := repository.ListOpenFlareMetricSnapshotsSince(ctx, node.NodeID, since, limit)
	if err != nil {
		return nil, err
	}
	edgeHealth, err := repository.ListOpenFlareEdgeHealth(ctx, node.NodeID, since, limit)
	if err != nil {
		return nil, err
	}
	accessLogRegions, err := repository.ListOpenFlareAccessLogRegionCounts(ctx, node.NodeID, since, defaultTrafficDistributionLimit)
	if err != nil {
		return nil, err
	}
	events, err := repository.ListOpenFlareHealthEvents(ctx, node.NodeID, false, limit)
	if err != nil {
		return nil, err
	}
	distributions := buildTrafficDistributionsForNode(
		ctx, node.NodeID, since, now, defaultTrafficDistributionLimit, accessLogRegions,
	)
	trafficSummary := buildTrafficWindowSummaryFromAccessLogs(ctx, node.NodeID, since, now)
	view := &NodeView{
		NodeID:          node.NodeID,
		Profile:         profile,
		MetricSnapshots: BuildMetricSnapshotViews(snapshots, edgeHealth),
		HealthEvents:    events,
		Analytics: NodeAnalytics{
			Traffic:       trafficSummary,
			Distributions: distributions,
			Health:        buildHealthSummary(latestMetricSnapshot(snapshots), trafficSummary, events),
		},
		Trends: BuildNodeTrends(ctx, now, node.NodeID, snapshots),
	}
	if node.NodeType == "tunnel_relay" {
		frpsObs, frpsErr := repository.ListOpenFlareNodeObservationFrps(ctx, node.NodeID, time.Time{}, 1)
		if frpsErr != nil {
			return nil, frpsErr
		}
		var latestFrps *model.OpenFlareNodeObservationFrps
		if len(frpsObs) > 0 {
			latestFrps = frpsObs[0]
		}
		view.RelayDashboard = buildRelayDashboardSnapshot(node, latestFrps)
	}
	return view, nil
}

// CleanupHealthEvents removes all health events for a node.
func CleanupHealthEvents(ctx context.Context, id uint) (*HealthEventCleanupResult, error) {
	node, err := repository.GetOpenFlareNodeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	deletedCount, err := repository.DeleteOpenFlareHealthEventsByNodeID(ctx, node.NodeID)
	if err != nil {
		return nil, err
	}
	invalidateNodeObservability(ctx, node.NodeID)
	return &HealthEventCleanupResult{
		NodeID:       node.NodeID,
		DeletedCount: deletedCount,
	}, nil
}

func buildRelayDashboardSnapshot(node *model.OpenFlareNode, obs *model.OpenFlareNodeObservationFrps) *RelayDashboardSnapshot {
	if node == nil {
		return nil
	}
	totalProxies := 0
	totalConnections := 0
	clientCounts := 0
	proxies := []RelayProxyStat{}

	if obs != nil {
		totalProxies = obs.FrpsProxyCount
		totalConnections = obs.FrpsConnections
		clientCounts = obs.FrpsClientCount
		if obs.FrpsProxies != "" {
			var decoded []RelayProxyStat
			if err := json.Unmarshal([]byte(obs.FrpsProxies), &decoded); err == nil {
				proxies = decoded
			}
		}
	}
	if totalProxies < 0 {
		totalProxies = 0
	}
	onlineProxies := 0
	for _, proxy := range proxies {
		if proxy.Status == "online" {
			onlineProxies++
		}
	}
	if len(proxies) == 0 {
		onlineProxies = totalProxies
		if node.RelayStatus != "healthy" {
			onlineProxies = 0
		}
	}

	return &RelayDashboardSnapshot{
		TotalProxies:     totalProxies,
		OnlineProxies:    onlineProxies,
		OfflineProxies:   totalProxies - onlineProxies,
		Proxies:          proxies,
		TotalConnections: maxInt(totalConnections, 0),
		ClientCounts:     maxInt(clientCounts, 0),
	}
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func nodeObservabilityCacheKey(nodeID string, hours, limit int, nodeType string) string {
	return nodeID + ":" + nodeType + ":" + strconv.Itoa(hours) + ":" + strconv.Itoa(limit)
}

func normalizeObservabilityLimit(limit int) int {
	if limit <= 0 {
		return defaultObservabilityLimit
	}
	if limit > maxObservabilityLimit {
		return maxObservabilityLimit
	}
	return limit
}

func normalizeObservabilityHours(hours int) int {
	if hours <= 0 {
		return int(defaultObservabilityWindow / time.Hour)
	}
	return hours
}

func normalizeObservabilityWindow(hours int) time.Duration {
	if hours <= 0 {
		return defaultObservabilityWindow
	}
	return time.Duration(hours) * time.Hour
}
