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
import type {
  StorageOperationSource,
  StorageOperations,
} from '@/lib/services/openflare';
import type { TaskExecution } from '@/lib/services/admin';
import { formatBytes } from '@/lib/utils/metrics';

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
  const [operations, setOperations] = useState<StorageOperations | null>(null);
  const [loadingOperations, setLoadingOperations] = useState<boolean>(true);
  const [operationsFailed, setOperationsFailed] = useState<boolean>(false);
  const [cleanup, setCleanup] = useState<TaskExecution | null>(null);
  const [loadingCleanup, setLoadingCleanup] = useState<boolean>(true);
  const [cleanupFailed, setCleanupFailed] = useState<boolean>(false);
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
        setOverview(null);
        toast.error(t('fetchOverviewFailed'), {
          description: err instanceof Error ? err.message : t('unknownError'),
        });
      } finally {
        setLoadingOverview(false);
      }
    },
    [t],
  );

  const fetchOperations = useCallback(async () => {
    setLoadingOperations(true);
    setOperationsFailed(false);
    setLoadingCleanup(true);
    setCleanupFailed(false);
    const [operationsResult, cleanupResult, maintenanceResult] =
      await Promise.allSettled([
        services.openflareAccessLog.getStorageOperations(),
        services.adminTask.listTaskExecutions({
          task_type_prefix: 'system:cleanup',
          page: 1,
          page_size: 1,
        }),
        services.adminTask.listTaskExecutions({
          task_type: 'database:maintenance',
          page: 1,
          page_size: 1,
        }),
      ]);
    if (operationsResult.status === 'fulfilled') {
      setOperations(operationsResult.value);
    } else {
      setOperations(null);
      setOperationsFailed(true);
    }
    setLoadingOperations(false);
    if (cleanupResult.status === 'fulfilled' && maintenanceResult.status === 'fulfilled') {
      const candidates = [
        cleanupResult.value.items[0],
        maintenanceResult.value.items[0],
      ].filter((item): item is TaskExecution => item !== undefined);
      setCleanup(
        candidates.reduce<TaskExecution | null>((latest, item) => {
          if (!latest) return item;
          const itemTime = Date.parse(item.created_at);
          const latestTime = Date.parse(latest.created_at);
          return itemTime > latestTime ? item : latest;
        }, null),
      );
    } else {
      setCleanup(null);
      setCleanupFailed(true);
    }
    setLoadingCleanup(false);
  }, []);

  const handleRefreshAll = () => {
    void fetchOverview();
    void fetchOperations();
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
    void fetchOperations();
  }, [fetchOverview, fetchOperations]);

  const formatDate = (value?: string | null) =>
    value ? new Date(value).toLocaleString() : t('notAvailable');

  const renderSource = (source?: StorageOperationSource) => {
    if (!source) return t('queryFailed');
    if (source.status === 'available') {
      return `${formatDate(source.oldest)} - ${formatDate(source.newest)}`;
    }
    if (source.status === 'empty') return t('noRetainedData');
    if (source.status === 'unavailable') return t('unavailable');
    return t('queryFailed');
  };

  const renderTask = () => {
    if (cleanupFailed) return t('queryFailed');
    if (!cleanup) return t('noTaskRecord');
    return `${t(`taskStatus.${cleanup.status}`)} (${formatDate(cleanup.created_at)})`;
  };

  const renderBackup = () => {
    if (!overview) return t('queryFailed');
    if (overview.export_prerequisite === 'pg_dump_missing') {
      return t('backupToolMissing');
    }
    return t('backupNotVerified');
  };

  const logCapacity = operations?.capacity;
  const renderLogCapacity = () => {
    if (logCapacity?.status === 'shared') return t('sharedLogCapacity');
    if (logCapacity?.status !== 'available') return t('queryFailed');
    return formatBytes(logCapacity.bytes_on_disk, { zeroText: t('confirmedZero') });
  };

  const renderOutbox = () => {
    if (!operations) return t('queryFailed');
    if (operations.outbox_status === 'unsupported') return t('unsupported');
    if (operations.outbox_status !== 'available' || !operations.outbox) {
      return t('queryFailed');
    }
    const { pending, healthy, worker_running: workerRunning } = operations.outbox;
    return `${t('outboxPending', { count: pending })}; ${healthy ? t('healthy') : t('unhealthy')}; ${workerRunning ? t('workerRunning') : t('workerStopped')}`;
  };

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
                    [
                      'dbSize',
                      overview.size_bytes === null
                        ? t('unavailable')
                        : overview.size_bytes === 0
                          ? t('confirmedZero')
                          : overview.size,
                    ],
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
        <Card className='min-w-0 gap-4 xl:col-span-2'>
          <CardHeader>
            <CardTitle>{t('sections.operations')}</CardTitle>
            <CardDescription>{t('operationsDescription')}</CardDescription>
          </CardHeader>
          <CardContent>
            {loadingOperations || loadingCleanup ? (
              <div className='grid gap-3 sm:grid-cols-2'>
                {Array.from({ length: 6 }, (_, index) => (
                  <Skeleton key={index} className='h-10 w-full' />
                ))}
              </div>
            ) : (
              <dl className='grid min-w-0 grid-cols-1 gap-4 sm:grid-cols-2'>
                <div className='flex min-w-0 flex-col gap-1'>
                  <dt className='text-sm text-muted-foreground'>{t('backend')}</dt>
                  <dd className='break-all text-sm font-medium'>
                    {!operations || operationsFailed
                      ? t('queryFailed')
                      : operations.backend_status === 'error'
                        ? t('unavailable')
                        : operations.backend}
                  </dd>
                </div>
                <div className='flex min-w-0 flex-col gap-1'>
                  <dt className='text-sm text-muted-foreground'>{t('logCapacity')}</dt>
                  <dd className='text-sm font-medium'>{renderLogCapacity()}</dd>
                  {logCapacity?.scope === 'clickhouse_active_parts' ? (
                    <dd className='text-sm text-muted-foreground'>
                      {t('clickHouseCapacityScope')}
                    </dd>
                  ) : null}
                  {logCapacity?.status === 'available' ? (
                    <dd className='text-sm text-muted-foreground'>
                      {t('clickHouseCapacityCounts', {
                        parts: logCapacity.active_parts,
                        rows: logCapacity.total_rows,
                      })}
                    </dd>
                  ) : null}
                </div>
                <div className='flex min-w-0 flex-col gap-1'>
                  <dt className='text-sm text-muted-foreground'>{t('nodeLogs')}</dt>
                  <dd className='break-all text-sm font-medium'>
                    {operations ? renderSource(operations.node_logs) : t('queryFailed')}
                  </dd>
                </div>
                <div className='flex min-w-0 flex-col gap-1'>
                  <dt className='text-sm text-muted-foreground'>{t('userLogs')}</dt>
                  <dd className='break-all text-sm font-medium'>
                    {renderSource(operations?.user_logs)}
                  </dd>
                </div>
                <div className='flex min-w-0 flex-col gap-1'>
                  <dt className='text-sm text-muted-foreground'>{t('outbox')}</dt>
                  <dd className='break-all text-sm font-medium'>
                    {renderOutbox()}
                  </dd>
                </div>
                <div className='flex min-w-0 flex-col gap-1'>
                  <dt className='text-sm text-muted-foreground'>{t('cleanupStatus')}</dt>
                  <dd className='break-all text-sm font-medium'>
                    {renderTask()}
                  </dd>
                </div>
                <div className='flex min-w-0 flex-col gap-1'>
                  <dt className='text-sm text-muted-foreground'>{t('backupAvailability')}</dt>
                  <dd className='break-all text-sm font-medium'>
                    {renderBackup()}
                  </dd>
                </div>
              </dl>
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
