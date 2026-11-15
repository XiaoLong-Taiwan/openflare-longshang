import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { NextIntlClientProvider } from 'next-intl';
import { afterEach, expect, it } from 'vitest';

import { DashboardMetricScope } from '@/app/(main)/components/dashboard/metric-scope';
import type { DashboardOverview } from '@/lib/services/openflare';
import en from '@/messages/en.json';
import ru from '@/messages/ru.json';
import cn from '@/messages/zh-CN.json';
import tw from '@/messages/zh-TW.json';
import enUi from '@/messages/fragments/ops-ui.en.json';
import ruUi from '@/messages/fragments/ops-ui.ru.json';
import cnUi from '@/messages/fragments/ops-ui.zh-CN.json';
import twUi from '@/messages/fragments/ops-ui.zh-TW.json';

afterEach(cleanup);

const overview = {
  summary: { total_nodes: 4 },
  traffic: {
    estimated_qps: 0.5,
    average_qps: 1,
    status_4xx_rate: 10,
    status_5xx_rate: 0,
    summary_fallback: true,
  },
  capacity: {
    reported_nodes: 3,
    fresh_nodes: 2,
    stale_nodes: 1,
    missing_nodes: 1,
    freshness_threshold_seconds: 120,
  },
} as DashboardOverview;

it.each([
  ['en', en, enUi],
  ['ru', ru, ruUi],
  ['zh-CN', cn, cnUi],
  ['zh-TW', tw, twUi],
] as const)('renders rates and toggles explanations in %s', (locale, messages, ui) => {
  expect(messages.dashboard.scope).toHaveProperty(
    'explanation',
    ui.dashboard.scope.explanation,
  );
  render(
    <NextIntlClientProvider locale={locale} messages={messages} timeZone='UTC'>
      <DashboardMetricScope overview={overview} />
    </NextIntlClientProvider>,
  );
  expect(screen.getByText(messages.dashboard.scope.title)).toBeVisible();
  expect(screen.getByText('1.0000')).toBeVisible();
  expect(screen.getByText('3/4')).toBeVisible();
  expect(screen.getByText('0%')).toBeVisible();
  const trigger = screen.getByRole('button', {
    name: ui.dashboard.scope.explanation,
  });
  expect(trigger).toHaveAttribute('aria-expanded', 'false');
  expect(screen.queryByText(messages.dashboard.scope.detailBasis)).not.toBeInTheDocument();
  expect(screen.queryByText(messages.dashboard.scope.fallback)).not.toBeInTheDocument();

  fireEvent.click(trigger);

  expect(trigger).toHaveAttribute('aria-expanded', 'true');
  expect(screen.getByText(messages.dashboard.scope.detailBasis)).toBeVisible();
  expect(screen.getByText(messages.dashboard.scope.fallback)).toBeVisible();
  expect(screen.getByText('1.0000')).toBeVisible();
  expect(screen.getByText('3/4')).toBeVisible();
  expect(screen.getByText('0%')).toBeVisible();

  fireEvent.click(trigger);

  expect(trigger).toHaveAttribute('aria-expanded', 'false');
  expect(screen.queryByText(messages.dashboard.scope.detailBasis)).not.toBeInTheDocument();
  expect(screen.queryByText(messages.dashboard.scope.fallback)).not.toBeInTheDocument();
});

it('keeps unknown legacy coverage and rates unavailable', () => {
  render(
    <NextIntlClientProvider locale='en' messages={en} timeZone='UTC'>
      <DashboardMetricScope
        overview={
          {
            ...overview,
            traffic: { estimated_qps: 0.5 },
            capacity: {},
          } as DashboardOverview
        }
      />
    </NextIntlClientProvider>,
  );
  expect(screen.getByText('0.5000')).toBeVisible();
  expect(screen.getByText(en.dashboard.scope.legacyWindow)).toBeVisible();
  expect(
    screen.getAllByText(en.dashboard.scope.unavailable).length,
  ).toBeGreaterThanOrEqual(3);
});

it('renders when legacy traffic omits both QPS fields', () => {
  render(
    <NextIntlClientProvider locale='en' messages={en} timeZone='UTC'>
      <DashboardMetricScope
        overview={
          {
            ...overview,
            traffic: {},
            capacity: {},
          } as DashboardOverview
        }
      />
    </NextIntlClientProvider>,
  );
  expect(screen.getByText('0.0000')).toBeVisible();
});
