import { renderToString } from 'react-dom/server';
import { cleanup, render } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { ResponsiveSection } from '@/components/common/responsive-section';

afterEach(cleanup);

it('uses the Tailwind lg boundary and server-renders compact sections without children', () => {
  const matchMedia = vi.spyOn(window, 'matchMedia');
  const html = renderToString(
    <ResponsiveSection title='Trends'>
      <p>Chart content</p>
    </ResponsiveSection>,
  );
  expect(html).not.toContain('Chart content');
  render(
    <ResponsiveSection title='Trends'>
      <p>Chart content</p>
    </ResponsiveSection>,
  );
  expect(matchMedia).toHaveBeenCalledWith('(width < 64rem)');
  matchMedia.mockRestore();
});
