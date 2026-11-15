import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { NextIntlClientProvider } from 'next-intl';
import { afterEach, expect, it, vi } from 'vitest';
import messages from '@/messages/zh-CN.json';
import { CertificateImportDialog } from './certificate-import-dialog';
import { CertificateApplyDialog } from './certificate-apply-dialog';
import { CertificateEditorDialog } from './certificate-editor-dialog';

const { create, apply, update } = vi.hoisted(() => ({ create: vi.fn(), apply: vi.fn(), update: vi.fn() }));

vi.mock('@/lib/services/openflare', () => ({
  TlsCertificateService: {
    create,
    apply,
    update,
    getDefaultAcmeAccount: vi.fn().mockResolvedValue(null),
    getContent: vi.fn().mockResolvedValue({ name: '', remark: '', cert_pem: '', key_pem: '', provider: 'upload' }),
  },
  DnsAccountService: { list: vi.fn().mockResolvedValue([]) },
}));

afterEach(() => { cleanup(); vi.clearAllMocks(); });

function renderDialog(dialog: React.ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <NextIntlClientProvider locale='zh-CN' messages={messages} timeZone='Asia/Shanghai'>
      <QueryClientProvider client={client}>{dialog}</QueryClientProvider>
    </NextIntlClientProvider>,
  );
}

async function expectInvalidFields(names: string[]) {
  for (const name of names) {
    const control = screen.queryByRole('tabpanel')
      ? within(screen.getByRole('tabpanel')).getByLabelText(name, { exact: true })
      : screen.getByLabelText(name, { exact: true });
    await waitFor(() => expect(control).toHaveAttribute('aria-invalid', 'true'));
    const errorId = control.getAttribute('aria-describedby');
    expect(errorId).toBeTruthy();
    expect(document.getElementById(errorId!)).toHaveAttribute('role', 'alert');
    expect(document.getElementById(errorId!)).not.toBeEmptyDOMElement();
  }
}

it('associates manual import labels and validation errors and prevents an invalid request', async () => {
  renderDialog(<CertificateImportDialog open onOpenChange={vi.fn()} />);
  fireEvent.submit(within(screen.getByRole('tabpanel')).getByLabelText(messages.certificates.name, { exact: true }).closest('form')!);
  await expectInvalidFields([messages.certificates.name, messages.certificates.certPem, messages.certificates.keyPem]);
  expect(create).not.toHaveBeenCalled();
});

it('shows name, primary domain and DNS account errors before applying', async () => {
  renderDialog(<CertificateApplyDialog open onOpenChange={vi.fn()} />);
  fireEvent.submit(screen.getByLabelText(messages.certificates.name, { exact: true }).closest('form')!);
  await expectInvalidFields([messages.certificates.name, messages.certificates.primaryDomain, messages.certificates.dnsAccount]);
  expect(apply).not.toHaveBeenCalled();
});

it('shows validation errors when editing a certificate', async () => {
  renderDialog(<CertificateEditorDialog open certificateId={1} onOpenChange={vi.fn()} />);
  const name = await screen.findByLabelText(messages.certificates.name, { exact: true });
  fireEvent.submit(name.closest('form')!);
  await expectInvalidFields([messages.certificates.name, messages.certificates.certPem, messages.certificates.keyPem]);
  expect(update).not.toHaveBeenCalled();
});

it('retains required file inputs when switching import tabs', async () => {
  const user = userEvent.setup();
  renderDialog(<CertificateImportDialog open onOpenChange={vi.fn()} />);
  await user.click(screen.getByRole('tab', { name: messages.certificates.fileImport }));
  await user.type(within(screen.getByRole('tabpanel')).getByLabelText(messages.certificates.name, { exact: true }), 'example');
  const cert = new File(['certificate'], 'example.crt', { type: 'text/plain' });
  const key = new File(['private key'], 'example.key', { type: 'text/plain' });
  await user.upload(screen.getByLabelText(messages.certificates.certFile), cert);
  await user.upload(screen.getByLabelText(messages.certificates.keyFile), key);
  await user.click(screen.getByRole('tab', { name: messages.certificates.manualImport }));
  await user.click(screen.getByRole('tab', { name: messages.certificates.fileImport }));
  const certInput = screen.getByLabelText(messages.certificates.certFile) as HTMLInputElement;
  const keyInput = screen.getByLabelText(messages.certificates.keyFile) as HTMLInputElement;
  expect(certInput.files?.[0]).toBe(cert);
  expect(keyInput.files?.[0]).toBe(key);
  expect(certInput).toBeRequired();
  expect(keyInput).toBeRequired();
  expect(within(screen.getByRole('tabpanel')).getByLabelText(messages.certificates.name, { exact: true })).toHaveValue('example');
  await user.click(screen.getByRole('button', { name: messages.certificates.clearFiles }));
  expect((screen.getByLabelText(messages.certificates.certFile) as HTMLInputElement).files).toHaveLength(0);
  expect((screen.getByLabelText(messages.certificates.keyFile) as HTMLInputElement).files).toHaveLength(0);
});
