import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { NextIntlClientProvider } from 'next-intl';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import DatabasePage from '@/app/(main)/admin/database/page';
import services from '@/lib/services';
import { DbManageService } from '@/lib/services/db-manage';
import english from '@/messages/en.json';

vi.mock('next/dynamic', async () => {
  const { lazy, Suspense } = await import('react');
  return {
    default: (loader: () => Promise<React.ComponentType>) => {
      const Component = lazy(async () => ({ default: await loader() }));
      return function DynamicComponent(props: Record<string, unknown>) {
        return (
          <Suspense>
            <Component {...props} />
          </Suspense>
        );
      };
    },
  };
});

vi.mock('@/lib/services', () => ({
  default: {
    dbManage: {
      getOverview: vi.fn(),
      listTables: vi.fn(),
      getMaintenancePreview: vi.fn(),
      runMaintenance: vi.fn(),
      exportDatabase: vi.fn(),
    },
    adminCache: {
      getCacheStatus: vi.fn(),
      clearCache: vi.fn(),
      updateCacheConfig: vi.fn(),
    },
  },
}));

vi.mock('@/lib/services/db-manage', () => ({
  DbManageService: { getTableData: vi.fn() },
}));

describe('database overview', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(services.dbManage.getOverview).mockResolvedValue({
      type: 'postgres',
      version: '17.0',
      name: 'openflare',
      size: '2 GB',
      table_count: 12,
      connections: 0,
    });
    vi.mocked(services.dbManage.listTables).mockResolvedValue(['users']);
    vi.mocked(DbManageService.getTableData).mockResolvedValue({
      columns: ['id'],
      total: 0,
      results: [],
    });
    vi.mocked(services.adminCache.getCacheStatus).mockResolvedValue({
      total_size: 0,
      keys_count: 0,
      base_path: '/cache',
      max_size_mb: 100,
      ttl_minutes: 60,
      lru_enabled: true,
    });
  });

  function renderPage() {
    return render(
      <NextIntlClientProvider locale='en' messages={english} timeZone='UTC'>
        <DatabasePage />
      </NextIntlClientProvider>,
    );
  }

  it('loads only capacity initially and preserves lazy table data across tab switches', async () => {
    renderPage();
    expect(await screen.findByText('2 GB')).toBeVisible();
    expect(screen.getAllByRole('tab')).toHaveLength(2);
    expect(services.dbManage.listTables).not.toHaveBeenCalled();
    expect(services.adminCache.getCacheStatus).not.toHaveBeenCalled();
    expect(services.dbManage.getMaintenancePreview).not.toHaveBeenCalled();
    expect(services.dbManage.runMaintenance).not.toHaveBeenCalled();
    expect(services.dbManage.exportDatabase).not.toHaveBeenCalled();
    fireEvent.mouseDown(screen.getByRole('tab', { name: 'Table browser' }), {
      button: 0,
      ctrlKey: false,
    });
    await screen.findByText('No records in this table');
    expect(services.dbManage.listTables).toHaveBeenCalledTimes(1);
    expect(DbManageService.getTableData).toHaveBeenCalledTimes(1);
    fireEvent.mouseDown(screen.getByRole('tab', { name: 'Overview' }), {
      button: 0,
      ctrlKey: false,
    });
    fireEvent.mouseDown(screen.getByRole('tab', { name: 'Table browser' }), {
      button: 0,
      ctrlKey: false,
    });
    expect(services.dbManage.listTables).toHaveBeenCalledTimes(1);
    expect(DbManageService.getTableData).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole('button', { name: 'Refresh data' }));
    await waitFor(() =>
      expect(DbManageService.getTableData).toHaveBeenCalledTimes(2),
    );
    expect(services.dbManage.listTables).toHaveBeenCalledTimes(2);
    expect(services.dbManage.getOverview).toHaveBeenCalledTimes(1);
  });

  it('fetches cache once when opened and requires confirmation before clearing', async () => {
    renderPage();
    await screen.findByText('2 GB');
    fireEvent.click(
      screen.getByRole('button', { name: 'Load cache management' }),
    );
    const clear = await screen.findByRole('button', {
      name: 'Clear cache now',
    });
    expect(services.adminCache.getCacheStatus).toHaveBeenCalledTimes(1);
    fireEvent.click(clear);
    expect(screen.getByRole('dialog')).toBeVisible();
    expect(services.adminCache.clearCache).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    fireEvent.click(screen.getByRole('button', { name: 'Refresh data' }));
    await waitFor(() =>
      expect(services.adminCache.getCacheStatus).toHaveBeenCalledTimes(2),
    );
    expect(services.adminCache.updateCacheConfig).not.toHaveBeenCalled();
  });
});
