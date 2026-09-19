'use client';

import { Plus, Trash2 } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type {
  ProxyRouteLoadBalancing,
  ProxyRouteUpstreamTarget,
} from '@/lib/services/openflare';

interface UpstreamTargetEditorProps {
  value: ProxyRouteUpstreamTarget[];
  loadBalancing: ProxyRouteLoadBalancing;
  onChange: (value: ProxyRouteUpstreamTarget[]) => void;
  onLoadBalancingChange: (value: ProxyRouteLoadBalancing) => void;
  labels: {
    address: string;
    priority: string;
    loadBalancing: string;
    roundRobin: string;
    leastConn: string;
    add: string;
    remove: string;
    hint: string;
  };
}

export function UpstreamTargetEditor({
  value,
  loadBalancing,
  onChange,
  onLoadBalancingChange,
  labels,
}: UpstreamTargetEditorProps) {
  const updateTarget = (
    index: number,
    patch: Partial<ProxyRouteUpstreamTarget>,
  ) => {
    onChange(
      value.map((target, targetIndex) =>
        targetIndex === index ? { ...target, ...patch } : target,
      ),
    );
  };

  return (
    <div className='space-y-3'>
      {value.map((target, index) => (
        <div
          key={index}
          className='grid gap-2 sm:grid-cols-[minmax(0,1fr)_7rem_auto]'
        >
          <div className='space-y-1'>
            {index === 0 ? <Label>{labels.address}</Label> : null}
            <Input
              value={target.url}
              onChange={(event) =>
                updateTarget(index, { url: event.target.value })
              }
              placeholder='https://origin.example.internal:443'
              className='font-mono text-xs'
            />
          </div>
          <div className='space-y-1'>
            {index === 0 ? <Label>{labels.priority}</Label> : null}
            <Input
              type='number'
              min={0}
              step={1}
              value={target.priority}
              onChange={(event) =>
                updateTarget(index, { priority: Number(event.target.value) })
              }
            />
          </div>
          <div className='flex items-end'>
            <Button
              type='button'
              variant='ghost'
              size='icon'
              aria-label={labels.remove}
              title={labels.remove}
              disabled={value.length === 1}
              onClick={() =>
                onChange(
                  value.filter((_, targetIndex) => targetIndex !== index),
                )
              }
            >
              <Trash2 className='size-4' />
            </Button>
          </div>
        </div>
      ))}
      <div className='flex flex-wrap items-center gap-2'>
        <Button
          type='button'
          variant='outline'
          size='sm'
          onClick={() => onChange([...value, { url: '', priority: 0 }])}
        >
          <Plus className='size-4' />
          {labels.add}
        </Button>
        <div className='flex items-center gap-2'>
          <Label htmlFor='proxy-route-load-balancing'>
            {labels.loadBalancing}
          </Label>
          <Select value={loadBalancing} onValueChange={onLoadBalancingChange}>
            <SelectTrigger id='proxy-route-load-balancing' className='w-44'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='round_robin'>{labels.roundRobin}</SelectItem>
              <SelectItem value='least_conn'>{labels.leastConn}</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>
      <p className='text-sm text-muted-foreground'>{labels.hint}</p>
    </div>
  );
}
