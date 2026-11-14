'use client';

import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';

import { Badge } from '@/components/ui/badge';
import { ErrorInline } from '@/components/layout/error';
import { LoadingStateWithBorder } from '@/components/layout/loading';
import services from '@/lib/services';

const TASK_TYPE = 'of_ssl_single_renew';

export function CertificateTaskTracking() {
  const t = useTranslations('certificates');
  const taskQuery = useQuery({
    queryKey: ['certificates', 'renewal-executions'],
    queryFn: () =>
      services.adminTask.listTaskExecutions({
        task_type: TASK_TYPE,
        page: 1,
        page_size: 20,
      }),
    refetchInterval: (query) =>
      query.state.data?.items.some(
        (task) => task.status === 'pending' || task.status === 'running',
      )
        ? 5000
        : false,
  });
  const executions = useMemo(
    () => taskQuery.data?.items ?? [],
    [taskQuery.data],
  );

  return (
    <section className='space-y-3'>
      <h2 className='text-lg font-semibold'>{t('renewalTasks')}</h2>
      {taskQuery.isPending ? (
        <LoadingStateWithBorder description={t('renewalTasksLoading')} />
      ) : taskQuery.isError ? (
        <ErrorInline
          message={taskQuery.error.message || t('renewalTasksFailed')}
          onRetry={() => void taskQuery.refetch()}
        />
      ) : executions.length === 0 ? (
        <p className='text-sm text-muted-foreground'>
          {t('renewalTasksEmpty')}
        </p>
      ) : (
        <div className='divide-y rounded-md border'>
          {executions.map((execution) => {
            let certificateID = '';
            try {
              certificateID = String(
                (JSON.parse(execution.payload) as { id?: number }).id ?? '',
              );
            } catch {
              certificateID = '';
            }
            return (
              <div
                key={execution.id}
                className='flex flex-wrap items-center justify-between gap-3 p-3 text-sm'
              >
                <div className='min-w-0'>
                  <p className='font-medium'>
                    {t('renewalTaskCertificate', { id: certificateID || '-' })}
                  </p>
                  <p className='text-muted-foreground'>
                    {execution.created_at}
                  </p>
                  {execution.error_message ? (
                    <p className='mt-1 break-words text-destructive'>
                      {execution.error_message}
                    </p>
                  ) : null}
                </div>
                <Badge
                  variant={
                    execution.status === 'failed' ? 'destructive' : 'outline'
                  }
                >
                  {t(`renewalStatus.${execution.status}`)}
                </Badge>
              </div>
            );
          })}
        </div>
      )}
    </section>
  );
}
