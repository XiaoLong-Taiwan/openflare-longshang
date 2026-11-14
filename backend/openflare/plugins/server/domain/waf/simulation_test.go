// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package waf

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimulateRuleGraphAllowsDefaultGraph(t *testing.T) {
	result, err := SimulateRuleGraph(context.Background(), SimulationInput{
		Graph: DefaultRuleGraph(), Request: SimulationRequest{IP: "192.0.2.1"},
	})
	require.NoError(t, err)
	require.Equal(t, "allow", result.Decision)
	require.Equal(t, []string{"start", "allow"}, result.Trace)
}

func TestSimulateRuleGraphFollowsIPBranch(t *testing.T) {
	graph := RuleGraph{SchemaVersion: RuleGraphSchemaVersion,
		Nodes: []RuleNode{
			{ID: "start", Type: RuleNodeStart, Config: json.RawMessage(`{}`)},
			{ID: "ip", Type: RuleNodeIPMatch, Config: json.RawMessage(`{"ips":["192.0.2.1"]}`)},
			{ID: "allow", Type: RuleNodeAllow, Config: json.RawMessage(`{}`)},
			{ID: "block", Type: RuleNodeBlock, Config: json.RawMessage(`{"status_code":403}`)},
		},
		Edges: []RuleEdge{
			{ID: "e1", Source: "start", SourceHandle: "next", Target: "ip"},
			{ID: "e2", Source: "ip", SourceHandle: "true", Target: "allow"},
			{ID: "e3", Source: "ip", SourceHandle: "false", Target: "block"},
		},
	}
	result, err := SimulateRuleGraph(context.Background(), SimulationInput{Graph: graph, Request: SimulationRequest{IP: "192.0.2.2"}})
	require.NoError(t, err)
	require.Equal(t, "block", result.Decision)
	require.Equal(t, 403, result.StatusCode)
	require.Equal(t, []string{"start", "ip", "block"}, result.Trace)
}

func TestSimulateRuleGraphRejectsInvalidIP(t *testing.T) {
	_, err := SimulateRuleGraph(context.Background(), SimulationInput{Graph: DefaultRuleGraph(), Request: SimulationRequest{IP: "invalid"}})
	require.ErrorContains(t, err, "IP 地址无效")
}

func TestMatchesUAUsesRuntimeBlockSwitches(t *testing.T) {
	if matchesUA("", UACheckConfig{RequireUA: true}) {
		t.Fatal("empty user agent matched required UA check")
	}
	if !matchesUA("curl/8.0", UACheckConfig{BlockCommonBots: true}) {
		t.Fatal("common bot switch blocked curl")
	}
	if matchesUA("ExampleBot/1.0", UACheckConfig{BlockCommonBots: true}) {
		t.Fatal("common bot switch did not block bot user agent")
	}
	if matchesUA("blocked-client", UACheckConfig{BlockCustomUA: true, CustomUAPatterns: []string{"blocked"}}) {
		t.Fatal("custom UA switch did not block matching user agent")
	}
}

func TestMatchesSecurityPayloadReturnsPassWhenNoSignatureMatches(t *testing.T) {
	if !matchesSecurityPayload(SimulationRequest{Path: "/safe"}, SecurityCheckConfig{PathTraversal: true}) {
		t.Fatal("safe request did not pass enabled security check")
	}
	if matchesSecurityPayload(SimulationRequest{Path: "/../etc/passwd"}, SecurityCheckConfig{PathTraversal: true}) {
		t.Fatal("path traversal signature was not blocked")
	}
}
