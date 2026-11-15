'use client';

import dynamic from 'next/dynamic';
import { useCallback, useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import {
  Activity,
  Cpu,
  Database,
  Download,
  HardDrive,
  Layers,
  RefreshCw,
  Server,
} from 'lucide-react';

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import services from '@/lib/services';
import type { DBOverview } from '@/lib/services/db-manage';

const sectionFallback = <Skeleton className='h-48 w-full' />;

const TableBrowser = dynamic(
  () => import('./components/table-browser').then((mod) => mod.TableBrowser),
  { loading: () => sectionFallback },
);

const CacheManager = dynamic(
  () => import('./components/cache-manager').then((mod) => mod.CacheManager),
  { loading: () => sectionFallback },
);

const MaintenanceCard = dynamic(
  () =>
    import('./components/maintenance-card').then((mod) => mod.MaintenanceCard),
  { loading: () => sectionFallback },
);

/**
 * 格式化数字，每3位加逗号
 */
const formatNumber = (num: number | string) => {
  if (num === undefined || num === null) return '0';
  return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',');
};

export function DatabasePageClient() {
  const t = useTranslations('admin.database');
  const [overview, setOverview] = useState<DBOverview | null>(null);
  const [tables, setTables] = useState<string[]>([]);
  const [loadingOverview, setLoadingOverview] = useState<boolean>(true);
  const [loadingTables, setLoadingTables] = useState<boolean>(true);
  const [exporting, setExporting] = useState<boolean>(false);

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

  const fetchTables = useCallback(async () => {
    setLoadingTables(true);
    try {
      const data = await services.dbManage.listTables();
      setTables(data);
    } catch (err) {
      toast.error(t('fetchTablesFailed'), {
        description: err instanceof Error ? err.message : t('unknownError'),
      });
    } finally {
      setLoadingTables(false);
    }
  }, [t]);

  const handleRefreshAll = () => {
    fetchOverview();
    fetchTables();
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
    fetchOverview();
    fetchTables();
  }, [fetchOverview, fetchTables]);

  const renderOverviewSkeleton = () => (
    <div className='grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4'>
      {Array.from({ length: 6 }).map((_, i) => (
        <Card key={i} className='border-border/40 bg-card/50'>
          <CardHeader className='pb-2'>
            <Skeleton className='h-4 w-16' />
          </CardHeader>
          <CardContent>
            <Skeleton className='h-6 w-24' />
          </CardContent>
        </Card>
      ))}
    </div>
  );

  return (
    <div className='flex w-full flex-col gap-6 py-6 px-1'>
      <div className='flex flex-col sm:flex-row sm:items-center sm:justify-between pb-4 gap-4'>
        <div className='flex items-center gap-2'>
          <Database className='size-5 text-primary' />
          <div>
            <h1 className='text-2xl font-semibold tracking-tight'>
              {t('pageTitle')}
            </h1>
          </div>
        </div>
        <Button
          size='sm'
          variant='secondary'
          className='h-8 gap-1.5 text-xs self-start sm:self-auto'
          onClick={handleRefreshAll}
          disabled={loadingOverview}
        >
          <RefreshCw
            className={`size-3 ${loadingOverview ? 'animate-spin' : ''}`}
          />
          {t('refreshData')}
        </Button>
      </div>

      <Tabs defaultValue='overview' className='w-full'>
        <TabsList variant='line'>
          <TabsTrigger value='overview'>{t('dbType')}</TabsTrigger>
          <TabsTrigger value='tables'>{t('sections.tables')}</TabsTrigger>
          <TabsTrigger value='cache'>{t('sections.cache')}</TabsTrigger>
          <TabsTrigger value='maintenance'>{t('sections.maintenance')}</TabsTrigger>
          <TabsTrigger value='storage'>{t('dbBackupExport')}</TabsTrigger>
        </TabsList>
        <TabsContent value='overview' className='flex flex-col gap-6'>
          {loadingOverview
            ? renderOverviewSkeleton()
            : overview && (
            <div className='grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4'>
              <Card className='shadow-sm border-border/40 bg-card/50 backdrop-blur-sm hover:border-primary/20 transition-all duration-300'>
                <CardHeader className='flex flex-row items-center justify-between pb-2 space-y-0'>
                  <CardDescription className='text-[10px] font-medium'>
                    {t('dbType')}
                  </CardDescription>
                  <Server className='size-3.5 text-muted-foreground' />
                </CardHeader>
                <CardContent>
                  <div className='text-sm font-bold uppercase'>
                    {overview.type}
                  </div>
                </CardContent>
              </Card>

              <Card className='shadow-sm border-border/40 bg-card/50 backdrop-blur-sm hover:border-primary/20 transition-all duration-300 col-span-1 md:col-span-2 lg:col-span-1'>
                <CardHeader className='flex flex-row items-center justify-between pb-2 space-y-0'>
                  <CardDescription className='text-[10px] font-medium'>
                    {t('versionInfo')}
                  </CardDescription>
                  <Cpu className='size-3.5 text-muted-foreground' />
                </CardHeader>
                <CardContent>
                  <div
                    className='text-xs font-semibold truncate'
                    title={overview.version}
                  >
                    {overview.version.split(' ').slice(0, 2).join(' ')}
                  </div>
                </CardContent>
              </Card>

              <Card className='shadow-sm border-border/40 bg-card/50 backdrop-blur-sm hover:border-primary/20 transition-all duration-300'>
                <CardHeader className='flex flex-row items-center justify-between pb-2 space-y-0'>
                  <CardDescription className='text-[10px] font-medium'>
                    {t('nameOrPath')}
                  </CardDescription>
                  <Database className='size-3.5 text-muted-foreground' />
                </CardHeader>
                <CardContent>
                  <div
                    className='text-xs font-semibold truncate'
                    title={overview.name}
                  >
                    {overview.name.substring(
                      overview.name.lastIndexOf('/') + 1,
                    )}
                  </div>
                </CardContent>
              </Card>

              <Card className='shadow-sm border-border/40 bg-card/50 backdrop-blur-sm hover:border-primary/20 transition-all duration-300'>
                <CardHeader className='flex flex-row items-center justify-between pb-2 space-y-0'>
                  <CardDescription className='text-[10px] font-medium'>
                    {t('dbSize')}
                  </CardDescription>
                  <HardDrive className='size-3.5 text-muted-foreground' />
                </CardHeader>
                <CardContent>
                  <div className='text-sm font-bold'>{overview.size}</div>
                </CardContent>
              </Card>

              <Card className='shadow-sm border-border/40 bg-card/50 backdrop-blur-sm hover:border-primary/20 transition-all duration-300'>
                <CardHeader className='flex flex-row items-center justify-between pb-2 space-y-0'>
                  <CardDescription className='text-[10px] font-medium'>
                    {t('physicalTables')}
                  </CardDescription>
                  <Layers className='size-3.5 text-muted-foreground' />
                </CardHeader>
                <CardContent>
                  <div className='text-sm font-bold'>
                    {formatNumber(overview.table_count)}
                  </div>
                </CardContent>
              </Card>

              <Card className='shadow-sm border-border/40 bg-card/50 backdrop-blur-sm hover:border-primary/20 transition-all duration-300'>
                <CardHeader className='flex flex-row items-center justify-between pb-2 space-y-0'>
                  <CardDescription className='text-[10px] font-medium'>
                    {t('activeConnections')}
                  </CardDescription>
                  <Activity className='size-3.5 text-muted-foreground' />
                </CardHeader>
                <CardContent>
                  <div className='text-sm font-bold'>
                    {formatNumber(overview.connections)}
                  </div>
                </CardContent>
              </Card>
            </div>
            )}
        </TabsContent>
        <TabsContent value='tables' className='flex flex-col gap-6'>
          <TableBrowser tables={tables} loadingTables={loadingTables} refreshTrigger={refreshTrigger} />
        </TabsContent>
        <TabsContent value='cache'>
          <CacheManager refreshTrigger={refreshTrigger} />
        </TabsContent>
        <TabsContent value='maintenance'>
          <MaintenanceCard />
        </TabsContent>
        <TabsContent value='storage'>
          <Card>
            <CardHeader>
              <CardTitle className='flex items-center gap-2'>
                <Download className='size-5 text-primary' />
                {t('dbBackupExport')}
              </CardTitle>
              <CardDescription>{t('dbBackupExportDesc')}</CardDescription>
            </CardHeader>
            <CardContent>
              <Button size='sm' variant='outline' onClick={handleExport} disabled={exporting}>
                <Download data-icon='inline-start' />
                {exporting ? t('preparing') : t('startExport')}
              </Button>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
