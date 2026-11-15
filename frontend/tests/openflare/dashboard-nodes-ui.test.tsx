import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { NextIntlClientProvider } from 'next-intl';
import { afterEach, expect, it, vi } from 'vitest';

import { DashboardMetricScope } from '@/app/(main)/components/dashboard/metric-scope';
import { NodeDetailShell } from '@/app/(main)/nodes/components/node-detail-shell';
import { NodesTable } from '@/app/(main)/nodes/components/nodes-table';
import type { DashboardOverview, NodeItem } from '@/lib/services/openflare';
import en from '@/messages/fragments/ops.en.json';
import enUi from '@/messages/fragments/ops-ui.en.json';
import zh from '@/messages/fragments/ops.zh-CN.json';
import zhUi from '@/messages/fragments/ops-ui.zh-CN.json';
import tw from '@/messages/fragments/ops.zh-TW.json';
import twUi from '@/messages/fragments/ops-ui.zh-TW.json';
import ru from '@/messages/fragments/ops.ru.json';
import ruUi from '@/messages/fragments/ops-ui.ru.json';

vi.mock('next/navigation', () => ({
  useRouter: () => ({ replace: vi.fn() }),
  usePathname: () => '/nodes/detail',
  useSearchParams: () => new URLSearchParams('id=1'),
}));

afterEach(cleanup);

const locales = [
  ['en', en, enUi],
  ['zh-CN', zh, zhUi],
  ['zh-TW', tw, twUi],
  ['ru', ru, ruUi],
] as const;

it.each(locales)(
  'expands metric definitions and keeps fallback information in %s',
  (locale, base, ui) => {
    const overview = {
      traffic: { average_qps: 0.5, summary_fallback: true },
      capacity: {},
      summary: { total_nodes: 3 },
    } as DashboardOverview;
    render(
      <NextIntlClientProvider
        locale={locale}
        messages={{
          dashboard: {
            ...base.dashboard,
            scope: { ...base.dashboard.scope, ...ui.dashboard.scope },
          },
        }}
        timeZone='UTC'
      >
        <DashboardMetricScope overview={overview} />
      </NextIntlClientProvider>,
    );
    expect(screen.getByText('0.5000')).toBeVisible();
    const trigger = screen.getByRole('button', {
      name: ui.dashboard.scope.explanation,
    });
    expect(trigger).toHaveAttribute('aria-expanded', 'false');
    expect(
      screen.queryByText(base.dashboard.scope.detailBasis),
    ).not.toBeInTheDocument();
    fireEvent.click(trigger);
    expect(trigger).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByText(base.dashboard.scope.detailBasis)).toBeVisible();
    expect(screen.getByText(base.dashboard.scope.fallback)).toBeVisible();
    fireEvent.click(trigger);
    expect(trigger).toHaveAttribute('aria-expanded', 'false');
  },
);

it('keeps a single selectable node and preserves actions for the responsive summary', () => {
  const onToggle = vi.fn();
  const onEdit = vi.fn();
  const onDelete = vi.fn();
  const node = {
    id: 1,
    name: 'edge-'.repeat(80),
    status: 'online',
    node_type: 'edge_node',
  } as NodeItem;
  render(
    <NextIntlClientProvider locale='en' messages={en} timeZone='UTC'>
      <NodesTable
        nodes={[node]}
        deletingId={null}
        selectedIds={new Set()}
        onToggle={onToggle}
        onToggleAll={vi.fn()}
        onEdit={onEdit}
        onDelete={onDelete}
      />
    </NextIntlClientProvider>,
  );
  expect(screen.getAllByText(node.name)).toHaveLength(1);
  fireEvent.click(
    screen.getByRole('checkbox', { name: `Select ${node.name}` }),
  );
  expect(onToggle).toHaveBeenCalledWith(1);
  expect(screen.getByRole('link', { name: 'Details' })).toHaveAttribute(
    'href',
    '/nodes/detail?id=1',
  );
  fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
  fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
  expect(onEdit).toHaveBeenCalledWith(node);
  expect(onDelete).toHaveBeenCalledWith(node);
});

it('toggles detail actions without invoking them and preserves long titles', () => {
  const onAction = vi.fn();
  const title = 'node-'.repeat(100);
  render(
    <NextIntlClientProvider
      locale='en'
      messages={{ nodes: { ...en.nodes, ...enUi.nodes } }}
      timeZone='UTC'
    >
      <NodeDetailShell
        title={title}
        typeLabel='Agent'
        statusBadges={[]}
        actions={<button onClick={onAction}>Restart</button>}
        kpis={[]}
        overview={<p>Overview content</p>}
        manage={<p>Manage content</p>}
      />
    </NextIntlClientProvider>,
  );
  expect(screen.getByRole('heading', { name: title })).toBeInTheDocument();
  const trigger = screen.getByRole('button', { name: 'More actions' });
  expect(trigger).toHaveAttribute('aria-expanded', 'false');
  fireEvent.click(trigger);
  expect(trigger).toHaveAttribute('aria-expanded', 'true');
  expect(onAction).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole('button', { name: 'Restart' }));
  expect(onAction).toHaveBeenCalledOnce();
});
