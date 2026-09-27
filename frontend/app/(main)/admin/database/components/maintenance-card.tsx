'use client';

import { useCallback, useEffect, useState } from 'react';
import { AlertTriangle, Loader2, Wrench } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';

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
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import services from '@/lib/services';
import type { DatabaseMaintenancePreview } from '@/lib/services/db-manage';

export function MaintenanceCard() {
  const t = useTranslations('admin.database');
  const [preview, setPreview] = useState<DatabaseMaintenancePreview | null>(
    null,
  );
  const [loading, setLoading] = useState(true);
  const [running, setRunning] = useState(false);

  const loadPreview = useCallback(async () => {
    setLoading(true);
    try {
      setPreview(await services.dbManage.getMaintenancePreview());
    } catch (error) {
      toast.error(t('maintenance.previewFailed'), {
        description: error instanceof Error ? error.message : t('unknownError'),
      });
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void loadPreview();
  }, [loadPreview]);

  const runMaintenance = async () => {
    setRunning(true);
    try {
      const taskID = await services.dbManage.runMaintenance();
      toast.success(t('maintenance.queued'), {
        description: t('maintenance.queuedDesc', { taskID }),
      });
    } catch (error) {
      toast.error(t('maintenance.failed'), {
        description: error instanceof Error ? error.message : t('unknownError'),
      });
    } finally {
      setRunning(false);
    }
  };

  return (
    <Card className='border-border/40 bg-card/50 backdrop-blur-sm shadow-sm'>
      <CardHeader className='pb-3 border-b border-dashed'>
        <CardTitle className='flex items-center gap-2 text-sm font-semibold'>
          <Wrench className='size-4 text-primary' />
          {t('maintenance.title')}
        </CardTitle>
        <CardDescription className='text-[11px]'>
          {t('maintenance.description')}
        </CardDescription>
      </CardHeader>
      <CardContent className='flex flex-col gap-4 pt-4'>
        {loading ? (
          <p className='text-sm text-muted-foreground'>
            {t('maintenance.loading')}
          </p>
        ) : preview ? (
          <div className='grid grid-cols-1 gap-2 text-sm md:grid-cols-3'>
            <p>
              {t('maintenance.expiredTasks', {
                count: preview.expired_task_executions,
              })}
            </p>
            <p>{t('maintenance.cacheItems', { count: preview.cache_keys })}</p>
            <p>
              {t('maintenance.databaseType', { type: preview.database_type })}
            </p>
          </div>
        ) : null}
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button
              variant='destructive'
              className='self-start'
              disabled={loading || running}
            >
              <AlertTriangle data-icon='inline-start' />
              {t('maintenance.run')}
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>
                {t('maintenance.confirmTitle')}
              </AlertDialogTitle>
              <AlertDialogDescription>
                {t('maintenance.confirmDescription')}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel disabled={running}>
                {t('maintenance.cancel')}
              </AlertDialogCancel>
              <Button
                variant='destructive'
                onClick={() => void runMaintenance()}
                disabled={running}
              >
                {running ? (
                  <Loader2 className='animate-spin' data-icon='inline-start' />
                ) : null}
                {running ? t('maintenance.running') : t('maintenance.confirm')}
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </CardContent>
    </Card>
  );
}
