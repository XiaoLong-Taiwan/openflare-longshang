'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  AlertTriangle,
  Cloud,
  Plus,
  RefreshCw,
  Settings,
  Trash2,
} from 'lucide-react';
import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { useState } from 'react';
import { toast } from 'sonner';

import { formatDateTime } from '@/lib/utils';

import { GroupDialog } from '@/app/(main)/cloudflare/components/group-dialog';
import { SyncTasksPanel } from '@/app/(main)/cloudflare/components/sync-tasks-panel';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { EmptyStateWithBorder } from '@/components/layout/empty';
import { ErrorInline } from '@/components/layout/error';
import { LoadingStateWithBorder } from '@/components/layout/loading';
import {
  CloudflareService,
  cloudflareQueryKey,
  NodeService,
  type CloudflareGroup,
  type CloudflareGroupPayload,
} from '@/lib/services/openflare';
import { getErrorMessage } from '../websites/components/website-utils';

export default function CloudflarePage() {
  const t = useTranslations('cloudflare');
  const tCommon = useTranslations('common');
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<CloudflareGroup | null>(
    null,
  );
  const overviewQuery = useQuery({
    queryKey: [...cloudflareQueryKey, 'overview'],
    queryFn: () => CloudflareService.getOverview(),
  });
  const groupsQuery = useQuery({
    queryKey: [...cloudflareQueryKey, 'groups'],
    queryFn: () => CloudflareService.listGroups(),
  });
  const nodesQuery = useQuery({
    queryKey: ['openflare', 'nodes'],
    queryFn: () => NodeService.listNodes(),
  });

  const invalidate = async () =>
    queryClient.invalidateQueries({ queryKey: cloudflareQueryKey });
  const createMutation = useMutation({
    mutationFn: (payload: CloudflareGroupPayload) =>
      CloudflareService.createGroup(payload),
    onSuccess: async () => {
      toast.success(t('created'));
      setCreateOpen(false);
      await invalidate();
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });
  const syncMutation = useMutation({
    mutationFn: (id: number) => CloudflareService.syncGroup(id),
    onSuccess: async () => {
      toast.success(t('syncQueued'));
      await queryClient.invalidateQueries({
        queryKey: [...cloudflareQueryKey, 'sync-executions'],
      });
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });
  const deleteMutation = useMutation({
    mutationFn: (id: number) => CloudflareService.deleteGroup(id),
    onSuccess: async () => {
      toast.success(t('deleted'));
      setDeleteTarget(null);
      await invalidate();
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  const overview = overviewQuery.data;

  return (
    <div className='flex w-full flex-col gap-6 py-6 px-1'>
      <div className='flex items-center justify-between gap-3'>
        <div className='flex items-center gap-2'>
          <Cloud className='size-5 text-primary' />
          <h1 className='text-2xl font-semibold tracking-tight'>
            {t('title')}
          </h1>
        </div>
        <div className='flex items-center gap-2'>
          <Button asChild variant='outline' size='sm'>
            <Link href='/cloudflare/settings'>
              <Settings data-icon='inline-start' />
              {t('connectionSettings')}
            </Link>
          </Button>
          <Button size='sm' onClick={() => setCreateOpen(true)}>
            <Plus data-icon='inline-start' />
            {t('addGroup')}
          </Button>
        </div>
      </div>

      {overviewQuery.isLoading ? (
        <LoadingStateWithBorder
          icon={Cloud}
          description={t('loadingOverview')}
        />
      ) : overviewQuery.isError ? (
        <ErrorInline
          message={getErrorMessage(overviewQuery.error)}
          onRetry={() => void overviewQuery.refetch()}
        />
      ) : !overview?.connection.ready ? (
        <Alert>
          <AlertTriangle />
          <AlertTitle>{t('notReadyTitle')}</AlertTitle>
          <AlertDescription className='flex flex-col items-start gap-3'>
            <span>{t('notReadyDesc')}</span>
            <Button asChild size='sm'>
              <Link href='/cloudflare/settings'>{t('configure')}</Link>
            </Button>
          </AlertDescription>
        </Alert>
      ) : null}

      <div className='flex flex-col gap-4'>
        <div>
          <h2 className='text-lg font-semibold'>{t('groupsTitle')}</h2>
          <p className='text-sm text-muted-foreground'>{t('groupsDesc')}</p>
        </div>

        {groupsQuery.isLoading ? (
          <LoadingStateWithBorder
            icon={Cloud}
            description={t('loadingGroups')}
          />
        ) : groupsQuery.isError ? (
          <ErrorInline
            message={getErrorMessage(groupsQuery.error)}
            onRetry={() => void groupsQuery.refetch()}
          />
        ) : (groupsQuery.data ?? []).length === 0 ? (
          <EmptyStateWithBorder
            icon={Cloud}
            title={t('emptyTitle')}
            description={t('emptyDesc')}
            actionText={t('addGroup')}
            onAction={() => setCreateOpen(true)}
          />
        ) : (
          <div className='overflow-x-auto rounded-lg border border-dashed'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('columns.name')}</TableHead>
                  <TableHead>{t('columns.status')}</TableHead>
                  <TableHead>{t('columns.nodes')}</TableHead>
                  <TableHead>{t('columns.members')}</TableHead>
                  <TableHead>{t('columns.updatedAt')}</TableHead>
                  <TableHead className='w-[180px] text-right'>
                    {t('columns.actions')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(groupsQuery.data ?? []).map((group) => (
                  <TableRow key={group.id}>
                    <TableCell className='font-medium'>{group.name}</TableCell>
                    <TableCell>
                      <Badge variant={group.enabled ? 'default' : 'secondary'}>
                        {group.enabled ? t('enabled') : t('disabled')}
                      </Badge>
                    </TableCell>
                    <TableCell>{group.nodes?.length ?? 1}</TableCell>
                    <TableCell>{group.member_count}</TableCell>
                    <TableCell className='text-sm text-muted-foreground'>
                      {formatDateTime(group.updated_at)}
                    </TableCell>
                    <TableCell className='text-right'>
                      <div className='flex justify-end gap-1'>
                        <Button
                          asChild
                          variant='ghost'
                          size='icon'
                          title={t('manage')}
                          aria-label={t('manage')}
                        >
                          <Link href={`/cloudflare/group?id=${group.id}`}>
                            <Settings />
                          </Link>
                        </Button>
                        <Button
                          variant='ghost'
                          size='icon'
                          title={t('sync')}
                          aria-label={t('sync')}
                          onClick={() => syncMutation.mutate(group.id)}
                        >
                          <RefreshCw />
                        </Button>
                        <Button
                          variant='ghost'
                          size='icon'
                          className='text-destructive hover:text-destructive'
                          title={t('delete')}
                          aria-label={t('delete')}
                          onClick={() => setDeleteTarget(group)}
                        >
                          <Trash2 />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>

      <SyncTasksPanel />

      <GroupDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        nodes={nodesQuery.data ?? []}
        pending={createMutation.isPending}
        onSubmit={(payload) => createMutation.mutate(payload)}
      />

      <AlertDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('deleteTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('deleteDesc', { name: deleteTarget?.name ?? '' })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon('cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() =>
                deleteTarget && deleteMutation.mutate(deleteTarget.id)
              }
            >
              {t('confirmDelete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
