'use client';

import * as React from 'react';
import {
  AlertTriangle,
  Database,
  Eraser,
  Loader2,
  RefreshCw,
} from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog';
import services, { formatFileSize } from '@/lib/services';

export function LogStorage() {
  const t = useTranslations('admin.logs.storage');
  const queryClient = useQueryClient();
  const [days, setDays] = React.useState('');
  const [forceDialogOpen, setForceDialogOpen] = React.useState(false);
  const query = useQuery({
    queryKey: ['admin', 'logs', 'storage'],
    queryFn: () => services.adminLog.getLogStorage(),
  });
  const cleanup = useMutation({
    mutationFn: (category: string) => {
      const message = days ? t('clearBeforeDays', { days }) : t('clearAll');
      if (!window.confirm(message)) {
        return Promise.reject(new Error('Cleanup cancelled'));
      }
      return services.adminLog.cleanupLogStorage({
        category,
        days: days.trim() ? Number(days) : undefined,
      });
    },
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ['admin', 'logs', 'storage'] }),
  });
  const forceCleanup = useMutation({
    mutationFn: () => services.adminLog.forceCleanupOldLogStorage(),
    onSuccess: (taskID) => {
      setForceDialogOpen(false);
      queryClient.invalidateQueries({ queryKey: ['admin', 'logs', 'storage'] });
      toast.success(t('forceCleanupQueued', { taskID }));
    },
    onError: (error) =>
      toast.error(t('loadFailed'), {
        description: error instanceof Error ? error.message : undefined,
      }),
  });

  if (query.isLoading) {
    return <Loader2 className='size-6 animate-spin text-muted-foreground' />;
  }
  if (!query.data) {
    return <p className='text-sm text-muted-foreground'>{t('loadFailed')}</p>;
  }

  return (
    <div className='flex flex-col gap-6'>
      <div className='flex items-center justify-between gap-4'>
        <div>
          <p className='text-sm text-muted-foreground'>
            {t('database', { name: query.data.database })}
          </p>
          <p className='text-2xl font-semibold'>
            {formatFileSize(query.data.total_bytes)}
          </p>
          <p className='text-xs text-muted-foreground'>
            {query.data.total_rows.toLocaleString()} {t('rows')}
          </p>
        </div>
        <Button
          variant='outline'
          size='sm'
          onClick={() => query.refetch()}
          disabled={query.isFetching}
        >
          <RefreshCw data-icon='inline-start' />
          {t('refresh')}
        </Button>
        <AlertDialog open={forceDialogOpen} onOpenChange={setForceDialogOpen}>
          <AlertDialogTrigger asChild>
            <Button
              variant='destructive'
              size='sm'
              disabled={forceCleanup.isPending || cleanup.isPending}
            >
              <AlertTriangle data-icon='inline-start' />
              {t('forceCleanup')}
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>{t('forceCleanup')}</AlertDialogTitle>
              <AlertDialogDescription>
                {t('forceCleanupConfirm')}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel disabled={forceCleanup.isPending}>
                {t('cancel')}
              </AlertDialogCancel>
              <Button
                variant='destructive'
                onClick={() => forceCleanup.mutate()}
                disabled={forceCleanup.isPending}
              >
                {forceCleanup.isPending ? (
                  <Loader2 data-icon='inline-start' className='animate-spin' />
                ) : null}
                {t('forceCleanup')}
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>

      <div className='flex flex-wrap items-end gap-3 border-b pb-5'>
        <div className='flex flex-col gap-1'>
          <label
            htmlFor='log-retention-days'
            className='text-xs text-muted-foreground'
          >
            {days ? t('daysLabel', { days }) : t('daysPlaceholder')}
          </label>
          <Input
            id='log-retention-days'
            type='number'
            min={1}
            value={days}
            onChange={(event) => setDays(event.target.value)}
            placeholder={t('daysPlaceholder')}
            className='w-40'
          />
        </div>
        <Button
          variant='destructive'
          size='sm'
          onClick={() => cleanup.mutate('all')}
          disabled={cleanup.isPending || (days !== '' && Number(days) <= 0)}
        >
          {cleanup.isPending ? (
            <Loader2 data-icon='inline-start' className='animate-spin' />
          ) : (
            <Eraser data-icon='inline-start' />
          )}
          {days ? t('clearBeforeDays', { days }) : t('clearAll')}
        </Button>
      </div>

      <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-3'>
        {query.data.categories.map((category) => (
          <Card key={category.key}>
            <CardHeader className='flex flex-row items-center justify-between gap-3'>
              <CardTitle className='text-sm'>
                {t(`categories.${category.key}`)}
              </CardTitle>
              <Database className='size-4 text-muted-foreground' />
            </CardHeader>
            <CardContent className='flex items-end justify-between gap-3'>
              <div>
                <p className='font-mono text-lg'>
                  {formatFileSize(category.bytes)}
                </p>
                <p className='text-xs text-muted-foreground'>
                  {category.rows.toLocaleString()} {t('rows')}
                </p>
              </div>
              {category.clearable ? (
                <Button
                  variant='ghost'
                  size='sm'
                  onClick={() => cleanup.mutate(category.key)}
                  disabled={
                    cleanup.isPending || (days !== '' && Number(days) <= 0)
                  }
                >
                  <Eraser data-icon='inline-start' />
                  {t('clear')}
                </Button>
              ) : null}
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
