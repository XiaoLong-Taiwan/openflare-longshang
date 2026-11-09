import { describe, expect, it } from 'vitest';

import { buildPayloadFromRoute } from '@/app/(main)/proxy-routes/components/helpers';
import type { ProxyRouteItem } from '@/lib/services/openflare';

describe('proxy route HTTP/2 payload', () => {
  it.each([true, false, undefined])(
    'preserves HTTP/2 %s while saving another section',
    (enabled) => {
      const route = { enable_http2: enabled } as ProxyRouteItem;
      const payload = buildPayloadFromRoute(route, { cache_enabled: true });

      expect(payload.enable_http2).toBe(enabled ?? true);
      expect(payload.cache_enabled).toBe(true);
    },
  );

  it('allows the proxy section to disable HTTP/2 explicitly', () => {
    const route = { enable_http2: true } as ProxyRouteItem;

    expect(
      buildPayloadFromRoute(route, { enable_http2: false }).enable_http2,
    ).toBe(false);
  });
});
