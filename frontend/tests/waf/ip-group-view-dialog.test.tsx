import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { NextIntlClientProvider } from 'next-intl';
import type { ComponentProps } from 'react';
import { describe, expect, it, vi } from 'vitest';

import { IPGroupViewDialog } from '@/app/(main)/waf/components/ip-group-view-dialog';
import type { WAFIPGroup } from '@/lib/services/openflare';
import en from '@/messages/fragments/security.en.json';
import ru from '@/messages/fragments/security.ru.json';
import zhCN from '@/messages/fragments/security.zh-CN.json';
import zhTW from '@/messages/fragments/security.zh-TW.json';

const group: WAFIPGroup = {
  id: 1,
  name: 'Test group',
  type: 'manual',
  enabled: true,
  ip_list: ['192.0.2.1', '198.51.100.0/24', '2001:db8:ABCD::/48'],
  auto_config: {},
  subscription_url: '',
  subscription_format: 'text',
  subscription_mapping_rule: '',
  sync_interval_minutes: 1440,
  last_sync_status: '',
  last_sync_message: '',
  referenced_by_rule_count: 0,
  created_at: '',
  updated_at: '',
};

function renderDialog(
  overrides: Partial<ComponentProps<typeof IPGroupViewDialog>> = {},
  locale = 'zh-TW',
  messages = zhTW,
) {
  const props: ComponentProps<typeof IPGroupViewDialog> = {
    open: true,
    group,
    loading: false,
    removingIp: null,
    onOpenChange: vi.fn(),
    onRemoveIp: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
  const view = (next: typeof props) => (
    <NextIntlClientProvider
      locale={locale}
      messages={{ ...messages, common: { cancel: 'Cancel' } }}
      timeZone='Asia/Taipei'
    >
      <IPGroupViewDialog {...next} />
    </NextIntlClientProvider>
  );
  const result = render(view(props));
  return {
    ...props,
    rerenderDialog: (next: Partial<typeof props>) =>
      result.rerender(view({ ...props, ...next })),
  };
}

function search(value: string) {
  fireEvent.change(screen.getByRole('searchbox'), { target: { value } });
}

describe('IP group content search', () => {
  it('filters IP and CIDR text, trims whitespace and ignores IPv6 case', () => {
    renderDialog();
    search(' /24 ');
    expect(screen.getByText('198.51.100.0/24')).toBeVisible();
    expect(screen.queryByText('192.0.2.1')).not.toBeInTheDocument();
    expect(screen.getByText(/共 3 條 IP · 符合 1 條/)).toBeVisible();
    search('abcd');
    expect(screen.getByText('2001:db8:ABCD::/48')).toBeVisible();
    search('192.0');
    expect(screen.getByText('192.0.2.1')).toBeVisible();
  });

  it('shows no matches and restores every entry on clear or whitespace input', () => {
    renderDialog();
    search('missing');
    expect(screen.getByText('沒有符合搜尋條件的 IP。')).toBeVisible();
    fireEvent.click(screen.getByRole('button', { name: '清空搜尋' }));
    expect(screen.getByRole('searchbox')).toHaveValue('');
    expect(screen.getAllByRole('row')).toHaveLength(4);
    search('   ');
    expect(screen.getAllByRole('row')).toHaveLength(4);
  });

  it('resets on external close and group changes', () => {
    const { rerenderDialog } = renderDialog();
    search('missing');
    rerenderDialog({ open: false });
    rerenderDialog({ open: true });
    expect(screen.getByRole('searchbox')).toHaveValue('');
    search('missing');
    rerenderDialog({ group: { ...group, id: 2 } });
    expect(screen.getByRole('searchbox')).toHaveValue('');
  });

  it('deletes the filtered IP and recomputes results after the group refresh', async () => {
    const { onRemoveIp, rerenderDialog } = renderDialog();
    search('192.0');
    fireEvent.click(screen.getByRole('button', { name: '刪除 192.0.2.1' }));
    fireEvent.click(screen.getByRole('button', { name: '確認刪除' }));
    await waitFor(() => expect(onRemoveIp).toHaveBeenCalledWith('192.0.2.1'));
    await waitFor(() =>
      expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument(),
    );
    rerenderDialog({ group: { ...group, ip_list: group.ip_list.slice(1) } });
    expect(screen.getByRole('searchbox')).toHaveValue('192.0');
    expect(screen.getByText('沒有符合搜尋條件的 IP。')).toBeVisible();
    expect(screen.getByText(/共 2 條 IP · 符合 0 條/)).toBeVisible();
    rerenderDialog({ group: { ...group, ip_list: [] } });
    fireEvent.click(screen.getByRole('button', { name: '清空搜尋' }));
    expect(screen.getByText('目前 IP 组暫無 IP 條目。')).toBeVisible();
  });

  it('retains automatic metadata and subscription warnings while filtering', () => {
    const { rerenderDialog } = renderDialog({
      group: {
        ...group,
        type: 'automatic',
        auto_config: { ttl: -1 },
        ext_ips: [{ ip: '203.0.113.5', captured_at: '2026-11-09T00:00:00Z' }],
      },
    });
    search('203.0');
    expect(
      screen.getByRole('columnheader', { name: '抓取時間' }),
    ).toBeVisible();
    expect(screen.getByText('永久')).toBeVisible();
    rerenderDialog({ group: { ...group, id: 2, type: 'subscription' } });
    search('/24');
    expect(
      screen.getByText(zhTW.ipGroups.viewDialog.subscriptionHint),
    ).toBeVisible();
    expect(
      screen.queryByRole('columnheader', { name: '抓取時間' }),
    ).not.toBeInTheDocument();
  });

  it.each([
    ['zh-TW', zhTW],
    ['zh-CN', zhCN],
    ['en', en],
    ['ru', ru],
  ])('renders translated search controls in %s', (locale, messages) => {
    renderDialog({}, locale, messages);
    expect(screen.getByRole('searchbox')).toHaveAccessibleName(
      messages.ipGroups.viewDialog.searchPlaceholder,
    );
    search('missing');
    expect(
      screen.getByText(messages.ipGroups.viewDialog.noSearchResult),
    ).toBeVisible();
    expect(
      screen.getByRole('button', {
        name: messages.ipGroups.viewDialog.clearSearch,
      }),
    ).toBeVisible();
  });
});
