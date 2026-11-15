'use client';

import { useTranslations } from 'next-intl';
import { ChevronDown } from 'lucide-react';

import { Button } from '@/components/ui/button';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible';
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
      <CardHeader className='pb-3'>
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
        <dl className='grid grid-cols-2 gap-3 xl:grid-cols-4'>
          {metrics.map((metric) => (
            <div key={metric.label}>
              <dt className='text-sm text-muted-foreground'>{metric.label}</dt>
              <dd className='text-xl font-semibold tabular-nums'>
                {metric.value}
              </dd>
            </div>
          ))}
        </dl>
        <Collapsible>
          <CollapsibleTrigger asChild>
            <Button
              variant='ghost'
              size='sm'
              className='group w-full justify-between'
            >
              {t('explanation')}
              <ChevronDown
                data-icon='inline-end'
                className='group-data-[state=open]:rotate-180'
              />
            </Button>
          </CollapsibleTrigger>
          <CollapsibleContent className='pt-3'>
            <div className='flex flex-col gap-3 text-sm text-muted-foreground'>
              <p>{t('detailBasis')}</p>
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
            </div>
          </CollapsibleContent>
        </Collapsible>
      </CardContent>
    </Card>
  );
}
