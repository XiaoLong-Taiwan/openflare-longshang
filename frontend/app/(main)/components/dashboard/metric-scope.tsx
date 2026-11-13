'use client';

import { useTranslations } from 'next-intl';

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import type { DashboardOverview } from '@/lib/services/openflare';
import { formatDateTime } from '@/lib/utils';

import { formatPercent } from './dashboard-utils';

export function DashboardMetricScope({
  overview,
}: {
  overview: DashboardOverview;
}) {
  const t = useTranslations('dashboard.scope');
  const { traffic, capacity, summary } = overview;
  const unavailable = t('unavailable');
  const start = traffic.window_started_at;
  const end = traffic.window_ended_at;
  const coverage = capacity.reported_nodes;
  const metrics = [
    {
      label: t('averageQps'),
      value: (traffic.average_qps ?? traffic.estimated_qps ?? 0).toFixed(4),
    },
    {
      label: t('clientErrorRate'),
      value:
        traffic.status_4xx_rate === undefined
          ? unavailable
          : formatPercent(traffic.status_4xx_rate),
    },
    {
      label: t('serverErrorRate'),
      value:
        traffic.status_5xx_rate === undefined
          ? unavailable
          : formatPercent(traffic.status_5xx_rate),
    },
    {
      label: t('coverage'),
      value:
        coverage === undefined
          ? unavailable
          : `${coverage}/${summary.total_nodes}`,
    },
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('title')}</CardTitle>
        <CardDescription>
          {start && end
            ? t('window', {
                start: formatDateTime(start),
                end: formatDateTime(end),
              })
            : t('legacyWindow')}
        </CardDescription>
      </CardHeader>
      <CardContent className='flex flex-col gap-4'>
        <dl className='grid gap-4 sm:grid-cols-2 xl:grid-cols-4'>
          {metrics.map((metric) => (
            <div key={metric.label}>
              <dt className='text-sm text-muted-foreground'>{metric.label}</dt>
              <dd className='text-xl font-semibold tabular-nums'>
                {metric.value}
              </dd>
            </div>
          ))}
        </dl>
        <p className='text-sm text-muted-foreground'>{t('detailBasis')}</p>
        <p className='text-sm text-muted-foreground'>
          {coverage === undefined
            ? unavailable
            : t('freshness', {
                fresh: capacity.fresh_nodes ?? 0,
                stale: capacity.stale_nodes ?? 0,
                missing: capacity.missing_nodes ?? 0,
                seconds: capacity.freshness_threshold_seconds ?? 120,
              })}
        </p>
        <p className='text-sm text-muted-foreground'>
          {t('latestSnapshot', {
            time: capacity.latest_snapshot_at
              ? formatDateTime(capacity.latest_snapshot_at)
              : unavailable,
          })}
        </p>
        {traffic.summary_fallback ? (
          <p className='text-sm text-muted-foreground'>{t('fallback')}</p>
        ) : null}
      </CardContent>
    </Card>
  );
}
