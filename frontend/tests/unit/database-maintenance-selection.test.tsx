import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { NextIntlClientProvider } from 'next-intl';
import { describe, expect, it, vi } from 'vitest';
import { MaintenanceCard } from '@/app/(main)/admin/database/components/maintenance-card';
import services from '@/lib/services';
import english from '@/messages/en.json';

vi.mock('@/lib/services', () => ({
  default: {
    dbManage: {
      getMaintenancePreview: vi.fn(),
      runMaintenance: vi.fn(),
    },
  },
}));

describe('database maintenance selection', () => {
  it('confirms and submits only the selected operations', async () => {
    vi.mocked(services.dbManage.getMaintenancePreview).mockResolvedValue({
      database_type: 'sqlite',
      expired_task_executions: 7,
      cache_keys: 3,
      cache_bytes: 100,
      operations: [],
    });
    vi.mocked(services.dbManage.runMaintenance).mockResolvedValue('task-1');
    render(
      <NextIntlClientProvider locale='en' messages={english} timeZone='UTC'>
        <MaintenanceCard />
      </NextIntlClientProvider>,
    );
    fireEvent.click(await screen.findByRole('checkbox', { name: 'Expired task records: 7' }));
    fireEvent.click(screen.getByRole('checkbox', { name: 'Disk cache: 3 items' }));
    fireEvent.click(screen.getByRole('button', { name: 'Preview and run maintenance' }));
    const dialog = screen.getByRole('alertdialog');
    expect(dialog).not.toHaveTextContent('deletes task records');
    expect(dialog).not.toHaveTextContent('clears the disk cache');
    expect(dialog).not.toHaveTextContent('Expired task records: 7');
    expect(dialog).not.toHaveTextContent('Disk cache: 3 items');
    expect(dialog).toHaveTextContent('Reclaim database space and maintain indexes');
    fireEvent.click(screen.getByRole('button', { name: 'Confirm and run' }));
    await waitFor(() => expect(services.dbManage.runMaintenance).toHaveBeenCalledWith({
      cleanup_task_executions: false,
      clear_disk_cache: false,
      optimize_database: true,
    }));
  });
});
