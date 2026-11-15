'use client';

import Link from 'next/link';
import { Pencil, Trash2 } from 'lucide-react';
import { useTranslations } from 'next-intl';

import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { formatDateTime } from '@/lib/utils';
import type { NodeItem } from '@/lib/services/openflare';

import { NodeStatusBadge } from './node-status-badge';
import {
  formatRelativeTime,
  getApplyLabel,
  getApplyTone,
  getNodeStatusLabel,
  getNodeStatusTone,
  getNodeTypeLabel,
  getOpenrestyStatusLabel,
  getOpenrestyStatusTone,
  getRelayStatusLabel,
  getRelayStatusTone,
  isMeaningfulTime,
  isWSConnectedLastSeen,
} from './node-utils';

export function NodesTable({
  nodes,
  deletingId,
  onEdit,
  onDelete,
  selectedIds,
  onToggle,
  onToggleAll,
  actionsDisabled = false,
}: {
  nodes: NodeItem[];
  deletingId: number | null;
  actionsDisabled?: boolean;
  onEdit: (node: NodeItem) => void;
  onDelete: (node: NodeItem) => void;
  selectedIds: Set<number>;
  onToggle: (id: number) => void;
  onToggleAll: (checked: boolean) => void;
}) {
  const t = useTranslations('nodes');
  return (
    <div
      className='border border-dashed rounded-lg overflow-hidden'
      aria-busy={actionsDisabled}
    >
      <Table className='block w-full md:table'>
        <TableHeader className='block md:table-header-group'>
          <TableRow className='border-dashed hover:bg-transparent'>
            <TableHead className='w-10 py-2 h-8'>
              <Checkbox
                checked={
                  nodes.length > 0 &&
                  nodes.every((node) => selectedIds.has(node.id))
                }
                onCheckedChange={(checked) => onToggleAll(checked === true)}
                aria-label={t('selectAll')}
              />
            </TableHead>
            <TableHead className='hidden py-2 h-8 md:table-cell'>
              {t('table.node')}
            </TableHead>
            <TableHead className='hidden py-2 h-8 md:table-cell'>
              {t('table.status')}
            </TableHead>
            <TableHead className='hidden py-2 h-8 md:table-cell'>
              {t('table.version')}
            </TableHead>
            <TableHead className='hidden py-2 h-8 md:table-cell'>
              {t('table.runHealth')}
            </TableHead>
            <TableHead className='hidden py-2 h-8 md:table-cell'>
              {t('table.currentVersion')}
            </TableHead>
            <TableHead className='hidden py-2 h-8 md:table-cell'>
              {t('table.latestApply')}
            </TableHead>
            <TableHead className='hidden py-2 h-8 md:table-cell'>
              {t('table.lastSeen')}
            </TableHead>
            <TableHead className='hidden py-2 h-8 text-right md:table-cell'>
              {t('table.actions')}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody className='block md:table-row-group'>
          {nodes.map((node) => (
            <TableRow
              key={node.id}
              className='grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 border-dashed p-3 align-top md:table-row md:p-0'
            >
              <TableCell className='py-3 md:table-cell'>
                <Checkbox
                  checked={selectedIds.has(node.id)}
                  onCheckedChange={() => onToggle(node.id)}
                  aria-label={t('selectNode', { name: node.name })}
                />
              </TableCell>
              <TableCell className='min-w-0 whitespace-normal py-3 md:max-w-72'>
                <div className='space-y-1'>
                  <div className='flex flex-wrap items-center gap-2'>
                    <p className='min-w-0 font-medium [overflow-wrap:anywhere]'>
                      {node.name}
                    </p>
                    <NodeStatusBadge
                      label={getNodeTypeLabel(node.node_type)}
                      tone='info'
                    />
                  </div>
                  <p className='text-xs text-muted-foreground [overflow-wrap:anywhere]'>
                    {t('ipLine', { ip: node.ip || '—' })}
                    {node.ip_manual_override ? t('lockedSuffix') : ''}
                  </p>
                  <p className='text-xs text-muted-foreground [overflow-wrap:anywhere]'>
                    {t('locationLine', {
                      location: node.geo_name || t('noGeo'),
                    })}
                  </p>
                </div>
              </TableCell>
              <TableCell className='col-start-2 flex flex-wrap items-center gap-2 py-2 md:table-cell md:py-3'>
                <span className='text-xs text-muted-foreground md:hidden'>
                  {t('table.status')}
                </span>
                <NodeStatusBadge
                  label={getNodeStatusLabel(node.status, t)}
                  tone={getNodeStatusTone(node.status)}
                />
              </TableCell>
              <TableCell className='hidden whitespace-normal py-3 text-muted-foreground [overflow-wrap:anywhere] md:table-cell md:max-w-40'>
                {node.version || 'unknown'}
              </TableCell>
              <TableCell className='col-start-2 flex flex-wrap items-center gap-2 py-2 md:table-cell md:py-3'>
                <span className='text-xs text-muted-foreground md:hidden'>
                  {t('table.runHealth')}
                </span>
                {node.node_type === 'tunnel_relay' ? (
                  <NodeStatusBadge
                    label={getRelayStatusLabel(node.relay_status, t)}
                    tone={getRelayStatusTone(node.relay_status)}
                  />
                ) : node.node_type === 'tunnel_client' ? (
                  <NodeStatusBadge
                    label={
                      node.status === 'online' ? t('running') : t('unknown')
                    }
                    tone={node.status === 'online' ? 'success' : 'warning'}
                  />
                ) : (
                  <NodeStatusBadge
                    label={getOpenrestyStatusLabel(node.openresty_status, t)}
                    tone={getOpenrestyStatusTone(node.openresty_status)}
                  />
                )}
              </TableCell>
              <TableCell className='hidden whitespace-normal py-3 text-muted-foreground [overflow-wrap:anywhere] md:table-cell md:max-w-48'>
                {node.current_version ||
                  (node.node_type === 'tunnel_relay'
                    ? t('liveConfig')
                    : t('notApplied'))}
              </TableCell>
              <TableCell className='hidden py-3 md:table-cell'>
                {node.node_type === 'tunnel_relay' ? (
                  <span className='text-sm text-muted-foreground'>—</span>
                ) : (
                  <NodeStatusBadge
                    label={getApplyLabel(node.latest_apply_result, t)}
                    tone={getApplyTone(node.latest_apply_result)}
                  />
                )}
              </TableCell>
              <TableCell className='col-start-2 whitespace-normal py-2 text-muted-foreground md:py-3'>
                <span className='mr-2 text-xs md:hidden'>
                  {t('table.lastSeen')}
                </span>
                {isWSConnectedLastSeen(node.last_seen_at)
                  ? t('wsConnected')
                  : isMeaningfulTime(node.last_seen_at)
                    ? `${formatRelativeTime(node.last_seen_at, t)} · ${formatDateTime(node.last_seen_at)}`
                    : t('na')}
              </TableCell>
              <TableCell className='col-span-2 py-3'>
                <div className='flex flex-wrap justify-end gap-2'>
                  <Button
                    variant='outline'
                    size='sm'
                    className='h-7 text-xs'
                    asChild={!actionsDisabled}
                    disabled={actionsDisabled}
                  >
                    {actionsDisabled ? (
                      t('detail')
                    ) : (
                      <Link href={`/nodes/detail?id=${node.id}`}>
                        {t('detail')}
                      </Link>
                    )}
                  </Button>
                  <Button
                    variant='outline'
                    size='sm'
                    className='h-7 text-xs'
                    disabled={actionsDisabled}
                    onClick={() => onEdit(node)}
                  >
                    <Pencil className='size-3 mr-1' />
                    {t('edit')}
                  </Button>
                  <Button
                    variant='outline'
                    size='sm'
                    className='h-7 text-xs text-destructive hover:text-destructive'
                    disabled={actionsDisabled || deletingId === node.id}
                    onClick={() => onDelete(node)}
                  >
                    <Trash2 className='size-3 mr-1' />
                    {t('delete')}
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
