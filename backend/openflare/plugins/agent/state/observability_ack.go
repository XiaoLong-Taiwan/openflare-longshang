package state

import "Wavelet/openflare/plugins/agent/protocol"

func (s *ObservabilityBufferStore) AckRecords(sent []ObservabilityBufferRecord, retainAfterUnix int64) error {
	if s == nil || len(sent) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.loadUnlocked()
	if err != nil {
		return err
	}
	acked := make(map[int64]ObservabilityBufferRecord, len(sent))
	for _, record := range sent {
		acked[record.WindowStartedAtUnix] = record
	}
	remaining := make([]ObservabilityBufferRecord, 0, len(records))
	for _, record := range records {
		if delivered, ok := acked[record.WindowStartedAtUnix]; ok {
			keys := make(map[protocol.NodeAccessLog]int, len(delivered.AccessLogs))
			for _, item := range delivered.AccessLogs {
				keys[item]++
			}
			logs := make([]protocol.NodeAccessLog, 0, len(record.AccessLogs))
			for _, item := range record.AccessLogs {
				if keys[item] == 0 {
					logs = append(logs, item)
				} else {
					keys[item]--
				}
			}
			record.AccessLogs = logs
			if delivered.HostMetrics != nil && record.HostMetrics != nil && *record.HostMetrics == *delivered.HostMetrics {
				record.HostMetrics = nil
			}
			if delivered.EdgeHealth != nil && record.EdgeHealth != nil && *record.EdgeHealth == *delivered.EdgeHealth {
				record.EdgeHealth = nil
			}
			if len(record.AccessLogs) == 0 && record.HostMetrics == nil && record.EdgeHealth == nil {
				continue
			}
		}
		remaining = append(remaining, record)
	}
	return s.saveUnlocked(pruneObservabilityBufferRecords(remaining, retainAfterUnix))
}
