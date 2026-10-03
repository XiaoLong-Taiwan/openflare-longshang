'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Switch } from '@/components/ui/switch';
import { Spinner } from '@/components/ui/spinner';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Calendar as CalendarIcon,
  Clock,
  Database,
  Info,
  Layers,
  Play,
} from 'lucide-react';

import type {
  DispatchTaskRequest,
  LogDatabaseStatus,
  TaskMeta,
} from '@/lib/services/admin';
import services from '@/lib/services';
import { buildTaskPayload } from '@/lib/task-param-utils';
import { ErrorInline } from '@/components/layout/error';
import { LoadingStateWithBorder } from '@/components/layout/loading';
import { EmptyStateWithBorder } from '@/components/layout/empty';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';

import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import { Calendar } from '@/components/ui/calendar';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';

const TASK_CONFIGS: Record<
  string,
  {
    icon: React.ComponentType<{ className?: string }>;
    color: string;
    gradient: string;
  }
> = {
  order_sync: {
    icon: Layers,
    color: 'text-blue-600 dark:text-blue-400',
    gradient:
      'from-blue-500/10 via-blue-500/5 to-transparent border-blue-200/50 dark:border-blue-800/50 hover:border-blue-400 dark:hover:border-blue-500',
  },
  user_gamification: {
    icon: Play,
    color: 'text-amber-600 dark:text-amber-400',
    gradient:
      'from-amber-500/10 via-amber-500/5 to-transparent border-amber-200/50 dark:border-amber-800/50 hover:border-amber-400 dark:hover:border-amber-500',
  },
  dispute_auto_refund: {
    icon: Clock,
    color: 'text-rose-600 dark:text-rose-400',
    gradient:
      'from-rose-500/10 via-rose-500/5 to-transparent border-rose-200/50 dark:border-rose-800/50 hover:border-rose-400 dark:hover:border-rose-500',
  },
  of_log_db_switch: {
    icon: Database,
    color: 'text-teal-600 dark:text-teal-400',
    gradient:
      'from-teal-500/10 via-teal-500/5 to-transparent border-teal-200/50 dark:border-teal-800/50 hover:border-teal-400 dark:hover:border-teal-500',
  },
};

const LOG_DATABASE_LABEL_KEYS: Record<string, string> = {
  postgres: 'dbPostgresPrimary',
  sqlite: 'dbSqlitePrimary',
  clickhouse: 'dbClickhouse',
};

const DEFAULT_TASK_CONFIG = {
  icon: Layers,
  color: 'text-zinc-600 dark:text-zinc-400',
  gradient:
    'from-zinc-500/10 via-zinc-500/5 to-transparent border-zinc-200/50 dark:border-zinc-800/50 hover:border-zinc-400 dark:hover:border-zinc-500',
};

function DatePickerWithTime({
  date,
  setDate,
}: {
  date: Date | undefined;
  setDate: (date: Date | undefined) => void;
}) {
  const t = useTranslations('admin.tasks');
  const timeString = date ? format(date, 'HH:mm:ss') : '';

  const handleDateSelect = (newDate: Date | undefined) => {
    if (!newDate) {
      setDate(undefined);
      return;
    }

    const updated = new Date(newDate);

    if (date) {
      updated.setHours(date.getHours());
      updated.setMinutes(date.getMinutes());
      updated.setSeconds(date.getSeconds());
    } else {
      updated.setHours(0, 0, 0, 0);
    }

    setDate(updated);
  };

  const handleTimeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newTime = e.target.value;
    if (!newTime) return;

    if (date) {
      const [hours, minutes, seconds] = newTime.split(':').map(Number);
      const updated = new Date(date);
      updated.setHours(hours || 0);
      updated.setMinutes(minutes || 0);
      updated.setSeconds(seconds || 0);
      setDate(updated);
    } else {
      const today = new Date();
      const [hours, minutes, seconds] = newTime.split(':').map(Number);
      today.setHours(hours || 0);
      today.setMinutes(minutes || 0);
      today.setSeconds(seconds || 0);
      setDate(today);
    }
  };

  return (
    <div className='flex items-center gap-2'>
      <div className='flex-1'>
        <Popover>
          <PopoverTrigger asChild>
            <Button
              variant={'outline'}
              className={cn(
                'w-full justify-start text-left font-normal text-xs h-8',
                !date && 'text-muted-foreground',
              )}
            >
              <CalendarIcon className='mr-1 size-3' />
              {date ? (
                format(date, 'yyyy-MM-dd')
              ) : (
                <span>{t('selectDate')}</span>
              )}
            </Button>
          </PopoverTrigger>
          <PopoverContent className='w-auto p-0' align='start'>
            <Calendar
              mode='single'
              selected={date}
              onSelect={handleDateSelect}
              locale={zhCN}
            />
          </PopoverContent>
        </Popover>
      </div>
      <div className='w-[120px]'>
        <Input
          type='time'
          step='1'
          value={timeString}
          onChange={handleTimeChange}
          className='text-xs font-mono'
        />
      </div>
    </div>
  );
}

export function TaskManager() {
  const t = useTranslations('admin.tasks');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [taskTypes, setTaskTypes] = useState<TaskMeta[]>([]);

  const [dispatching, setDispatching] = useState(false);
  const [selectedTaskType, setSelectedTaskType] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [detailTask, setDetailTask] = useState<TaskMeta | null>(null);

  const [startTime, setStartTime] = useState<Date | undefined>(undefined);
  const [endTime, setEndTime] = useState<Date | undefined>(undefined);
  const [userId, setUserId] = useState('');
  const [paramValues, setParamValues] = useState<Record<string, string>>({});

  const fetchTaskTypes = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await services.adminTask.getTaskTypes();
      setTaskTypes(data);
    } catch (err) {
      setError(
        err instanceof Error ? err : new Error(t('loadTaskTypesFailed')),
      );
    } finally {
      setLoading(false);
    }
  }, [t]);

  const [logDbStatus, setLogDbStatus] = useState<LogDatabaseStatus | null>(
    null,
  );

  // 日志库状态用于「切换日志数据库」卡片与 target 下拉；获取失败不阻塞任务列表。
  const fetchLogDbStatus = useCallback(async () => {
    try {
      const data = await services.adminStatus.getLogDatabaseStatus();
      setLogDbStatus(data);
    } catch {
      setLogDbStatus(null);
    }
  }, []);

  useEffect(() => {
    fetchTaskTypes();
    fetchLogDbStatus();
  }, [fetchTaskTypes, fetchLogDbStatus]);

  const availableLogDbTargets = useMemo(
    () => logDbStatus?.available_targets ?? [],
    [logDbStatus],
  );

  useEffect(() => {
    if (selectedTaskType) {
      const targetTask = taskTypes.find((t) => t.type === selectedTaskType);
      if (targetTask?.params) {
        const initialValues: Record<string, string> = {};
        targetTask.params.forEach((p) => {
          initialValues[p.name] = '';
        });
        setParamValues(initialValues);
      } else {
        setParamValues({});
      }
    }
  }, [selectedTaskType, taskTypes]);

  const handleDispatch = async () => {
    if (!selectedTaskType) return;

    try {
      setDispatching(true);

      const targetTask = taskTypes.find((t) => t.type === selectedTaskType);

      const params: DispatchTaskRequest = {
        task_type: selectedTaskType,
      };

      if (targetTask?.supports_time) {
        if (startTime) params.start_time = startTime.toISOString();
        if (endTime) params.end_time = endTime.toISOString();
      }

      if (targetTask?.type === 'user_gamification') {
        if (userId) params.user_id = userId;
      }

      // Handle dynamic parameters — type coercion (number vs string) is
      // centralised in buildTaskPayload; do not inline it here.
      if (targetTask?.params && targetTask.params.length > 0) {
        const { payload, error } = buildTaskPayload(
          targetTask.params,
          paramValues,
        );
        if (error) {
          toast.error(error);
          setDispatching(false);
          return;
        }
        params.payload = payload ?? undefined;
      }

      const taskID = await services.adminTask.dispatchTask(params);

      toast.success(t('dispatchSuccess'), {
        description: t('dispatchSuccessDesc', {
          name: targetTask?.name || selectedTaskType,
          taskId: taskID,
        }),
      });
      setDialogOpen(false);

      setStartTime(undefined);
      setEndTime(undefined);
      setUserId('');
      setParamValues({});
    } catch (err) {
      toast.error(t('dispatchFailed'), {
        description: err instanceof Error ? err.message : t('unknownError'),
      });
    } finally {
      setDispatching(false);
    }
  };

  const openDispatchDialog = (type: string) => {
    setSelectedTaskType(type);
    setDialogOpen(true);
  };

  const getSelectedTaskMeta = () => {
    return taskTypes.find((t) => t.type === selectedTaskType);
  };

  return (
    <div className='space-y-6'>
      <br />

      <div className='space-y-6'>
        {error ? (
          <div className='p-8 border border-dashed rounded-xl bg-card'>
            <ErrorInline
              error={error}
              onRetry={fetchTaskTypes}
              className='justify-center'
            />
          </div>
        ) : loading && taskTypes.length === 0 ? (
          <LoadingStateWithBorder
            icon={Layers}
            description={t('loadingTaskTypes')}
          />
        ) : taskTypes.length === 0 ? (
          <EmptyStateWithBorder icon={Layers} description={t('noTaskTypes')} />
        ) : (
          <div className='overflow-x-auto rounded-lg border border-dashed'>
            <Table className='min-w-[820px]'>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('colTask')}</TableHead>
                  <TableHead>{t('taskTypeLabel')}</TableHead>
                  <TableHead>{t('colQueue')}</TableHead>
                  <TableHead>{t('colRetry')}</TableHead>
                  <TableHead>{t('colParameters')}</TableHead>
                  <TableHead>{t('colStatus')}</TableHead>
                  <TableHead className='w-[130px] text-right'>{t('colActions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {taskTypes.map((task) => {
                  const config = TASK_CONFIGS[task.type] || DEFAULT_TASK_CONFIG;
                  const Icon = config.icon;
                  const isLogSwitch = task.type === 'of_log_db_switch';
                  return (
                    <TableRow
                      key={task.type}
                      className='cursor-pointer'
                      onClick={() => setDetailTask(task)}
                    >
                      <TableCell>
                        <div className='flex min-w-[220px] items-center gap-2'>
                          <Icon className={cn('size-4 shrink-0', config.color)} />
                          <div className='min-w-0'>
                            <div className='truncate font-medium'>{task.name}</div>
                            <div className='truncate text-xs text-muted-foreground'>
                              {task.description || task.type}
                            </div>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className='font-mono text-xs text-muted-foreground'>
                        {task.type}
                      </TableCell>
                      <TableCell><Badge variant='outline'>{task.queue}</Badge></TableCell>
                      <TableCell className='font-mono text-xs'>{task.max_retry}</TableCell>
                      <TableCell className='text-xs text-muted-foreground'>
                        {task.params?.length ?? 0} {t('parameters')}
                        {task.supports_time ? ` · ${t('supportsTime')}` : ''}
                      </TableCell>
                      <TableCell>
                        {isLogSwitch && logDbStatus ? (
                          <Badge variant={logDbStatus.migration === 'migrating' ? 'default' : 'outline'}>
                            {logDbStatus.migration === 'migrating' ? t('migrating') : t('idle')}
                          </Badge>
                        ) : (
                          <Badge variant='secondary'>{t('ready')}</Badge>
                        )}
                      </TableCell>
                      <TableCell className='text-right'>
                        <Button
                          size='sm'
                          variant='outline'
                          onClick={(event) => {
                            event.stopPropagation();
                            openDispatchDialog(task.type);
                          }}
                        >
                          <Play data-icon='inline-start' />
                          {t('executeNow')}
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>
        )}
      </div>

      <Dialog
        open={detailTask !== null}
        onOpenChange={(open) => !open && setDetailTask(null)}
      >
        <DialogContent className='sm:max-w-[560px]'>
          <DialogHeader>
            <DialogTitle>{detailTask?.name || t('taskDetails')}</DialogTitle>
            <DialogDescription>
              {detailTask?.description || detailTask?.type || '-'}
            </DialogDescription>
          </DialogHeader>
          {detailTask ? (
            <div className='grid gap-3 py-2 sm:grid-cols-2'>
              <div className='rounded-lg border p-3'>
                <div className='text-xs text-muted-foreground'>{t('taskTypeLabel')}</div>
                <div className='mt-1 font-mono text-sm'>{detailTask.type}</div>
              </div>
              <div className='rounded-lg border p-3'>
                <div className='text-xs text-muted-foreground'>{t('colQueue')}</div>
                <div className='mt-1 font-mono text-sm'>{detailTask.queue}</div>
              </div>
              <div className='rounded-lg border p-3'>
                <div className='text-xs text-muted-foreground'>{t('colRetry')}</div>
                <div className='mt-1 font-mono text-sm'>{detailTask.max_retry}</div>
              </div>
              <div className='rounded-lg border p-3'>
                <div className='text-xs text-muted-foreground'>{t('colParameters')}</div>
                <div className='mt-1 text-sm'>{detailTask.params?.length ?? 0} {t('parameters')}</div>
              </div>
              <div className='rounded-lg border p-3 sm:col-span-2'>
                <div className='text-xs text-muted-foreground'>{t('executionSupport')}</div>
                <div className='mt-1 text-sm'>
                  {detailTask.supports_time ? t('supportsTime') : t('noTimeSupport')}
                </div>
              </div>
            </div>
          ) : null}
          <DialogFooter>
            <Button variant='outline' onClick={() => setDetailTask(null)}>
              {t('cancel')}
            </Button>
            <Button
              onClick={() => {
                if (detailTask) {
                  setDetailTask(null);
                  openDispatchDialog(detailTask.type);
                }
              }}
            >
              <Play data-icon='inline-start' />
              {t('executeNow')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className='sm:max-w-[500px]'>
          <DialogHeader>
            <DialogTitle>{t('dispatchTask')}</DialogTitle>
            <DialogDescription>{t('dispatchTaskDesc')}</DialogDescription>
          </DialogHeader>

          <div className='grid gap-4 py-4'>
            <div className='grid gap-2'>
              <Label>{t('executeTask')}</Label>
              <div className='flex items-center gap-2 p-2 rounded-lg bg-muted/50'>
                {(() => {
                  const meta = getSelectedTaskMeta();
                  if (!meta) return null;
                  const config = TASK_CONFIGS[meta.type] || DEFAULT_TASK_CONFIG;
                  const Icon = config.icon;
                  return (
                    <>
                      <div className={cn('p-2', config.color)}>
                        <Icon className='size-4' />
                      </div>
                      <div className='flex flex-col'>
                        <span className='text-xs font-medium'>{meta.name}</span>
                        <span className='text-[10px] text-muted-foreground font-mono'>
                          {meta.type}
                        </span>
                      </div>
                    </>
                  );
                })()}
              </div>
            </div>

            {(() => {
              const targetTask = getSelectedTaskMeta();
              if (targetTask?.supports_time) {
                return (
                  <>
                    <div className='grid gap-2'>
                      <Label>{t('startTime')}</Label>
                      <DatePickerWithTime
                        date={startTime}
                        setDate={setStartTime}
                      />
                      <p className='text-xs text-muted-foreground'>
                        {t('startTimeDesc')}
                      </p>
                    </div>
                    <div className='grid gap-2'>
                      <Label>{t('endTime')}</Label>
                      <DatePickerWithTime date={endTime} setDate={setEndTime} />
                      <p className='text-xs text-muted-foreground'>
                        {t('endTimeDesc')}
                      </p>
                    </div>
                  </>
                );
              }
              return null;
            })()}

            {(() => {
              const targetTask = getSelectedTaskMeta();
              if (
                !targetTask ||
                !targetTask.params ||
                targetTask.params.length === 0
              ) {
                return null;
              }
              return (
                <div className='space-y-4'>
                  {targetTask.params.map((param) => {
                    const isSwitchTarget =
                      param.name === 'target' &&
                      getSelectedTaskMeta()?.type === 'of_log_db_switch';
                    return (
                      <div key={param.name} className='grid gap-2'>
                        <Label
                          htmlFor={`param-${param.name}`}
                          className='flex items-center gap-1'
                        >
                          {param.label}
                          {param.required && (
                            <span className='text-destructive font-bold'>
                              *
                            </span>
                          )}
                        </Label>
                        {param.type === 'text' ? (
                          <Textarea
                            id={`param-${param.name}`}
                            placeholder={param.placeholder}
                            className='text-xs min-h-[80px]'
                            value={paramValues[param.name] || ''}
                            onChange={(e) =>
                              setParamValues((prev) => ({
                                ...prev,
                                [param.name]: e.target.value,
                              }))
                            }
                          />
                        ) : param.type === 'boolean' ? (
                          <div className='flex items-center gap-2 pt-1 h-9'>
                            <Switch
                              id={`param-${param.name}`}
                              checked={paramValues[param.name] === 'true'}
                              onCheckedChange={(checked) =>
                                setParamValues((prev) => ({
                                  ...prev,
                                  [param.name]: checked ? 'true' : 'false',
                                }))
                              }
                            />
                            <span className='text-xs text-muted-foreground'>
                              {paramValues[param.name] === 'true'
                                ? t('paramOn')
                                : t('paramOff')}
                            </span>
                          </div>
                        ) : isSwitchTarget &&
                          availableLogDbTargets.length > 0 ? (
                          <Select
                            value={paramValues[param.name] || ''}
                            onValueChange={(value) =>
                              setParamValues((prev) => ({
                                ...prev,
                                [param.name]: value,
                              }))
                            }
                            disabled={dispatching}
                          >
                            <SelectTrigger
                              id={`param-${param.name}`}
                              className='w-full text-xs'
                              size='sm'
                            >
                              <SelectValue placeholder={t('selectLogDb')} />
                            </SelectTrigger>
                            <SelectContent>
                              {availableLogDbTargets.map((target) => (
                                <SelectItem key={target} value={target}>
                                  {LOG_DATABASE_LABEL_KEYS[target]
                                    ? t(LOG_DATABASE_LABEL_KEYS[target])
                                    : target}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        ) : (
                          <Input
                            id={`param-${param.name}`}
                            type={param.type === 'number' ? 'number' : 'text'}
                            placeholder={param.placeholder}
                            className='text-xs'
                            value={paramValues[param.name] || ''}
                            onChange={(e) =>
                              setParamValues((prev) => ({
                                ...prev,
                                [param.name]: e.target.value,
                              }))
                            }
                          />
                        )}
                        {param.description && (
                          <p className='text-[10px] text-muted-foreground'>
                            {param.description}
                          </p>
                        )}
                      </div>
                    );
                  })}
                </div>
              );
            })()}

            {(() => {
              const targetTask = getSelectedTaskMeta();
              const hasTime = targetTask?.supports_time;
              const hasParams =
                targetTask?.params && targetTask.params.length > 0;
              if (!hasTime && !hasParams) {
                return (
                  <div className='flex items-center gap-2 text-sm text-muted-foreground bg-muted/50 p-3 rounded-md border border-dashed'>
                    <Info className='h-4 w-4' />
                    <span>{t('noParamsHint')}</span>
                  </div>
                );
              }
              return null;
            })()}
          </div>

          <DialogFooter>
            <Button
              variant='ghost'
              onClick={() => setDialogOpen(false)}
              disabled={dispatching}
              className='h-8 text-xs'
            >
              {t('cancel')}
            </Button>
            <Button
              onClick={handleDispatch}
              disabled={dispatching}
              className='h-8 text-xs'
            >
              {dispatching ? (
                <Spinner className='size-3' />
              ) : (
                <Play className='size-3' />
              )}
              {dispatching ? t('dispatching') : t('start')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
