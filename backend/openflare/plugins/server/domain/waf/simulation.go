// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package waf

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"strings"

	"Wavelet/openflare/plugins/server/kernel/repository"
)

const maxSimulationSteps = 128

type SimulationInput struct {
	Graph   RuleGraph         `json:"graph"`
	Request SimulationRequest `json:"request"`
}

type SimulationRequest struct {
	IP        string `json:"ip"`
	Country   string `json:"country"`
	Region    string `json:"region"`
	UserAgent string `json:"user_agent"`
	Path      string `json:"path"`
	Query     string `json:"query"`
	Body      string `json:"body"`
}

type SimulationResult struct {
	Decision   string   `json:"decision"`
	StatusCode int      `json:"status_code,omitempty"`
	Trace      []string `json:"trace"`
}

type simulationIPGroup struct {
	IPs      []netip.Addr
	Prefixes []netip.Prefix
}

func SimulateRuleGraph(ctx context.Context, input SimulationInput) (*SimulationResult, error) {
	if err := ValidateRuleGraph(ctx, input.Graph, ruleIPGroupExists); err != nil {
		return nil, &RuleValidationError{Err: fmt.Errorf("规则图无效: %w", err)}
	}
	addr, err := netip.ParseAddr(strings.TrimSpace(input.Request.IP))
	if err != nil {
		return nil, &RuleValidationError{Err: errors.New("模拟请求的 IP 地址无效")}
	}
	groups, err := loadSimulationIPGroups(ctx, ReferencedIPGroupIDs(input.Graph))
	if err != nil {
		return nil, err
	}
	compiled, err := CompileRuleGraph(input.Graph)
	if err != nil {
		return nil, &RuleValidationError{Err: err}
	}
	result := &SimulationResult{Decision: "indeterminate", Trace: make([]string, 0, len(compiled.Nodes))}
	nodeID := compiled.Entry
	for range maxSimulationSteps {
		node := compiled.Nodes[nodeID]
		result.Trace = append(result.Trace, nodeID)
		switch node.Type {
		case RuleNodeAllow:
			result.Decision = "allow"
			return result, nil
		case RuleNodeBlock:
			config := node.Config.(BlockNodeConfig)
			result.Decision = "block"
			result.StatusCode = config.StatusCode
			return result, nil
		case RuleNodePoW:
			result.Decision = "challenge"
			return result, nil
		case RuleNodeIPMatch:
			config := node.Config.(IPMatchConfig)
			nodeID = branchTarget(node, simulationIPMatches(addr, config, groups))
		case RuleNodeGeoMatch:
			config := node.Config.(GeoMatchConfig)
			nodeID = branchTarget(node, matchesGeo(input.Request, config))
		case RuleNodeUACheck:
			config := node.Config.(UACheckConfig)
			nodeID = branchTarget(node, matchesUA(input.Request.UserAgent, config))
		case RuleNodeSecurityCheck:
			config := node.Config.(SecurityCheckConfig)
			nodeID = branchTarget(node, matchesSecurityPayload(input.Request, config))
		case RuleNodeStart:
			nodeID = node.Next["next"]
		default:
			return nil, fmt.Errorf("unsupported simulation node type %s", node.Type)
		}
	}
	return nil, errors.New("WAF 模拟超过最大执行步数")
}

func loadSimulationIPGroups(ctx context.Context, ids []uint) (map[uint]simulationIPGroup, error) {
	groups := make(map[uint]simulationIPGroup, len(ids))
	for _, id := range ids {
		group, err := repository.GetOpenFlareWAFIPGroupByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if !group.Enabled {
			continue
		}
		var raw []string
		if err = json.Unmarshal([]byte(group.IPList), &raw); err != nil {
			return nil, fmt.Errorf("decode WAF IP group %d: %w", id, err)
		}
		entry := simulationIPGroup{}
		for _, value := range raw {
			if prefix, parseErr := netip.ParsePrefix(value); parseErr == nil {
				entry.Prefixes = append(entry.Prefixes, prefix)
			} else if address, addressErr := netip.ParseAddr(value); addressErr == nil {
				entry.IPs = append(entry.IPs, address)
			}
		}
		groups[id] = entry
	}
	return groups, nil
}

func branchTarget(node RuntimeRuleNode, matched bool) string {
	if matched {
		return node.Next["true"]
	}
	return node.Next["false"]
}

func simulationIPMatches(addr netip.Addr, config IPMatchConfig, groups map[uint]simulationIPGroup) bool {
	for _, raw := range config.IPs {
		if candidate, err := netip.ParseAddr(raw); err == nil && candidate == addr {
			return true
		}
	}
	for _, raw := range config.CIDRs {
		if prefix, err := netip.ParsePrefix(raw); err == nil && prefix.Contains(addr) {
			return true
		}
	}
	for _, id := range config.IPGroupIDs {
		group := groups[id]
		for _, candidate := range group.IPs {
			if candidate == addr {
				return true
			}
		}
		for _, prefix := range group.Prefixes {
			if prefix.Contains(addr) {
				return true
			}
		}
	}
	return false
}

func matchesGeo(request SimulationRequest, config GeoMatchConfig) bool {
	for _, country := range config.Countries {
		if strings.EqualFold(country, request.Country) {
			return true
		}
	}
	for _, region := range config.Regions {
		if strings.EqualFold(region, request.Region) {
			return true
		}
	}
	return false
}

func matchesUA(userAgent string, config UACheckConfig) bool {
	value := strings.ToLower(userAgent)
	if config.RequireUA && strings.TrimSpace(value) == "" {
		return false
	}
	if config.BlockCommonBots && (strings.Contains(value, "bot") || strings.Contains(value, "spider") || strings.Contains(value, "crawler") || strings.Contains(value, "slurp")) {
		return false
	}
	knownUA := strings.Contains(value, "micromessenger") || strings.Contains(value, "postman") || strings.Contains(value, "curl/") || strings.Contains(value, "wget/") || strings.Contains(value, "edg/") || strings.Contains(value, "firefox") || strings.Contains(value, "chrome") || strings.Contains(value, "safari") || strings.Contains(value, "android") || strings.Contains(value, "iphone") || strings.Contains(value, "ipad") || strings.Contains(value, "windows") || strings.Contains(value, "linux")
	if config.BlockAbnormalUA && !knownUA {
		return false
	}
	if config.BlockCustomUA {
		for _, pattern := range config.CustomUAPatterns {
			if matched, err := regexp.MatchString(pattern, userAgent); err == nil && matched {
				return false
			}
		}
	}
	return true
}

func matchesSecurityPayload(request SimulationRequest, config SecurityCheckConfig) bool {
	value := strings.ToLower(request.Path + "?" + request.Query + " " + request.Body)
	checks := []struct {
		enabled bool
		markers []string
	}{
		{config.PathTraversal, []string{"../", "..\\"}},
		{config.FileInclusion, []string{"file://", "php://", "/etc/passwd"}},
		{config.SQLInjection, []string{"union select", "' or 1=1", "--"}},
		{config.CommandInjection, []string{";id", "|id", "$("}},
		{config.XSS, []string{"<script", "javascript:"}},
		{config.SSRF, []string{"169.254.169.254"}},
		{config.XXE, []string{"<!entity", "<!doctype"}},
		{config.CRLFInjection, []string{"%0d%0a", "\\r\\n"}},
		{config.MaliciousUpload, []string{"<?php", "mimetype="}},
	}
	for _, check := range checks {
		if !check.enabled {
			continue
		}
		for _, marker := range check.markers {
			if strings.Contains(value, marker) {
				return false
			}
		}
	}
	return true
}
