import { cleanup, render, screen } from '@testing-library/react';
import { NextIntlClientProvider } from 'next-intl';
import { afterEach, expect, it } from 'vitest';

import { DashboardMetricScope } from '@/app/(main)/components/dashboard/metric-scope';
import type { DashboardOverview } from '@/lib/services/openflare';
import en from '@/messages/fragments/ops.en.json';
import ru from '@/messages/fragments/ops.ru.json';
import cn from '@/messages/fragments/ops.zh-CN.json';
import tw from '@/messages/fragments/ops.zh-TW.json';

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
  ['en', en],
  ['ru', ru],
  ['zh-CN', cn],
  ['zh-TW', tw],
] as const)('renders scope and rates in %s', (locale, messages) => {
  render(
    <NextIntlClientProvider locale={locale} messages={messages}>
      <DashboardMetricScope overview={overview} />
    </NextIntlClientProvider>,
  );
  expect(screen.getByText(messages.dashboard.scope.title)).toBeVisible();
  expect(screen.getByText('1.0000')).toBeVisible();
  expect(screen.getByText('3/4')).toBeVisible();
  expect(screen.getByText(messages.dashboard.scope.fallback)).toBeVisible();
  expect(screen.getByText('0%')).toBeVisible();
});

it('keeps unknown legacy coverage and rates unavailable', () => {
  render(
    <NextIntlClientProvider locale='en' messages={en}>
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
    <NextIntlClientProvider locale='en' messages={en}>
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
