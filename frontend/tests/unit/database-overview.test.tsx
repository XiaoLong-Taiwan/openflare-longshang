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
    openflareAccessLog: {
      getStorageOperations: vi.fn(),
    },
    adminTask: {
      listTaskExecutions: vi.fn(),
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
      size_bytes: 2147483648,
      export_prerequisite: 'pg_dump_missing',
      table_count: 12,
      connections: 0,
    });
    vi.mocked(services.openflareAccessLog.getStorageOperations).mockResolvedValue({
      backend: 'clickhouse',
      backend_status: 'available',
      capacity: {
        status: 'available',
        scope: 'clickhouse_active_parts',
        bytes_on_disk: 1024 * 1024,
        active_parts: 3,
        total_rows: 2500,
      },
      node_logs: {
        status: 'available',
        oldest: '2026-11-01T00:00:00Z',
        newest: '2026-11-19T00:00:00Z',
      },
      user_logs: { status: 'empty' },
      outbox_status: 'available',
      outbox: {
        pending: 0,
        worker_running: true,
        healthy: true,
      },
    });
    vi.mocked(services.adminTask.listTaskExecutions).mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 1,
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

  it('loads operational status without side effects and preserves lazy table data across tab switches', async () => {
    renderPage();
    expect(await screen.findByText('2 GB')).toBeVisible();
    expect(await screen.findByText('1 MB')).toBeVisible();
    expect(screen.getByText('Active parts: 3; rows: 2,500')).toBeVisible();
    expect(screen.getByText(/This is not free disk space/)).toBeVisible();
    expect(screen.getByText('No retained data')).toBeVisible();
    expect(screen.getByText('No task record')).toBeVisible();
    expect(screen.getByText(/0 pending records; Healthy; Worker running/)).toBeVisible();
    expect(screen.getAllByRole('tab')).toHaveLength(2);
    expect(services.dbManage.listTables).not.toHaveBeenCalled();
    expect(services.adminCache.getCacheStatus).not.toHaveBeenCalled();
    expect(services.dbManage.getMaintenancePreview).not.toHaveBeenCalled();
    expect(services.dbManage.runMaintenance).not.toHaveBeenCalled();
    expect(services.dbManage.exportDatabase).not.toHaveBeenCalled();
    expect(services.openflareAccessLog.getStorageOperations).toHaveBeenCalledTimes(1);
    expect(services.adminTask.listTaskExecutions).toHaveBeenCalledTimes(2);
    expect(services.adminTask.listTaskExecutions).toHaveBeenNthCalledWith(1, {
      task_type_prefix: 'system:cleanup',
      page: 1,
      page_size: 1,
    });
    expect(services.adminTask.listTaskExecutions).toHaveBeenNthCalledWith(2, {
      task_type: 'database:maintenance',
      page: 1,
      page_size: 1,
    });
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

  it('displays confirmed zero for a successful empty ClickHouse result', async () => {
    vi.mocked(services.openflareAccessLog.getStorageOperations).mockResolvedValueOnce({
      backend: 'clickhouse',
      backend_status: 'available',
      capacity: {
        status: 'available',
        scope: 'clickhouse_active_parts',
        bytes_on_disk: 0,
        active_parts: 0,
        total_rows: 0,
      },
      node_logs: { status: 'empty' },
      user_logs: { status: 'empty' },
      outbox_status: 'unsupported',
    });
    renderPage();
    expect(await screen.findByText('Confirmed zero')).toBeVisible();
    expect(screen.getByText('Active parts: 0; rows: 0')).toBeVisible();
  });

  it('displays ClickHouse capacity failure without a zero or part counts', async () => {
    vi.mocked(services.openflareAccessLog.getStorageOperations).mockResolvedValueOnce({
      backend: 'clickhouse',
      backend_status: 'available',
      capacity: { status: 'error', scope: 'clickhouse_active_parts' },
      node_logs: { status: 'empty' },
      user_logs: { status: 'empty' },
      outbox_status: 'unsupported',
    });
    renderPage();
    expect(await screen.findByText('Query failed')).toBeVisible();
    expect(screen.getByText(/This is not free disk space/)).toBeVisible();
    expect(screen.queryByText('Confirmed zero')).not.toBeInTheDocument();
    expect(screen.queryByText(/^Active parts:/)).not.toBeInTheDocument();
  });

  it.each(['postgres', 'sqlite'])('shows shared primary capacity for %s without counting it twice', async (backend) => {
    vi.mocked(services.openflareAccessLog.getStorageOperations).mockResolvedValueOnce({
      backend,
      backend_status: 'available',
      capacity: { status: 'shared', scope: 'primary_database' },
      node_logs: { status: 'empty' },
      user_logs: { status: 'empty' },
      outbox_status: 'unsupported',
    });
    renderPage();
    expect(await screen.findByText('Shared with the primary database; not measured separately.')).toBeVisible();
    expect(screen.getByText('2 GB')).toBeVisible();
    expect(screen.queryByText(/This is not free disk space/)).not.toBeInTheDocument();
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
    expect(services.dbManage.getMaintenancePreview).not.toHaveBeenCalled();
    expect(services.dbManage.runMaintenance).not.toHaveBeenCalled();
    expect(services.dbManage.exportDatabase).not.toHaveBeenCalled();
    expect(services.adminTask.listTaskExecutions).toHaveBeenCalledTimes(4);
  });
});
