'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useMemo, useState } from 'react';

import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field';
import { Input } from '@/components/ui/input';
import { Switch } from '@/components/ui/switch';
import type {
  CloudflareGroup,
  CloudflareGroupPayload,
  NodeItem,
} from '@/lib/services/openflare';

export function GroupDialog({
  open,
  onOpenChange,
  group,
  nodes,
  pending,
  onSubmit,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  group?: CloudflareGroup | null;
  nodes: NodeItem[];
  pending: boolean;
  onSubmit: (payload: CloudflareGroupPayload) => void;
}) {
  const t = useTranslations('cloudflare.groupDialog');
  const tCommon = useTranslations('common');
  const edgeNodes = useMemo(
    () => nodes.filter((node) => node.node_type === 'edge_node'),
    [nodes],
  );
  const [name, setName] = useState('');
  const [selectedNodes, setSelectedNodes] = useState<
    Array<{ nodeID: number; priority: number }>
  >([]);
  const [defaultProxied, setDefaultProxied] = useState(true);
  const [enabled, setEnabled] = useState(true);

  useEffect(() => {
    if (!open) return;
    setName(group?.name ?? '');
    setSelectedNodes(
      group?.nodes?.length
        ? group.nodes.map((node) => ({
            nodeID: node.id,
            priority: node.priority,
          }))
        : group
          ? [
              { nodeID: group.primary_node.id, priority: 0 },
              ...(group.backup_node
                ? [{ nodeID: group.backup_node.id, priority: 1 }]
                : []),
            ]
          : [],
    );
    setDefaultProxied(group?.default_proxied ?? true);
    setEnabled(group?.enabled ?? true);
  }, [group, open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{group ? t('editTitle') : t('createTitle')}</DialogTitle>
          <DialogDescription>{t('description')}</DialogDescription>
        </DialogHeader>
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor='cf-group-name'>{t('name')}</FieldLabel>
            <Input
              id='cf-group-name'
              value={name}
              onChange={(event) => setName(event.target.value)}
            />
          </Field>
          <Field>
            <FieldLabel>{t('nodes')}</FieldLabel>
            <div className='flex flex-col gap-2'>
              {edgeNodes.map((node) => {
                const selected = selectedNodes.find(
                  (item) => item.nodeID === node.id,
                );
                return (
                  <div key={node.id} className='flex items-center gap-3'>
                    <Checkbox
                      aria-label={t('selectNode', { name: node.name })}
                      checked={Boolean(selected)}
                      disabled={!node.ip}
                      onCheckedChange={(checked) =>
                        setSelectedNodes((current) =>
                          checked
                            ? [...current, { nodeID: node.id, priority: 0 }]
                            : current.filter((item) => item.nodeID !== node.id),
                        )
                      }
                    />
                    <span className='min-w-0 flex-1 truncate text-sm'>
                      {node.name} · {node.ip || t('noIp')}
                    </span>
                    <Input
                      className='w-24'
                      type='number'
                      min={0}
                      aria-label={t('priorityFor', { name: node.name })}
                      disabled={!selected}
                      value={selected?.priority ?? 0}
                      onChange={(event) =>
                        setSelectedNodes((current) =>
                          current.map((item) =>
                            item.nodeID === node.id
                              ? {
                                  ...item,
                                  priority: Math.max(
                                    0,
                                    Number(event.target.value) || 0,
                                  ),
                                }
                              : item,
                          ),
                        )
                      }
                    />
                  </div>
                );
              })}
            </div>
          </Field>
          <Field orientation='horizontal'>
            <FieldLabel htmlFor='cf-default-proxied'>
              {t('defaultProxied')}
            </FieldLabel>
            <Switch
              id='cf-default-proxied'
              checked={defaultProxied}
              onCheckedChange={setDefaultProxied}
            />
          </Field>
          <Field orientation='horizontal'>
            <FieldLabel htmlFor='cf-enabled'>{t('enableSync')}</FieldLabel>
            <Switch
              id='cf-enabled'
              checked={enabled}
              onCheckedChange={setEnabled}
            />
          </Field>
        </FieldGroup>
        <DialogFooter>
          <Button variant='outline' onClick={() => onOpenChange(false)}>
            {tCommon('cancel')}
          </Button>
          <Button
            disabled={pending || !name.trim() || selectedNodes.length === 0}
            onClick={() =>
              onSubmit({
                name: name.trim(),
                primary_node_id: selectedNodes[0]?.nodeID,
                backup_node_id: selectedNodes[1]?.nodeID ?? null,
                nodes: selectedNodes.map((node) => ({
                  node_id: node.nodeID,
                  priority: node.priority,
                })),
                default_proxied: defaultProxied,
                enabled,
              })
            }
          >
            {t('save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
