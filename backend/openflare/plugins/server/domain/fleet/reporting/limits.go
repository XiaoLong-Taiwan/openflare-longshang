package reporting

import "Wavelet/openflare/share/protocol"

const MaxPayloadBytes = 8 << 20

func Valid(payload protocol.NodePayload) bool {
	if len(payload.Buffered) > 500 || len(payload.HealthEvents) > 100 {
		return false
	}
	logs := len(payload.AccessLogs)
	for _, record := range payload.Buffered {
		logs += len(record.AccessLogs)
	}
	return logs <= 1_000
}
