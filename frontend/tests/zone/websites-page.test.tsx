import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import WebsitesPage from '@/app/(main)/websites/page';
import { ZoneService } from '@/lib/services/openflare';
import { NextIntlClientProvider } from 'next-intl';
import zhCN from '@/messages/zh-CN.json';

const pushMock = vi.fn();

vi.mock('next/link', () => ({
  default: ({
    children,
    href,
  }: {
    children: React.ReactNode;
    href: string;
  }) => <a href={href}>{children}</a>,
}));

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: pushMock }),
}));

vi.mock('@/lib/services/openflare', async (importOriginal) => {
  const actual =
    await importOriginal<typeof import('@/lib/services/openflare')>();
  return { ...actual, ZoneService: { list: vi.fn() } };
});

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <NextIntlClientProvider
      locale='zh-CN'
      messages={zhCN}
      timeZone='Asia/Shanghai'
    >
      <QueryClientProvider client={client}>
        <WebsitesPage />
      </QueryClientProvider>
    </NextIntlClientProvider>,
  );
}

describe('WebsitesPage', () => {
  it('filters zones, shows domain counts, and links every zone to the fixed detail route', async () => {
    vi.mocked(ZoneService.list).mockResolvedValue([
      {
        id: 42,
        domain: 'example.com',
        domain_count: 3,
        created_at: '',
        updated_at: '',
      },
      {
        id: 43,
        domain: 'another.com',
        domain_count: 0,
        created_at: '',
        updated_at: '',
      },
    ]);

    renderPage();

    expect(await screen.findByText('example.com')).toBeVisible();
    expect(screen.getByText('3')).toBeVisible();
    expect(screen.getByText('0')).toBeVisible();
    expect(screen.getByRole('columnheader', { name: '根域' })).toBeVisible();
    expect(screen.getAllByRole('link', { name: '管理' })[0]).toHaveAttribute(
      'href',
      '/websites/zone?id=42',
    );
    expect(screen.getAllByRole('link', { name: '管理' })).toHaveLength(2);
    expect(screen.getAllByRole('link', { name: '管理' })[1]).toHaveAttribute(
      'href',
      '/websites/zone?id=43',
    );
    fireEvent.click(screen.getByText('another.com'));
    expect(pushMock).toHaveBeenLastCalledWith('/websites/zone?id=43');
    fireEvent.click(screen.getByText('example.com'));
    expect(pushMock).toHaveBeenLastCalledWith('/websites/zone?id=42');

    fireEvent.change(screen.getByPlaceholderText('搜索 Zone 根域'), {
      target: { value: 'example' },
    });
    expect(screen.getByRole('link', { name: '管理' })).toHaveAttribute(
      'href',
      '/websites/zone?id=42',
    );
    expect(screen.queryByText('another.com')).not.toBeInTheDocument();
  });
});
