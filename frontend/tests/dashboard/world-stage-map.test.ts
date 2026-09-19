import { describe, expect, it } from 'vitest';

import {
  buildNodeMapHref,
  buildNodeMapTooltip,
  groupDashboardNodes,
} from '@/app/(main)/components/dashboard/world-stage-map';
import type { DashboardNodeHealth } from '@/lib/services/openflare';

function createNode(
  overrides: Partial<DashboardNodeHealth> = {},
): DashboardNodeHealth {
  return {
    id: 1,
    node_id: 'node-1',
    name: 'Taipei Edge',
    geo_name: 'Taipei, TW',
    geo_latitude: 25.033,
    geo_longitude: 121.5654,
    status: 'online',
    openresty_status: 'healthy',
    current_version: '1.0.0',
    last_seen_at: '2026-09-19T00:00:00Z',
    active_event_count: 0,
    cpu_usage_percent: 20,
    memory_usage_percent: 30,
    storage_usage_percent: 40,
    request_count: 100,
    error_count: 2,
    unique_visitor_count: 80,
    ...overrides,
  };
}

describe('global stage node map', () => {
  it('groups nodes at the same coordinates and keeps the most severe tone', () => {
    const groups = groupDashboardNodes([
      createNode(),
      createNode({
        id: 2,
        node_id: 'node-2',
        name: 'Taipei Backup',
        geo_name: 'New Taipei, TW',
        status: 'offline',
      }),
    ]);

    expect(groups).toHaveLength(1);
    expect(groups[0].nodes).toHaveLength(2);
    expect(groups[0].tone).toBe('danger');
  });

  it('groups missing-coordinate nodes by normalized geo name', () => {
    const groups = groupDashboardNodes([
      createNode({ geo_latitude: null, geo_longitude: null }),
      createNode({
        id: 2,
        node_id: 'node-2',
        name: 'Taipei Backup',
        geo_name: '  TAIPEI, TW  ',
        geo_latitude: null,
        geo_longitude: null,
      }),
    ]);

    expect(groups).toHaveLength(1);
    expect(groups[0].nodes.map((node) => node.id)).toEqual([1, 2]);
  });

  it('routes a single node to detail and multiple nodes to the list', () => {
    const first = createNode();
    const second = createNode({ id: 2, node_id: 'node-2' });

    expect(buildNodeMapHref([first])).toBe('/nodes/detail?id=1');
    expect(buildNodeMapHref([first, second])).toBe('/nodes');
  });

  it('lists grouped nodes and their count in the tooltip', () => {
    const nodes = [
      createNode(),
      createNode({
        id: 2,
        node_id: 'node-2',
        name: '<Taipei Backup>',
        status: 'pending',
      }),
    ];
    const t = (key: string, values?: Record<string, string | number | Date>) =>
      key === 'nodeCount' ? `${values?.count} nodes` : key;
    const tn = (key: string) => key;

    const tooltip = buildNodeMapTooltip(
      {
        activeEventCount: 1,
        derivedFromGeo: true,
        errorCount: 3,
        geoName: 'Taipei, TW',
        nodes,
        requestCount: 200,
      },
      t,
      tn,
    );

    expect(tooltip).toContain('2 nodes');
    expect(tooltip).toContain('Taipei Edge');
    expect(tooltip).toContain('&lt;Taipei Backup&gt;');
    expect(tooltip).not.toContain('<Taipei Backup>');
  });
});
