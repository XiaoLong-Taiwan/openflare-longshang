import type { AxiosResponse } from 'axios';
import { expect, it, vi } from 'vitest';

import apiClient from '@/lib/services/core/api-client';
import { DashboardService } from '@/lib/services/openflare/dashboard.service';

vi.mock('@/lib/services/core/api-client', () => ({
  default: { get: vi.fn() },
}));

it('preserves additive metrics and legacy compact indexes', async () => {
  vi.mocked(apiClient.get).mockResolvedValue({
    data: {
      error_msg: '',
      data: {
        traffic: { average_qps: 1, estimated_qps: 1, status_4xx_rate: 10 },
        capacity: { fresh_nodes: 1 },
        nodes: [
          [
            1,
            'edge',
            'Edge',
            '',
            null,
            null,
            'online',
            'healthy',
            'v1',
            '',
            0,
            0,
            50,
            25,
            100,
            2,
            10,
          ],
        ],
        trends: {
          traffic_24h: [['2026-11-13T12:00:00Z', 100, 2, 10, 88, 10, 2]],
        },
      },
    },
  } as AxiosResponse);
  const result = await DashboardService.getOverview();
  expect(result?.traffic.average_qps).toBe(1);
  expect(result?.capacity.fresh_nodes).toBe(1);
  expect(result?.nodes[0]).toMatchObject({
    cpu_usage_percent: 0,
    request_count: 100,
    error_count: 2,
    unique_visitor_count: 10,
  });
  expect(result?.trends.traffic_24h[0]).toMatchObject({
    request_count: 100,
    status_4xx_count: 10,
    status_5xx_count: 2,
  });
  expect(result?.trends.network_24h).toEqual([]);
});
