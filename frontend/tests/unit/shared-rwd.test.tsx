import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { TaskProgressCenter } from '@/components/providers/task-progress-center';
import { NodeKpiCard } from '@/app/(main)/nodes/components/node-detail-primitives';

const state = vi.hoisted(() => ({ mobile: true, pathname: '/admin/tasks' }));
vi.mock('next/navigation', () => ({ usePathname: () => state.pathname }));
vi.mock('next-intl', () => ({ useTranslations: () => (key: string) => key }));
vi.mock('@/contexts/user-context', () => ({
  useUser: () => ({ user: { id: 1 }, loading: false }),
}));
vi.mock('@/hooks/use-mobile', () => ({ useIsMobile: () => state.mobile }));
vi.mock('@/lib/services', () => ({ default: { adminTask: {} } }));
vi.mock('@tanstack/react-query', () => ({
  useQuery: ({ queryKey }: { queryKey: string[] }) => ({
    data:
      queryKey.includes('global') && queryKey.includes('running')
        ? {
            items: [
              {
                id: '1',
                task_id: 'task-1',
                task_name: 'Example task',
                status: 'running',
              },
            ],
          }
        : { items: [] },
  }),
}));

afterEach(() => {
  cleanup();
  state.mobile = true;
  state.pathname = '/admin/tasks';
  vi.restoreAllMocks();
});

describe('shared responsive components', () => {
  it('reveals a controlled active tab without scrolling the page', async () => {
    const rect = (left: number, right: number) => ({
      left,
      right,
      top: 0,
      bottom: 40,
      width: right - left,
      height: 40,
      x: left,
      y: 0,
      toJSON() {},
    });
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(
      function (this: HTMLElement) {
        const offset = this.closest('[role="tablist"]')?.scrollLeft ?? 0;
        return this.textContent === 'Second'
          ? rect(200 - offset, 280 - offset)
          : rect(0, 100);
      },
    );
    const content = (value: string) => (
      <Tabs value={value}>
        <TabsList>
          <TabsTrigger value='first'>First</TabsTrigger>
          <TabsTrigger value='second'>Second</TabsTrigger>
        </TabsList>
      </Tabs>
    );
    const view = render(content('first'));
    const list = screen.getByRole('tablist');
    expect(list.className).not.toContain('hide-scrollbar');
    view.rerender(content('second'));
    await waitFor(() => expect(list.scrollLeft).toBe(180));
    expect(screen.getByRole('tab', { name: 'Second' })).toHaveAttribute(
      'aria-selected',
      'true',
    );
  });

  it('keeps node KPI content within a shrinkable card', () => {
    render(
      <NodeKpiCard label='Node ID' value='node-with-a-very-long-identifier' />,
    );
    expect(
      screen.getByText('Node ID').parentElement?.parentElement,
    ).toHaveClass('min-w-0', 'border-border/70', 'rounded-lg');
  });

  it('keeps vertical tabs from scrolling horizontally', () => {
    render(
      <Tabs orientation='vertical' defaultValue='second'>
        <TabsList>
          <TabsTrigger value='first'>First</TabsTrigger>
          <TabsTrigger value='second'>Second</TabsTrigger>
        </TabsList>
      </Tabs>,
    );
    expect(screen.getByRole('tablist').scrollLeft).toBe(0);
  });

  it('starts mobile collapsed, preserves an open sheet across routes and restores focus on close', async () => {
    const view = render(<TaskProgressCenter />);
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    const trigger = screen.getByRole('button', { name: /activeTasks/ });
    trigger.focus();
    fireEvent.click(trigger);
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    state.pathname = '/admin/settings';
    view.rerender(<TaskProgressCenter />);
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Close' }));
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    );
    await waitFor(() => expect(trigger).toHaveFocus());
    expect(
      document.documentElement.style.getPropertyValue('--task-progress-inset'),
    ).not.toBe('');
    view.unmount();
    expect(
      document.documentElement.style.getPropertyValue('--task-progress-inset'),
    ).toBe('');
  });

  it('preserves the desktop collapse choice across routes', () => {
    state.mobile = false;
    const view = render(<TaskProgressCenter />);
    expect(screen.getByText('Example task')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'collapseTasks' }));
    state.pathname = '/admin/settings';
    view.rerender(<TaskProgressCenter />);
    expect(screen.queryByText('Example task')).not.toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'expandTasks' }),
    ).toBeInTheDocument();
  });
});
