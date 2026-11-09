package websocket

// ConnectedNodeIDs returns the distinct identities with live hub connections.
func ConnectedNodeIDs() []string {
	seen := make(map[string]struct{})
	defaultAgentHub.mu.RLock()
	for id, client := range defaultAgentHub.clients {
		select {
		case <-client.done:
		default:
			seen[id] = struct{}{}
		}
	}
	defaultAgentHub.mu.RUnlock()
	defaultRelayHub.mu.RLock()
	for id, client := range defaultRelayHub.clients {
		select {
		case <-client.done:
		default:
			seen[id] = struct{}{}
		}
	}
	defaultRelayHub.mu.RUnlock()
	defaultFlaredHub.mu.RLock()
	for id, client := range defaultFlaredHub.clients {
		select {
		case <-client.done:
		default:
			seen[id] = struct{}{}
		}
	}
	defaultFlaredHub.mu.RUnlock()
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	return ids
}
