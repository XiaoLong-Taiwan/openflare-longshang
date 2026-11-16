import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { ResponsiveSection } from '@/components/common/responsive-section';
import { TrendPreview } from '@/app/(main)/nodes/components/trend-preview';

const state = vi.hoisted(() => ({ compact: true }));
vi.mock('@/hooks/use-compact-layout', () => ({
  useCompactLayout: () => state.compact,
}));
vi.mock('next-intl', () => ({ useTranslations: () => (key: string) => key }));
afterEach(() => {
  cleanup();
  state.compact = true;
});

it('mounts a compact chart only when expanded and retains desktop content after a breakpoint change', () => {
  const view = render(
    <ResponsiveSection title='Trends'>
      <p>Chart content</p>
    </ResponsiveSection>,
  );
  expect(screen.queryByText('Chart content')).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Trends' }));
  expect(screen.getByText('Chart content')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Trends' }));
  expect(screen.queryByText('Chart content')).not.toBeInTheDocument();
  state.compact = false;
  view.rerender(
    <ResponsiveSection title='Trends'>
      <p>Chart content</p>
    </ResponsiveSection>,
  );
  expect(screen.getByText('Chart content')).toBeInTheDocument();
});

it('draws trend points and shows an explicit empty state without enough samples', () => {
  const view = render(<TrendPreview label='CPU' values={[0, 10, 30]} />);
  expect(
    screen.getByRole('img', { name: 'CPU' }).querySelector('polyline'),
  ).toHaveAttribute('points', '0,48 100,33.333333333333336 200,4');
  view.rerender(<TrendPreview label='CPU' values={[]} />);
  expect(screen.queryByRole('img')).not.toBeInTheDocument();
  expect(screen.getByText('trendEmpty')).toBeInTheDocument();
});
