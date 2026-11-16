'use client';

import { useCallback, useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';
import { RefreshCw } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import services from '@/lib/services';
import { TableBrowser } from './table-browser';

export function TablesTab() {
  const t = useTranslations('admin.database');
  const [tables, setTables] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  const fetchTables = useCallback(async () => {
    setLoading(true);
    try {
      setTables(await services.dbManage.listTables());
    } catch (error) {
      toast.error(t('fetchTablesFailed'), {
        description: error instanceof Error ? error.message : t('unknownError'),
      });
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void fetchTables();
  }, [fetchTables]);

  return (
    <div className='flex min-w-0 flex-col gap-4'>
      <Button
        variant='outline'
        size='sm'
        className='self-start'
        disabled={loading}
        onClick={() => {
          void fetchTables();
          setRefreshTrigger((previous) => previous + 1);
        }}
      >
        <RefreshCw data-icon='inline-start' />
        {t('refreshData')}
      </Button>
      <TableBrowser
        tables={tables}
        loadingTables={loading}
        refreshTrigger={refreshTrigger}
      />
    </div>
  );
}
