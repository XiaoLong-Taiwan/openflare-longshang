'use client';

import dynamic from 'next/dynamic';
import { useState } from 'react';
import { useTranslations } from 'next-intl';

import { RankChart } from '@/components/data/rank-chart';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Map } from 'lucide-react';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import type {
  DistributionItem,
  TrafficDistributions,
} from '@/lib/services/openflare';

const WorldStageMap = dynamic(
  () => import('./world-stage-map').then((module) => module.WorldStageMap),
  { ssr: false },
);

function toRankItems(items: DistributionItem[]) {
  return items.map((item) => ({
    label: item.key,
    value: item.value,
  }));
}

export function SourceDistributionChart({
  items,
}: {
  items: TrafficDistributions['source_countries'];
}) {
  const t = useTranslations('dashboard.distributions');
  const [mapOpen, setMapOpen] = useState(false);
  return (
    <>
      <Card className='border-dashed shadow-none'>
        <CardHeader>
          <div className='flex items-start justify-between gap-3'>
            <div>
              <CardTitle className='text-sm font-semibold'>
                {t('sourceTitle')}
              </CardTitle>
              <CardDescription className='text-xs'>
                {t('sourceDesc')}
              </CardDescription>
            </div>
            <Button
              variant='ghost'
              size='icon'
              className='size-7 shrink-0'
              aria-label={t('viewMap')}
              title={t('viewMap')}
              onClick={() => setMapOpen(true)}
            >
              <Map data-icon='inline-start' />
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <RankChart
            items={toRankItems(items)}
            color='#38bdf8'
            emptyMessage={t('sourceEmpty')}
          />
        </CardContent>
      </Card>

      <Dialog open={mapOpen} onOpenChange={setMapOpen}>
        <DialogContent className='max-w-5xl'>
          <DialogHeader>
            <DialogTitle>{t('mapTitle')}</DialogTitle>
            <DialogDescription>{t('mapDesc')}</DialogDescription>
          </DialogHeader>
          <div className='grid gap-4 lg:grid-cols-[minmax(0,1fr)_260px]'>
            <div className='h-[420px] min-h-0 rounded-lg border border-dashed bg-muted/10 p-2'>
              <WorldStageMap nodes={[]} sourceCountries={items} />
            </div>
            <div className='max-h-[420px] overflow-auto rounded-lg border border-dashed'>
              <div className='border-b px-3 py-2 text-xs font-semibold'>
                {t('requestBreakdown')}
              </div>
              <div className='divide-y'>
                {items.length === 0 ? (
                  <p className='p-3 text-xs text-muted-foreground'>
                    {t('sourceEmpty')}
                  </p>
                ) : (
                  items.map((item) => (
                    <div
                      key={item.key}
                      className='flex items-center justify-between gap-3 px-3 py-2 text-xs'
                    >
                      <span className='truncate'>{item.key}</span>
                      <span className='shrink-0 font-mono tabular-nums text-muted-foreground'>
                        {item.value.toLocaleString()}
                      </span>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}

export function StatusCodeDistributionChart({
  items,
}: {
  items: TrafficDistributions['status_codes'];
}) {
  const t = useTranslations('dashboard.distributions');
  return (
    <Card className='border-dashed shadow-none'>
      <CardHeader>
        <CardTitle className='text-sm font-semibold'>
          {t('statusTitle')}
        </CardTitle>
        <CardDescription className='text-xs'>{t('statusDesc')}</CardDescription>
      </CardHeader>
      <CardContent>
        <RankChart
          items={toRankItems(items).map((item) => ({
            ...item,
            label: t('httpLabel', { code: item.label }),
          }))}
          color='#f59e0b'
          emptyMessage={t('statusEmpty')}
        />
      </CardContent>
    </Card>
  );
}

export function TopDomainChart({
  items,
}: {
  items: TrafficDistributions['top_domains'];
}) {
  const t = useTranslations('dashboard.distributions');
  return (
    <Card className='border-dashed shadow-none'>
      <CardHeader>
        <CardTitle className='text-sm font-semibold'>
          {t('domainTitle')}
        </CardTitle>
        <CardDescription className='text-xs'>{t('domainDesc')}</CardDescription>
      </CardHeader>
      <CardContent>
        <RankChart
          items={toRankItems(items)}
          color='#34d399'
          emptyMessage={t('domainEmpty')}
        />
      </CardContent>
    </Card>
  );
}
