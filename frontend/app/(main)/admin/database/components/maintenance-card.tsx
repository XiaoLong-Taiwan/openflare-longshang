'use client';

import { useCallback, useState } from 'react';
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
import { Checkbox } from '@/components/ui/checkbox';
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
  const [loading, setLoading] = useState(false);
  const [running, setRunning] = useState(false);
  const [selection, setSelection] = useState({
    cleanup_task_executions: true,
    clear_disk_cache: true,
    optimize_database: true,
  });

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

  const runMaintenance = async () => {
    setRunning(true);
    try {
      const taskID = await services.dbManage.runMaintenance(selection);
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
    <Card className='min-w-0 gap-4'>
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
        <Button
          variant='outline'
          className='h-auto min-h-9 self-start whitespace-normal'
          disabled={loading || running}
          onClick={() => void loadPreview()}
        >
          {t('maintenance.loadPreview')}
        </Button>
        {loading ? (
          <p className='text-sm text-muted-foreground'>
            {t('maintenance.loading')}
          </p>
        ) : preview ? (
          <div className='flex flex-col gap-3 text-sm'>
            <p>
              {t('maintenance.databaseType', { type: preview.database_type })}
            </p>
            <label className='flex items-start gap-2'>
              <Checkbox
                checked={selection.cleanup_task_executions}
                onCheckedChange={(checked) =>
                  setSelection((prev) => ({
                    ...prev,
                    cleanup_task_executions: checked === true,
                  }))
                }
              />
              <span>
                {t('maintenance.expiredTasks', {
                  count: preview.expired_task_executions,
                })}
              </span>
            </label>
            <label className='flex items-start gap-2'>
              <Checkbox
                checked={selection.clear_disk_cache}
                onCheckedChange={(checked) =>
                  setSelection((prev) => ({
                    ...prev,
                    clear_disk_cache: checked === true,
                  }))
                }
              />
              <span>
                {t('maintenance.cacheItems', { count: preview.cache_keys })}
              </span>
            </label>
            <label className='flex items-start gap-2'>
              <Checkbox
                checked={selection.optimize_database}
                onCheckedChange={(checked) =>
                  setSelection((prev) => ({
                    ...prev,
                    optimize_database: checked === true,
                  }))
                }
              />
              <span>{t('maintenance.optimizeDatabase')}</span>
            </label>
          </div>
        ) : null}
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button
              variant='destructive'
              className='h-auto min-h-9 self-start whitespace-normal'
              disabled={
                !preview ||
                loading ||
                running ||
                !Object.values(selection).some(Boolean)
              }
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
                <span className='block'>
                  {t('maintenance.confirmDescription')}
                </span>
                <span className='mt-2 block'>
                  {selection.cleanup_task_executions
                    ? `${t('maintenance.expiredTasks', { count: preview?.expired_task_executions ?? 0 })}; `
                    : ''}
                  {selection.clear_disk_cache
                    ? `${t('maintenance.cacheItems', { count: preview?.cache_keys ?? 0 })}; `
                    : ''}
                  {selection.optimize_database
                    ? t('maintenance.optimizeDatabase')
                    : ''}
                </span>
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel disabled={running}>
                {t('maintenance.cancel')}
              </AlertDialogCancel>
              <Button
                variant='destructive'
                onClick={() => void runMaintenance()}
                disabled={running || !Object.values(selection).some(Boolean)}
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
