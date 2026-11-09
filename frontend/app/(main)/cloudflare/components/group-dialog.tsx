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
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field';
import { Input } from '@/components/ui/input';
import { Switch } from '@/components/ui/switch';
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group';
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
  const [targetMode, setTargetMode] = useState<'node' | 'custom'>('node');
  const [recordType, setRecordType] = useState<'A' | 'AAAA' | 'CNAME'>('A');
  const [recordContent, setRecordContent] = useState('');
  const [selectedNodes, setSelectedNodes] = useState<
    Array<{ nodeID: number; priority: number }>
  >([]);
  const [defaultProxied, setDefaultProxied] = useState(true);
  const [enabled, setEnabled] = useState(true);
  const [ttlMode, setTtlMode] = useState<'auto' | 'manual'>('auto');
  const [manualTTL, setManualTTL] = useState('300');
  const ttl = ttlMode === 'auto' ? 1 : Number(manualTTL);
  const invalidTTL =
    ttlMode === 'manual' &&
    (!manualTTL.trim() || !Number.isInteger(ttl) || ttl < 60 || ttl > 86400);

  useEffect(() => {
    if (!open) return;
    setName(group?.name ?? '');
    setTargetMode(group?.target_mode ?? 'node');
    setRecordType(group?.record_type || 'A');
    setRecordContent(group?.record_content ?? '');
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
    const groupTTL = group ? (group.ttl ?? 300) : 1;
    setTtlMode(groupTTL === 1 ? 'auto' : 'manual');
    setManualTTL(String(groupTTL === 1 ? 300 : groupTTL));
  }, [group, open]);

  return (
    <Dialog
      open={open}
      onOpenChange={(value) => {
        if (!pending) onOpenChange(value);
      }}
    >
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
            <FieldLabel id='cf-target-mode'>{t('targetMode')}</FieldLabel>
            <ToggleGroup
              type='single'
              variant='outline'
              value={targetMode}
              aria-labelledby='cf-target-mode'
              disabled={pending}
              onValueChange={(value) => {
                if (value === 'node' || value === 'custom')
                  setTargetMode(value);
              }}
            >
              <ToggleGroupItem value='node'>{t('nodeTarget')}</ToggleGroupItem>
              <ToggleGroupItem value='custom'>
                {t('customTarget')}
              </ToggleGroupItem>
            </ToggleGroup>
          </Field>
          {targetMode === 'custom' ? (
            <>
              <Field>
                <FieldLabel id='cf-record-type'>{t('recordType')}</FieldLabel>
                <ToggleGroup
                  type='single'
                  variant='outline'
                  value={recordType}
                  aria-labelledby='cf-record-type'
                  disabled={pending}
                  onValueChange={(value) => {
                    if (value === 'A' || value === 'AAAA' || value === 'CNAME')
                      setRecordType(value);
                  }}
                >
                  <ToggleGroupItem value='A'>A</ToggleGroupItem>
                  <ToggleGroupItem value='AAAA'>AAAA</ToggleGroupItem>
                  <ToggleGroupItem value='CNAME'>CNAME</ToggleGroupItem>
                </ToggleGroup>
              </Field>
              <Field>
                <FieldLabel htmlFor='cf-record-content'>
                  {t('recordContent')}
                </FieldLabel>
                <Input
                  id='cf-record-content'
                  value={recordContent}
                  disabled={pending}
                  maxLength={253}
                  onChange={(event) => setRecordContent(event.target.value)}
                />
              </Field>
            </>
          ) : (
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
                              : current.filter(
                                  (item) => item.nodeID !== node.id,
                                ),
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
          )}
          <Field>
            <FieldLabel id='cf-ttl-mode-label'>{t('ttl')}</FieldLabel>
            <ToggleGroup
              type='single'
              variant='outline'
              value={ttlMode}
              aria-labelledby='cf-ttl-mode-label'
              aria-describedby='cf-ttl-hint'
              className='grid w-full grid-cols-2'
              onValueChange={(value) => {
                if (value === 'auto' || value === 'manual') {
                  setTtlMode(value);
                }
              }}
            >
              <ToggleGroupItem value='auto' className='w-full'>
                {t('ttlAuto')}
              </ToggleGroupItem>
              <ToggleGroupItem value='manual' className='w-full'>
                {t('ttlManual')}
              </ToggleGroupItem>
            </ToggleGroup>
            <FieldDescription id='cf-ttl-hint'>{t('ttlHint')}</FieldDescription>
          </Field>
          {ttlMode === 'manual' && (
            <Field data-invalid={invalidTTL}>
              <FieldLabel htmlFor='cf-ttl-seconds'>
                {t('ttlSeconds')}
              </FieldLabel>
              <Input
                id='cf-ttl-seconds'
                type='number'
                min={60}
                max={86400}
                step={1}
                required
                value={manualTTL}
                aria-invalid={invalidTTL}
                aria-describedby={
                  invalidTTL ? 'cf-ttl-hint cf-ttl-error' : 'cf-ttl-hint'
                }
                onChange={(event) => setManualTTL(event.target.value)}
              />
              {invalidTTL && (
                <FieldError id='cf-ttl-error'>{t('ttlError')}</FieldError>
              )}
            </Field>
          )}
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
          <Button
            variant='outline'
            disabled={pending}
            onClick={() => onOpenChange(false)}
          >
            {tCommon('cancel')}
          </Button>
          <Button
            disabled={
              pending ||
              invalidTTL ||
              !name.trim() ||
              (targetMode === 'node'
                ? selectedNodes.length === 0
                : !recordContent.trim())
            }
            onClick={() => {
              if (invalidTTL) return;
              onSubmit({
                name: name.trim(),
                target_mode: targetMode,
                record_type: targetMode === 'custom' ? recordType : 'A',
                record_content:
                  targetMode === 'custom' ? recordContent.trim() : '',
                primary_node_id: selectedNodes[0]?.nodeID,
                backup_node_id: selectedNodes[1]?.nodeID ?? null,
                nodes: (targetMode === 'node' ? selectedNodes : []).map(
                  (node) => ({
                    node_id: node.nodeID,
                    priority: node.priority,
                  }),
                ),
                default_proxied: defaultProxied,
                ttl,
                enabled,
              });
            }}
          >
            {t('save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
