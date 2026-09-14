import { render } from '@testing-library/react';
import { useRouter } from 'next/navigation';
import { describe, expect, it, vi } from 'vitest';

import CloudflareGroupsPage from '@/app/(main)/cloudflare/groups/page';

vi.mock('next/navigation', () => ({
  useRouter: vi.fn(),
}));

describe('Cloudflare groups redirect', () => {
  it('redirects in the browser without a build-time server redirect', async () => {
    const replace = vi.fn();
    vi.mocked(useRouter).mockReturnValue({ replace } as never);

    render(<CloudflareGroupsPage />);

    await vi.waitFor(() => expect(replace).toHaveBeenCalledWith('/cloudflare'));
  });
});
