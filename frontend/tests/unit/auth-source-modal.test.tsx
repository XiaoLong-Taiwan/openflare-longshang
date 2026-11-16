import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { NextIntlClientProvider } from 'next-intl';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { toast } from 'sonner';

import { AuthSourceModal } from '@/components/common/settings/auth-source-modal';
import services from '@/lib/services';
import type {
  AuthSource,
  AuthSourceType,
  SystemConfig,
} from '@/lib/services/admin';
import en from '@/messages/fragments/admin.en.json';
import zhCN from '@/messages/fragments/admin.zh-CN.json';
import zhTW from '@/messages/fragments/admin.zh-TW.json';
import ru from '@/messages/fragments/admin.ru.json';

vi.mock('@/lib/services', () => ({
  default: {
    adminSystemConfig: { listSystemConfigs: vi.fn() },
    adminAuthSource: { createAuthSource: vi.fn(), updateAuthSource: vi.fn() },
  },
}));
vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

function config(value: string): SystemConfig {
  return {
    key: 'server_address',
    value,
    type: 'system',
    visibility: 0,
    description: '',
    created_at: '',
    updated_at: '',
  };
}

function renderModal(
  source: AuthSource | null = null,
  messages = en,
  locale = 'en',
  isOpen = true,
) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <NextIntlClientProvider
        locale={locale}
        messages={messages}
        timeZone='UTC'
      >
        <AuthSourceModal
          isOpen={isOpen}
          source={source}
          onClose={vi.fn()}
          onChanged={vi.fn()}
        />
      </NextIntlClientProvider>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(services.adminSystemConfig.listSystemConfigs).mockResolvedValue([
    config('https://auth.example.com///'),
  ]);
  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: { writeText: vi.fn().mockResolvedValue(undefined) },
  });
});

describe('AuthSourceModal callback URL', () => {
  it.each<AuthSourceType>(['oidc', 'google', 'discord', 'github'])(
    'uses the shared configured /login URL when editing %s',
    async (type) => {
      renderModal({
        id: '12',
        name: `${type}-source`,
        type,
        display_name: type,
        is_active: true,
        client_id: 'client',
        openid_discovery_url: '',
        scopes: '',
        icon_url: '',
        created_at: '',
        updated_at: '',
      });
      const input = await screen.findByDisplayValue(
        'https://auth.example.com/login',
      );
      expect(input).toHaveAttribute('readonly');
      expect(input).not.toBeDisabled();
      fireEvent.focus(input);
      expect((input as HTMLInputElement).selectionEnd).toBe(
        (input as HTMLInputElement).value.length,
      );
      fireEvent.click(
        screen.getByRole('button', {
          name: en.settings.authSourceModal.copyCallbackURL,
        }),
      );
      await waitFor(() =>
        expect(toast.success).toHaveBeenCalledWith(
          en.settings.authSourceModal.callbackCopied,
        ),
      );
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
        'https://auth.example.com/login',
      );
      expect(services.adminAuthSource.updateAuthSource).not.toHaveBeenCalled();
    },
  );

  it.each([
    'https://auth.example.com',
    'https://auth.example.com/',
    'https://auth.example.com/portal///',
  ])(
    'preserves the configured base path for %s when creating a source',
    async (address) => {
      vi.mocked(services.adminSystemConfig.listSystemConfigs).mockResolvedValue(
        [config(address)],
      );
      renderModal();
      expect(
        await screen.findByDisplayValue(`${address.replace(/\/+$/, '')}/login`),
      ).toHaveAttribute('readonly');
      expect(services.adminSystemConfig.listSystemConfigs).toHaveBeenCalledWith(
        'system',
      );
    },
  );

  it.each([{ configs: [] }, { configs: [config('')] }])(
    'falls back to origin only when the setting is absent or empty: $configs',
    async ({ configs }) => {
      vi.mocked(services.adminSystemConfig.listSystemConfigs).mockResolvedValue(
        configs,
      );
      renderModal();
      expect(
        await screen.findByDisplayValue(`${window.location.origin}/login`),
      ).toBeVisible();
    },
  );

  it.each([
    ['en', en],
    ['zh-CN', zhCN],
    ['zh-TW', zhTW],
    ['ru', ru],
  ] as const)(
    'translates the label and copy feedback in %s',
    async (locale, messages) => {
      renderModal(null, messages, locale);
      expect(
        await screen.findByLabelText(
          messages.settings.authSourceModal.callbackURL,
        ),
      ).toHaveAttribute('readonly');
      const button = screen.getByRole('button', {
        name: messages.settings.authSourceModal.copyCallbackURL,
      });
      await waitFor(() => expect(button).toBeEnabled());
      fireEvent.click(button);
      await waitFor(() =>
        expect(toast.success).toHaveBeenCalledWith(
          messages.settings.authSourceModal.callbackCopied,
        ),
      );
      vi.mocked(navigator.clipboard.writeText).mockRejectedValue(
        new Error('denied'),
      );
      fireEvent.click(button);
      await waitFor(() =>
        expect(toast.error).toHaveBeenCalledWith(
          messages.settings.authSourceModal.callbackCopyFailed,
        ),
      );
    },
  );

  it('reports unavailable clipboard support without closing the dialog', async () => {
    renderModal();
    await screen.findByDisplayValue('https://auth.example.com/login');
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: undefined,
    });
    fireEvent.click(
      screen.getByRole('button', {
        name: en.settings.authSourceModal.copyCallbackURL,
      }),
    );
    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith(
        en.settings.authSourceModal.callbackCopyFailed,
      ),
    );
    expect(screen.getByRole('dialog')).toBeVisible();
  });

  it('keeps copying disabled while loading settings', () => {
    vi.mocked(services.adminSystemConfig.listSystemConfigs).mockReturnValue(
      new Promise(() => {}),
    );
    renderModal();
    expect(
      screen.getByRole('button', {
        name: en.settings.authSourceModal.copyCallbackURL,
      }),
    ).toBeDisabled();
    expect(
      screen.getByText(en.settings.authSourceModal.callbackLoading),
    ).toBeVisible();
  });

  it('does not substitute origin after a settings request failure', async () => {
    vi.mocked(services.adminSystemConfig.listSystemConfigs).mockRejectedValue(
      new Error('unavailable'),
    );
    renderModal();
    expect(
      await screen.findByText(en.settings.authSourceModal.callbackLoadFailed),
    ).toBeVisible();
    expect(
      screen.getByLabelText(en.settings.authSourceModal.callbackURL),
    ).toHaveValue('');
    expect(
      screen.getByRole('button', {
        name: en.settings.authSourceModal.copyCallbackURL,
      }),
    ).toBeDisabled();
  });

  it('does not fetch settings while closed', () => {
    renderModal(null, en, 'en', false);
    expect(services.adminSystemConfig.listSystemConfigs).not.toHaveBeenCalled();
  });
});
