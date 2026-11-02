package reporting

import (
	"testing"

	"Wavelet/openflare/share/protocol"
)

func TestValidCountsNestedLogs(t *testing.T) {
	payload := protocol.NodePayload{
		AccessLogs: make([]protocol.NodeAccessLog, 600),
		Buffered:   []protocol.BufferedObservabilityRecord{{AccessLogs: make([]protocol.NodeAccessLog, 400)}},
	}
	if !Valid(payload) {
		t.Fatal("Valid(1000 combined logs) = false, want true")
	}
	payload.Buffered[0].AccessLogs = append(payload.Buffered[0].AccessLogs, protocol.NodeAccessLog{})
	if Valid(payload) {
		t.Error("Valid(1001 combined logs) = true, want false")
	}
}

func TestValidBoundsBufferedAndEvents(t *testing.T) {
	if Valid(protocol.NodePayload{Buffered: make([]protocol.BufferedObservabilityRecord, 501)}) {
		t.Error("Valid(501 buffered records) = true, want false")
	}
	if Valid(protocol.NodePayload{HealthEvents: make([]protocol.NodeHealthEvent, 101)}) {
		t.Error("Valid(101 health events) = true, want false")
	}
	if !Valid(protocol.NodePayload{Buffered: make([]protocol.BufferedObservabilityRecord, 500), HealthEvents: make([]protocol.NodeHealthEvent, 100)}) {
		t.Error("Valid(boundary counts) = false, want true")
	}
}
