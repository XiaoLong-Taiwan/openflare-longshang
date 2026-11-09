import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { CloudflareGroupDetailPageClient } from '@/app/(main)/cloudflare/groups/[id]/page-client';
import { CloudflareService, NodeService } from '@/lib/services/openflare';
import { NextIntlClientProvider } from 'next-intl';
import zhCN from '@/messages/zh-CN.json';

const navigation = vi.hoisted(() => ({
  pathname: '/cloudflare/group',
  search: 'id=7',
}));

vi.mock('next/navigation', () => ({
  usePathname: () => navigation.pathname,
  useSearchParams: () => new URLSearchParams(navigation.search),
}));

vi.mock('@/lib/services/openflare', async (importOriginal) => {
  const actual =
    await importOriginal<typeof import('@/lib/services/openflare')>();
  return {
    ...actual,
    CloudflareService: {
      ...actual.CloudflareService,
      getGroup: vi.fn(),
      listAvailableDomains: vi.fn(),
      updateGroup: vi.fn(),
      createMember: vi.fn(),
      updateMember: vi.fn(),
      syncMember: vi.fn(),
      removeMember: vi.fn(),
      listGroups: vi.fn(),
      batchMoveMembers: vi.fn(),
      batchRemoveMembers: vi.fn(),
    },
    NodeService: { ...actual.NodeService, listNodes: vi.fn() },
  };
});

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });
  render(
    <NextIntlClientProvider
      locale='zh-CN'
      messages={zhCN}
      timeZone='Asia/Shanghai'
    >
      <QueryClientProvider client={client}>
        <CloudflareGroupDetailPageClient />
      </QueryClientProvider>
    </NextIntlClientProvider>,
  );
}

describe('Cloudflare group detail refresh', () => {
  beforeEach(() => {
    navigation.pathname = '/cloudflare/group';
    navigation.search = 'id=7';
    vi.mocked(CloudflareService.getGroup).mockReset();
    vi.mocked(CloudflareService.listAvailableDomains).mockReset();
    vi.mocked(NodeService.listNodes).mockReset();
    vi.mocked(CloudflareService.getGroup).mockResolvedValue({
      group: {
        id: 7,
        name: '生产节点',
        ttl: 300,
        primary_node: { id: 1, name: '主节点', ip: '192.0.2.1' },
        backup_node: null,
        active_node: { id: 1, name: '主节点', ip: '192.0.2.1' },
        default_proxied: true,
        enabled: true,
        member_count: 0,
        created_at: '',
        updated_at: '',
      },
      members: [],
    });
    vi.mocked(CloudflareService.listAvailableDomains).mockResolvedValue([]);
    vi.mocked(NodeService.listNodes).mockResolvedValue([]);
    vi.mocked(CloudflareService.listGroups).mockResolvedValue([]);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('loads a non-template group ID from the pathname before the legacy query', async () => {
    navigation.pathname = '/cloudflare/groups/42';
    renderPage();
    await waitFor(() =>
      expect(CloudflareService.getGroup).toHaveBeenCalledWith(42),
    );
    expect(CloudflareService.getGroup).not.toHaveBeenCalledWith(7);
  });

  it('does not load a group for an invalid dynamic pathname', async () => {
    navigation.pathname = '/cloudflare/groups/invalid';
    renderPage();
    await act(async () => {});
    expect(CloudflareService.getGroup).not.toHaveBeenCalled();
  });

  it.each([
    { ttl: 1, label: '灰云 TTL：自动（约 300 秒）' },
    { ttl: 600, label: '灰云 TTL：600 秒' },
  ])(
    'shows the group TTL $ttl in the detail summary',
    async ({ ttl, label }) => {
      vi.mocked(CloudflareService.getGroup).mockResolvedValue({
        group: { ...(await CloudflareService.getGroup(7)).group, ttl },
        members: [],
      });
      renderPage();
      expect(await screen.findByText(label)).toBeVisible();
    },
  );

  it('refreshes detail data when the refresh button is clicked', async () => {
    renderPage();

    expect(
      await screen.findByRole('heading', { name: '生产节点' }),
    ).toBeVisible();
    fireEvent.click(screen.getByRole('button', { name: '刷新' }));

    await waitFor(() => {
      expect(CloudflareService.getGroup).toHaveBeenCalledTimes(2);
    });
  });

  it('shows a custom target without a fabricated active priority', async () => {
    const detail = await CloudflareService.getGroup(7);
    vi.mocked(CloudflareService.getGroup).mockResolvedValue({
      ...detail,
      group: {
        ...detail.group,
        target_mode: 'custom',
        record_type: 'CNAME',
        record_content: 'origin.example.net',
        nodes: [],
      },
    });
    renderPage();
    expect(
      (await screen.findAllByText('CNAME origin.example.net')).length,
    ).toBeGreaterThan(0);
    expect(screen.queryByText(/当前生效优先级/)).not.toBeInTheDocument();
  });

  it('locks selection and both batch actions until removal completes', async () => {
    const detail = await CloudflareService.getGroup(7);
    vi.mocked(CloudflareService.getGroup).mockResolvedValue({
      ...detail,
      members: [
        {
          id: 12,
          group_id: 7,
          zone_domain_id: 8,
          domain: 'api.example.com',
          zone_id: 1,
          proxied: true,
          desired_ip: '',
          sync_status: 'pending',
          last_error: '',
          synced_at: null,
        },
      ],
    });
    let resolveRemoval!: (value: {
      member_ids: number[];
      task_ids: string[];
    }) => void;
    vi.mocked(CloudflareService.batchRemoveMembers).mockReturnValue(
      new Promise((resolve) => {
        resolveRemoval = resolve;
      }),
    );
    renderPage();
    fireEvent.click(
      await screen.findByRole('checkbox', { name: 'api.example.com' }),
    );
    const remove = screen.getByRole('button', {
      name: zhCN.cloudflare.removeSelected,
    });
    fireEvent.click(remove);
    await waitFor(() => expect(remove).toBeDisabled());
    expect(
      screen.getByRole('checkbox', { name: 'api.example.com' }),
    ).toBeDisabled();
    expect(
      screen.getByRole('button', {
        name: zhCN.cloudflare.moveSelected,
      }),
    ).toBeDisabled();
    expect(screen.getByRole('combobox')).toBeDisabled();
    fireEvent.click(remove);
    expect(CloudflareService.batchRemoveMembers).toHaveBeenCalledTimes(1);
    await act(async () => {
      resolveRemoval({ member_ids: [12], task_ids: [] });
    });
  });

  it('automatically refreshes detail data every five seconds', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    renderPage();

    expect(
      await screen.findByRole('heading', { name: '生产节点' }),
    ).toBeVisible();
    expect(CloudflareService.getGroup).toHaveBeenCalledTimes(1);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });

    await waitFor(() => {
      expect(CloudflareService.getGroup).toHaveBeenCalledTimes(2);
    });
  });
});
