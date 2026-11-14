import { describe, expect, it } from 'vitest';

import type { CloudflareGroupPayload } from '@/lib/services/openflare';

describe('Cloudflare dual-stack payload', () => {
  it('keeps IPv6 and debounce settings in the group payload contract', () => {
    const payload: CloudflareGroupPayload = {
      name: 'dual-stack',
      nodes: [{ node_id: 1, priority: 0, ipv6: '2001:db8::1' }],
      debounce_seconds: 30,
      default_proxied: false,
      enabled: true,
    };

    expect(payload.nodes?.[0].ipv6).toBe('2001:db8::1');
    expect(payload.debounce_seconds).toBe(30);
  });
});
