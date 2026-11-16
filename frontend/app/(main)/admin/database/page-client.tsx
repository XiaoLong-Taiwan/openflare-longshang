'use client';

import dynamic from 'next/dynamic';
import { useCallback, useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Download, RefreshCw } from 'lucide-react';

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import services from '@/lib/services';
import type { DBOverview } from '@/lib/services/db-manage';

const sectionFallback = <Skeleton className='h-48 w-full' />;

const CacheManager = dynamic(
  () => import('./components/cache-manager').then((mod) => mod.CacheManager),
  { loading: () => sectionFallback },
);

const MaintenanceCard = dynamic(
  () =>
    import('./components/maintenance-card').then((mod) => mod.MaintenanceCard),
  { loading: () => sectionFallback },
);

export function DatabaseOverview() {
  const t = useTranslations('admin.database');
  const [overview, setOverview] = useState<DBOverview | null>(null);
  const [loadingOverview, setLoadingOverview] = useState<boolean>(true);
  const [exporting, setExporting] = useState<boolean>(false);
  const [cacheOpened, setCacheOpened] = useState(false);

  const [refreshTrigger, setRefreshTrigger] = useState<number>(0);

  const fetchOverview = useCallback(
    async (isSilent = false) => {
      if (!isSilent) setLoadingOverview(true);
      try {
        const data = await services.dbManage.getOverview();
        setOverview(data);
      } catch (err) {
        toast.error(t('fetchOverviewFailed'), {
          description: err instanceof Error ? err.message : t('unknownError'),
        });
      } finally {
        setLoadingOverview(false);
      }
    },
    [t],
  );

  const handleRefreshAll = () => {
    void fetchOverview();
    setRefreshTrigger((prev) => prev + 1);
  };

  const handleExport = async () => {
    setExporting(true);
    try {
      const { blob, filename } = await services.dbManage.exportDatabase();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
      toast.success(t('exportSuccess'), {
        description: t('exportSuccessDesc', { filename }),
      });
    } catch (err) {
      toast.error(t('exportFailed'), {
        description: err instanceof Error ? err.message : t('exportException'),
      });
    } finally {
      setExporting(false);
    }
  };

  useEffect(() => {
    void fetchOverview();
  }, [fetchOverview]);

  return (
    <div className='flex min-w-0 flex-col gap-4'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <p className='text-sm text-muted-foreground'>
          {t('overviewDescription')}
        </p>
        <Button
          size='sm'
          variant='secondary'
          className='h-8 gap-1.5 text-xs self-start sm:self-auto'
          onClick={handleRefreshAll}
          disabled={loadingOverview}
        >
          <RefreshCw data-icon='inline-start' />
          {t('refreshData')}
        </Button>
      </div>

      <div className='grid min-w-0 grid-cols-1 items-start gap-4 xl:grid-cols-2'>
        <Card className='min-w-0 gap-4'>
          <CardHeader>
            <CardTitle>{t('sections.capacity')}</CardTitle>
            <CardDescription>{t('capacityDescription')}</CardDescription>
          </CardHeader>
          <CardContent>
            {loadingOverview ? (
              <div className='flex flex-col gap-3'>
                {Array.from({ length: 6 }, (_, index) => (
                  <Skeleton key={index} className='h-10 w-full' />
                ))}
              </div>
            ) : overview ? (
              <dl className='grid min-w-0 grid-cols-1 gap-4 sm:grid-cols-2'>
                {(
                  [
                    ['dbSize', overview.size],
                    ['physicalTables', overview.table_count],
                    ['activeConnections', overview.connections],
                    ['dbType', overview.type],
                    ['versionInfo', overview.version],
                    ['nameOrPath', overview.name],
                  ] as const
                ).map(([key, value]) => (
                  <div key={key} className='flex min-w-0 flex-col gap-1'>
                    <dt className='text-sm text-muted-foreground'>{t(key)}</dt>
                    <dd className='break-all text-sm font-medium'>{value}</dd>
                  </div>
                ))}
              </dl>
            ) : (
              <p className='text-sm text-muted-foreground'>
                {t('fetchOverviewFailed')}
              </p>
            )}
          </CardContent>
        </Card>
        <div className='flex min-w-0 flex-col gap-4'>
          <MaintenanceCard />
          <Card className='min-w-0 gap-4'>
            <CardHeader>
              <CardTitle className='flex items-center gap-2'>
                <Download className='size-5 text-primary' />
                {t('dbBackupExport')}
              </CardTitle>
              <CardDescription>{t('dbBackupExportDesc')}</CardDescription>
            </CardHeader>
            <CardContent>
              <Button
                size='sm'
                variant='outline'
                className='h-auto min-h-9 whitespace-normal'
                onClick={handleExport}
                disabled={exporting}
              >
                <Download data-icon='inline-start' />
                {exporting ? t('preparing') : t('startExport')}
              </Button>
            </CardContent>
          </Card>
        </div>
        <div className='min-w-0 xl:col-span-2'>
          {cacheOpened ? (
            <CacheManager refreshTrigger={refreshTrigger} />
          ) : (
            <Card className='gap-4'>
              <CardHeader>
                <CardTitle>{t('cache.cacheManagement')}</CardTitle>
                <CardDescription>{t('cacheLazyDescription')}</CardDescription>
              </CardHeader>
              <CardContent>
                <Button variant='outline' onClick={() => setCacheOpened(true)}>
                  {t('loadCache')}
                </Button>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}
