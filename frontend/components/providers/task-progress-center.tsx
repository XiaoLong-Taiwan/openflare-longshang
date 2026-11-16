'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { usePathname } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { Activity, ChevronDown, ChevronUp, X } from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet';
import services from '@/lib/services';
import type { TaskExecution, TaskExecutionStatus } from '@/lib/services/admin';
import { useTranslations } from 'next-intl';

import { useUser } from '@/contexts/user-context';
import { useIsMobile } from '@/hooks/use-mobile';

const activeStatuses: TaskExecutionStatus[] = ['pending', 'running'];

function statusVariant(status: TaskExecutionStatus) {
  return status === 'running' ? 'default' : 'outline';
}

function TaskLog({ execution }: { execution: TaskExecution }) {
  const t = useTranslations('admin.tasks');
  const detailQuery = useQuery({
    queryKey: ['admin', 'task-execution', execution.id, 'global'],
    queryFn: () => services.adminTask.getTaskExecution(execution.id),
    refetchInterval: execution.status === 'running' ? 3000 : false,
  });
  const detail = detailQuery.data ?? execution;

  return (
    <div className='flex min-h-0 flex-1 flex-col gap-3 overflow-hidden px-4 pb-4'>
      <div className='grid grid-cols-2 gap-3'>
        <div className='rounded-lg border p-3'>
          <div className='text-xs text-muted-foreground'>
            {t('detailStatus')}
          </div>
          <Badge className='mt-2' variant={statusVariant(detail.status)}>
            {detail.status === 'running'
              ? t('statusRunning')
              : t('statusPending')}
          </Badge>
        </div>
        <div className='rounded-lg border p-3'>
          <div className='text-xs text-muted-foreground'>
            {t('detailTaskId')}
          </div>
          <div className='mt-2 truncate font-mono text-xs'>
            {detail.task_id}
          </div>
        </div>
      </div>
      <div className='flex min-h-0 flex-1 flex-col gap-2'>
        <div className='flex items-center justify-between text-xs font-medium'>
          <span>{t('detailLog')}</span>
          {detailQuery.isFetching ? (
            <span className='text-muted-foreground'>{t('refreshing')}</span>
          ) : null}
        </div>
        <pre className='min-h-48 flex-1 overflow-auto rounded-md border bg-muted/40 p-3 text-xs leading-relaxed whitespace-pre-wrap'>
          {detail.log || t('noLog')}
        </pre>
      </div>
    </div>
  );
}

export function TaskProgressCenter() {
  const t = useTranslations('admin.tasks');
  const pathname = usePathname();
  const { user, loading: userLoading } = useUser();
  const isMobile = useIsMobile();
  const [collapsed, setCollapsed] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const [selected, setSelected] = useState<TaskExecution | null>(null);

  const floatingRef = useRef<HTMLElement | null>(null);
  const enabled =
    !userLoading && !!user && pathname !== '/login' && pathname !== '/register';

  const pendingQuery = useQuery({
    queryKey: ['admin', 'task-executions', 'global', 'pending'],
    queryFn: () =>
      services.adminTask.listTaskExecutions({
        status: 'pending',
        page: 1,
        page_size: 20,
      }),
    refetchInterval: 3000,
    enabled,
  });
  const runningQuery = useQuery({
    queryKey: ['admin', 'task-executions', 'global', 'running'],
    queryFn: () =>
      services.adminTask.listTaskExecutions({
        status: 'running',
        page: 1,
        page_size: 20,
      }),
    refetchInterval: 3000,
    enabled,
  });

  const executions = useMemo(
    () =>
      Array.from(
        new Map(
          [
            ...(runningQuery.data?.items ?? []),
            ...(pendingQuery.data?.items ?? []),
          ]
            .filter((item) => activeStatuses.includes(item.status))
            .map((item) => [item.task_id, item]),
        ).values(),
      ),
    [pendingQuery.data?.items, runningQuery.data?.items],
  );

  useEffect(() => {
    const element = floatingRef.current;
    if (!element || !executions.length) return;
    const updateInset = () => {
      document.documentElement.style.setProperty(
        '--task-progress-inset',
        `${element.getBoundingClientRect().height + 32}px`,
      );
    };
    updateInset();
    const observer = new ResizeObserver(updateInset);
    observer.observe(element);
    return () => {
      observer.disconnect();
      document.documentElement.style.removeProperty('--task-progress-inset');
    };
  }, [executions.length, isMobile, dismissed, collapsed]);

  if (executions.length === 0) {
    return null;
  }

  if (dismissed) {
    return (
      <Button
        ref={(node) => {
          floatingRef.current = node;
        }}
        variant='outline'
        size='sm'
        className='task-progress-floating fixed z-40 shadow-lg'
        onClick={() => setDismissed(false)}
      >
        <Activity data-icon='inline-start' />
        {t('activeTasks')}
        <Badge variant='secondary'>{executions.length}</Badge>
      </Button>
    );
  }

  if (isMobile) {
    return (
      <>
        <Sheet
          open={mobileOpen}
          onOpenChange={(open) => {
            setMobileOpen(open);
            if (!open) setSelected(null);
          }}
        >
          <SheetTrigger asChild>
            <Button
              ref={(node) => {
                floatingRef.current = node;
              }}
              variant='outline'
              size='sm'
              className='task-progress-floating fixed z-40 shadow-lg'
            >
              <Activity data-icon='inline-start' />
              {t('activeTasks')}
              <Badge variant='secondary'>{executions.length}</Badge>
            </Button>
          </SheetTrigger>
          <SheetContent
            id='mobile-active-tasks'
            side='bottom'
            className='task-progress-sheet h-[min(70dvh,560px)] gap-0 rounded-t-lg'
          >
            <SheetHeader className='border-b'>
              <SheetTitle>{t('activeTasks')}</SheetTitle>
              <SheetDescription>{t('activeTaskDesc')}</SheetDescription>
            </SheetHeader>
            {selected ? (
              <TaskLog execution={selected} />
            ) : (
              <div className='min-h-0 flex-1 overflow-auto divide-y'>
                {executions.map((execution) => (
                  <button
                    key={execution.id}
                    type='button'
                    className='flex w-full flex-col gap-2 px-4 py-3 text-left hover:bg-muted/40'
                    onClick={() => {
                      setSelected(execution);
                    }}
                  >
                    <div className='flex items-center justify-between gap-3'>
                      <span className='truncate text-sm font-medium'>
                        {execution.task_name || execution.task_type}
                      </span>
                      <Badge variant={statusVariant(execution.status)}>
                        {execution.status === 'running'
                          ? t('statusRunning')
                          : t('statusPending')}
                      </Badge>
                    </div>
                    <Progress
                      value={execution.status === 'running' ? undefined : 8}
                      className='h-1.5'
                    />
                    <div className='flex items-center justify-between gap-3 text-[11px] text-muted-foreground'>
                      <span className='truncate font-mono'>
                        {execution.task_id}
                      </span>
                      <span>{t('viewLog')}</span>
                    </div>
                  </button>
                ))}
              </div>
            )}
          </SheetContent>
        </Sheet>
      </>
    );
  }

  return (
    <>
      <section
        ref={floatingRef}
        className='task-progress-floating fixed z-40 hidden max-h-[calc(100dvh-2rem-env(safe-area-inset-top)-env(safe-area-inset-bottom))] w-[min(420px,calc(100vw-2rem))] overflow-auto rounded-lg border bg-background shadow-lg md:block'
      >
        <div className='flex items-center justify-between gap-3 border-b px-3 py-2'>
          <div className='flex min-w-0 items-center gap-2'>
            <Activity className='size-4 shrink-0 text-primary' />
            <span className='truncate text-sm font-semibold'>
              {t('activeTasks')}
            </span>
            <Badge variant='secondary'>{executions.length}</Badge>
          </div>
          <div className='flex items-center gap-1'>
            <Button
              variant='ghost'
              size='icon'
              className='size-7'
              aria-label={collapsed ? t('expandTasks') : t('collapseTasks')}
              title={collapsed ? t('expandTasks') : t('collapseTasks')}
              onClick={() => setCollapsed(!collapsed)}
            >
              {collapsed ? <ChevronUp /> : <ChevronDown />}
            </Button>
            <Button
              variant='ghost'
              size='icon'
              className='size-7'
              aria-label={t('dismissTasks')}
              title={t('dismissTasks')}
              onClick={() => setDismissed(true)}
            >
              <X />
            </Button>
          </div>
        </div>
        {!collapsed ? (
          <div className='max-h-[300px] overflow-auto divide-y'>
            {executions.map((execution) => (
              <button
                key={execution.id}
                type='button'
                className='flex w-full flex-col gap-2 px-3 py-3 text-left hover:bg-muted/40'
                onClick={() => setSelected(execution)}
              >
                <div className='flex items-center justify-between gap-3'>
                  <span className='truncate text-sm font-medium'>
                    {execution.task_name || execution.task_type}
                  </span>
                  <Badge variant={statusVariant(execution.status)}>
                    {execution.status === 'running'
                      ? t('statusRunning')
                      : t('statusPending')}
                  </Badge>
                </div>
                <Progress
                  value={execution.status === 'running' ? undefined : 8}
                  className='h-1.5'
                />
                <div className='flex items-center justify-between gap-3 text-[11px] text-muted-foreground'>
                  <span className='truncate font-mono'>
                    {execution.task_id}
                  </span>
                  <span>{t('viewLog')}</span>
                </div>
              </button>
            ))}
          </div>
        ) : null}
      </section>

      <Sheet
        open={selected !== null}
        onOpenChange={(open) => !open && setSelected(null)}
      >
        <SheetContent className='flex w-full flex-col p-0 sm:max-w-[640px]'>
          <SheetHeader className='border-b'>
            <SheetTitle>
              {selected?.task_name || selected?.task_type || t('activeTask')}
            </SheetTitle>
            <SheetDescription>{t('activeTaskDesc')}</SheetDescription>
          </SheetHeader>
          {selected ? <TaskLog execution={selected} /> : null}
        </SheetContent>
      </Sheet>
    </>
  );
}
