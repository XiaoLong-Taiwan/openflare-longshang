// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"net"
	"testing"

	ofgeoip "Wavelet/openflare/plugins/server/kernel/geoip"
	"Wavelet/openflare/plugins/server/kernel/model"
	pkggeoip "Wavelet/openflare/share/geoip"
)

type fakeGeoIPProvider struct {
	info *pkggeoip.GeoInfo
}

func (f *fakeGeoIPProvider) Name() string { return "fake-geoip" }

func (f *fakeGeoIPProvider) GetGeoInfo(ip net.IP) (*pkggeoip.GeoInfo, error) {
	return f.info, nil
}

func (f *fakeGeoIPProvider) UpdateDatabase() error { return nil }

func (f *fakeGeoIPProvider) Close() error { return nil }

func withFakeGeoIPProvider(t *testing.T, info *pkggeoip.GeoInfo) {
	t.Helper()
	previous := pkggeoip.CurrentProvider
	pkggeoip.CurrentProvider = &fakeGeoIPProvider{info: info}
	t.Cleanup(func() {
		pkggeoip.CurrentProvider = previous
	})
}

func geoipFloat(value float64) *float64 {
	return &value
}

func TestApplyGeoInfoFromIP(t *testing.T) {
	latitude := 31.2304
	longitude := 121.4737
	withFakeGeoIPProvider(t, &pkggeoip.GeoInfo{
		Name:      "Shanghai",
		Latitude:  geoipFloat(latitude),
		Longitude: geoipFloat(longitude),
	})

	node := &model.OpenFlareNode{IP: "203.0.113.10"}
	ofgeoip.ApplyNodeGeoFromIP(context.Background(), node, node.IP)

	if node.GeoName != "Shanghai" {
		t.Fatalf("expected geo_name Shanghai, got %q", node.GeoName)
	}
	if node.GeoLatitude == nil || *node.GeoLatitude != latitude {
		t.Fatalf("unexpected geo_latitude: %+v", node.GeoLatitude)
	}
	if node.GeoLongitude == nil || *node.GeoLongitude != longitude {
		t.Fatalf("unexpected geo_longitude: %+v", node.GeoLongitude)
	}
}

func TestApplyGeoInfoFromIPSkipsInvalidIP(t *testing.T) {
	withFakeGeoIPProvider(t, &pkggeoip.GeoInfo{Name: "Should Not Apply"})

	node := &model.OpenFlareNode{
		IP:           "203.0.113.10",
		GeoName:      "Existing",
		GeoLatitude:  geoipFloat(1),
		GeoLongitude: geoipFloat(2),
	}
	ofgeoip.ApplyNodeGeoFromIP(context.Background(), node, "not-an-ip")

	if node.GeoName != "" || node.GeoLatitude != nil || node.GeoLongitude != nil {
		t.Fatalf("expected geo fields to be cleared on invalid IP, got %+v", node)
	}
}

func TestApplyNodeRuntimeRespectsGeoManualOverride(t *testing.T) {
	withFakeGeoIPProvider(t, &pkggeoip.GeoInfo{
		Name:      "Shanghai",
		Latitude:  geoipFloat(31.2304),
		Longitude: geoipFloat(121.4737),
	})

	node := &model.OpenFlareNode{
		GeoManualOverride: true,
		GeoName:           "Manual",
		GeoLatitude:       geoipFloat(10),
		GeoLongitude:      geoipFloat(20),
	}
	applyNodeRuntime(context.Background(), node, NodePayload{
		IP:      "203.0.113.10",
		Version: "1.0.0",
	}, true)

	if node.GeoName != "Manual" {
		t.Fatalf("expected manual geo_name to be preserved, got %q", node.GeoName)
	}
	if node.GeoLatitude == nil || *node.GeoLatitude != 10 {
		t.Fatalf("expected manual geo_latitude to be preserved, got %+v", node.GeoLatitude)
	}
}

func TestMergeNodeVersion(t *testing.T) {
	tests := []struct {
		name            string
		knownVersion    string
		reportedVersion string
		want            string
	}{
		{name: "old binary preserves known release", knownVersion: "v0.0.24", reportedVersion: "dev", want: "v0.0.24"},
		{name: "new release replaces known release", knownVersion: "v0.0.23", reportedVersion: "v0.0.24", want: "v0.0.24"},
		{name: "new dev node remains dev", reportedVersion: "dev", want: "dev"},
		{name: "empty known version accepts dev", reportedVersion: "dev", want: "dev"},
		{name: "blank report clears version", knownVersion: "v0.0.24", reportedVersion: " ", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mergeNodeVersion(tt.knownVersion, tt.reportedVersion); got != tt.want {
				t.Errorf("mergeNodeVersion(%q, %q) = %q, want %q", tt.knownVersion, tt.reportedVersion, got, tt.want)
			}
		})
	}
}

func TestApplyNodeRuntimePreservesReleaseVersionFromDev(t *testing.T) {
	withFakeGeoIPProvider(t, nil)
	node := &model.OpenFlareNode{Version: "v0.0.24"}
	applyNodeRuntime(context.Background(), node, NodePayload{
		IP:      "203.0.113.24",
		Version: "dev",
	}, true)
	if node.Version != "v0.0.24" {
		t.Errorf("applyNodeRuntime(version=dev) stored version = %q, want v0.0.24", node.Version)
	}

	newNode := &model.OpenFlareNode{}
	applyNodeRuntime(context.Background(), newNode, NodePayload{
		IP:      "203.0.113.25",
		Version: "dev",
	}, false)
	if newNode.Version != "dev" {
		t.Errorf("applyNodeRuntime(new node, version=dev) stored version = %q, want dev", newNode.Version)
	}
}

func TestCollectHeartbeatChangesTracksGeoFields(t *testing.T) {
	before := &model.OpenFlareNode{
		IP:      "10.0.0.1",
		GeoName: "Old Region",
	}
	after := &model.OpenFlareNode{
		IP:           "203.0.113.10",
		GeoName:      "New Region",
		GeoLatitude:  geoipFloat(31.2304),
		GeoLongitude: geoipFloat(121.4737),
	}

	changes := collectHeartbeatChanges(before, after)
	if changes["ip"] != after.IP {
		t.Fatalf("expected ip change, got %+v", changes)
	}
	if changes["geo_name"] != after.GeoName {
		t.Fatalf("expected geo_name change, got %+v", changes)
	}
	if changes["geo_latitude"] != after.GeoLatitude {
		t.Fatalf("expected geo_latitude change, got %+v", changes)
	}
	if changes["geo_longitude"] != after.GeoLongitude {
		t.Fatalf("expected geo_longitude change, got %+v", changes)
	}
}
