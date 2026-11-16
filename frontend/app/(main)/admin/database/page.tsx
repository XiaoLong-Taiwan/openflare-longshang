'use client';

import dynamic from 'next/dynamic';
import { useState } from 'react';
import { Database } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Skeleton } from '@/components/ui/skeleton';
import { DatabaseOverview } from './page-client';

const TablesTab = dynamic(
  () => import('./components/tables-tab').then((mod) => mod.TablesTab),
  { loading: () => <Skeleton className='h-48 w-full' /> },
);

export default function DatabasePage() {
  const t = useTranslations('admin.database');
  const [tab, setTab] = useState('overview');
  const [tablesVisited, setTablesVisited] = useState(false);

  return (
    <div className='flex w-full min-w-0 flex-col gap-4 py-6 px-1'>
      <div className='flex items-center gap-2'>
        <Database className='size-5 shrink-0 text-primary' />
        <h1 className='text-2xl font-semibold tracking-tight'>
          {t('pageTitle')}
        </h1>
      </div>
      <Tabs
        value={tab}
        onValueChange={(value) => {
          setTab(value);
          if (value === 'tables') setTablesVisited(true);
        }}
        className='w-full'
      >
        <TabsList variant='line'>
          <TabsTrigger value='overview'>{t('sections.overview')}</TabsTrigger>
          <TabsTrigger value='tables'>{t('sections.tables')}</TabsTrigger>
        </TabsList>
        <TabsContent value='overview' forceMount hidden={tab !== 'overview'}>
          <DatabaseOverview />
        </TabsContent>
        <TabsContent value='tables' forceMount hidden={tab !== 'tables'}>
          {tablesVisited ? <TablesTab /> : null}
        </TabsContent>
      </Tabs>
    </div>
  );
}
