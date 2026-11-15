'use client';

import type { ReactNode } from 'react';
import type { LucideIcon } from 'lucide-react';

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { cn } from '@/lib/utils';

export function NodeInfoRow({
  label,
  children,
  className,
}: {
  label: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        'flex min-w-0 flex-col gap-1 py-2.5 text-sm sm:flex-row sm:items-start sm:justify-between sm:gap-4',
        className,
      )}
    >
      <span className='text-muted-foreground sm:max-w-[45%]'>{label}</span>
      <div className='min-w-0 whitespace-pre-wrap font-medium [overflow-wrap:anywhere] sm:text-right'>
        {children}
      </div>
    </div>
  );
}

export function NodeKpiCard({
  label,
  value,
  icon: Icon,
}: {
  label: string;
  value: ReactNode;
  icon?: LucideIcon;
}) {
  return (
    <div className='rounded-xl border bg-card/60 px-4 py-3.5 backdrop-blur-sm'>
      <div className='flex items-center gap-2 text-xs text-muted-foreground'>
        {Icon ? <Icon className='size-3.5 opacity-70' /> : null}
        <span>{label}</span>
      </div>
      <p className='mt-2 text-sm font-semibold tracking-tight break-all'>
        {value}
      </p>
    </div>
  );
}

export function NodeSectionCard({
  title,
  description,
  action,
  children,
  className,
}: {
  title: string;
  description?: string;
  action?: React.ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <Card className={cn('min-w-0 border shadow-none', className)}>
      <CardHeader
        className={cn(
          'pb-3',
          action
            ? 'flex flex-col items-start gap-3 sm:flex-row sm:justify-between'
            : '',
        )}
      >
        <div className='flex min-w-0 flex-col gap-1 [overflow-wrap:anywhere]'>
          <CardTitle className='text-base font-semibold'>{title}</CardTitle>
          {description ? (
            <CardDescription>{description}</CardDescription>
          ) : null}
        </div>
        {action}
      </CardHeader>
      <CardContent>{children}</CardContent>
    </Card>
  );
}

export function NodeErrorBanner({ message }: { message: string }) {
  return (
    <div className='min-w-0 whitespace-pre-wrap rounded-xl border border-destructive/25 bg-destructive/5 px-4 py-3 text-sm text-destructive [overflow-wrap:anywhere]'>
      {message}
    </div>
  );
}
