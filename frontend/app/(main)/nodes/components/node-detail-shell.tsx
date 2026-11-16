'use client';

import Link from 'next/link';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { type ReactNode, useCallback } from 'react';
import type { LucideIcon } from 'lucide-react';
import { ArrowLeft, ChevronDown } from 'lucide-react';
import { useTranslations } from 'next-intl';

import { Button } from '@/components/ui/button';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { cn } from '@/lib/utils';

import { NodeKpiCard } from './node-detail-primitives';
import { NodeStatusBadge } from './node-status-badge';
import type { StatusTone } from './node-utils';

export type NodeDetailTabId =
  | 'overview'
  | 'manage'
  | 'health'
  | 'requests'
  | 'domains';

export type NodeDetailTabConfig = {
  id: NodeDetailTabId;
  label: string;
  icon?: LucideIcon;
};

export type NodeDetailKpi = {
  label: string;
  value: ReactNode;
  icon?: LucideIcon;
};

const TAB_IDS: NodeDetailTabId[] = [
  'overview',
  'health',
  'requests',
  'domains',
  'manage',
];

function resolveTab(value: string | null): NodeDetailTabId | null {
  if (value === 'overview' || value === 'dashboard') {
    return 'overview';
  }
  if (
    value === 'manage' ||
    value === 'health' ||
    value === 'requests' ||
    value === 'domains'
  ) {
    return value;
  }
  return null;
}

export function NodeDetailShell({
  title,
  typeLabel,
  typeTone = 'info',
  statusBadges,
  actions,
  kpis,
  overview,
  manage,
  health,
  requests,
  domains,
  defaultTab = 'overview',
}: {
  title: string;
  typeLabel: string;
  typeTone?: StatusTone;
  statusBadges: Array<{ label: string; tone: StatusTone }>;
  actions: ReactNode;
  kpis: NodeDetailKpi[];
  overview: ReactNode;
  manage: ReactNode;
  health?: ReactNode;
  requests?: ReactNode;
  domains?: ReactNode;
  defaultTab?: NodeDetailTabId;
}) {
  const t = useTranslations('nodes');
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const panels: Partial<Record<NodeDetailTabId, ReactNode>> = {
    overview,
    manage,
    health,
    requests,
    domains,
  };
  const tabIds = TAB_IDS.filter(
    (id) => id === 'overview' || id === 'manage' || panels[id] != null,
  );

  const activeTab = (() => {
    const tab = searchParams.get('tab');
    const resolved = resolveTab(tab);
    return resolved && panels[resolved] != null ? resolved : defaultTab;
  })();

  const setActiveTab = useCallback(
    (tab: NodeDetailTabId) => {
      const params = new URLSearchParams(searchParams.toString());
      params.set('tab', tab);
      router.replace(`${pathname}?${params.toString()}`, { scroll: false });
    },
    [pathname, router, searchParams],
  );

  const handleTabChange = useCallback(
    (value: string) => {
      const tab = resolveTab(value);
      if (tab) {
        setActiveTab(tab);
      }
    },
    [setActiveTab],
  );

  return (
    <div className='flex w-full min-w-0 flex-col gap-6 py-6 px-1'>
      <section className='overflow-hidden rounded-2xl border bg-gradient-to-br from-card via-card to-muted/30'>
        <div className='flex flex-col gap-5 p-5 md:p-6'>
          <div className='flex flex-wrap items-start justify-between gap-4'>
            <div className='flex min-w-0 flex-1 items-start gap-3'>
              <Button
                variant='ghost'
                size='sm'
                className='mt-0.5 h-8 w-8 shrink-0 p-0'
                asChild
              >
                <Link href='/nodes' aria-label={t('backToList')}>
                  <ArrowLeft className='size-4' />
                </Link>
              </Button>

              <div className='min-w-0 space-y-2'>
                <div className='flex flex-wrap items-center gap-2'>
                  <h1 className='text-2xl font-semibold tracking-tight [overflow-wrap:anywhere]'>
                    {title}
                  </h1>
                  <NodeStatusBadge label={typeLabel} tone={typeTone} />
                </div>
                <div className='flex flex-wrap items-center gap-2'>
                  {statusBadges.map((badge) => (
                    <NodeStatusBadge
                      key={badge.label}
                      label={badge.label}
                      tone={badge.tone}
                    />
                  ))}
                </div>
              </div>
            </div>

            <Collapsible className='w-full min-w-0'>
              <CollapsibleTrigger asChild>
                <Button
                  variant='outline'
                  size='sm'
                  className='group w-full justify-between sm:hidden'
                >
                  {t('moreActions')}
                  <ChevronDown
                    data-icon='inline-end'
                    className='group-data-[state=open]:rotate-180'
                  />
                </Button>
              </CollapsibleTrigger>
              <CollapsibleContent
                forceMount
                className='hidden flex-wrap items-center gap-2 pt-3 data-[state=open]:flex sm:flex sm:pt-0 [&_button]:h-auto [&_button]:min-h-8 [&_button]:max-w-full [&_button]:whitespace-normal [&_button]:[overflow-wrap:anywhere]'
              >
                {actions}
              </CollapsibleContent>
            </Collapsible>
          </div>

          {kpis.length > 0 ? (
            <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
              {kpis.map((kpi) => (
                <NodeKpiCard
                  key={kpi.label}
                  label={kpi.label}
                  value={kpi.value}
                  icon={kpi.icon}
                />
              ))}
            </div>
          ) : null}
        </div>
      </section>

      <Tabs
        value={activeTab}
        onValueChange={handleTabChange}
        className='w-full gap-0'
      >
        <div className='space-y-3 pb-1'>
          <TabsList
            variant='line'
            className='h-auto w-full flex-wrap justify-start gap-6 bg-transparent p-0'
          >
            {tabIds.map((tabId) => (
              <TabsTrigger
                key={tabId}
                value={tabId}
                className={cn(
                  'h-auto flex-none rounded-none px-0 pb-3 pt-1 text-sm font-semibold',
                  'data-[state=active]:text-foreground data-[state=inactive]:text-muted-foreground',
                )}
              >
                {tabId === 'overview'
                  ? t('tabOverview')
                  : tabId === 'manage'
                    ? t('tabManage')
                    : t(`detailSections.${tabId}`)}
              </TabsTrigger>
            ))}
          </TabsList>
        </div>

        <TabsContent value='overview' className='mt-6 outline-none'>
          {activeTab === 'overview' ? overview : null}
        </TabsContent>
        <TabsContent value='manage' className='mt-6 outline-none'>
          {activeTab === 'manage' ? manage : null}
        </TabsContent>
        {(['health', 'requests', 'domains'] as const).map((id) =>
          panels[id] != null ? (
            <TabsContent key={id} value={id} className='mt-6 outline-none'>
              {activeTab === id ? panels[id] : null}
            </TabsContent>
          ) : null,
        )}
      </Tabs>
    </div>
  );
}
