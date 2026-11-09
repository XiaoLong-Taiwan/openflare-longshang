import { fireEvent, render, screen } from '@testing-library/react';
import { NextIntlClientProvider } from 'next-intl';
import type { ComponentProps } from 'react';
import { describe, expect, it, vi } from 'vitest';

import { GroupDialog } from '@/app/(main)/cloudflare/components/group-dialog';
import type { CloudflareGroup } from '@/lib/services/openflare';
import zhCN from '@/messages/zh-CN.json';

const node = { id: 1, name: '主节点', ip: '192.0.2.1' };
const group: CloudflareGroup = {
  id: 7,
  name: '生产节点',
  primary_node: node,
  backup_node: null,
  active_node: node,
  ttl: 900,
  default_proxied: true,
  enabled: true,
  member_count: 2,
  created_at: '',
  updated_at: '',
};

function renderDialog(
  overrides: Partial<ComponentProps<typeof GroupDialog>> = {},
) {
  const props: ComponentProps<typeof GroupDialog> = {
    open: true,
    onOpenChange: vi.fn(),
    nodes: [
      { ...node, node_type: 'edge_node' } as ComponentProps<
        typeof GroupDialog
      >['nodes'][number],
    ],
    pending: false,
    onSubmit: vi.fn(),
    ...overrides,
  };
  const view = (nextProps: ComponentProps<typeof GroupDialog>) => (
    <NextIntlClientProvider
      locale='zh-CN'
      messages={zhCN}
      timeZone='Asia/Shanghai'
    >
      <GroupDialog {...nextProps} />
    </NextIntlClientProvider>
  );
  const result = render(view(props));
  return {
    ...props,
    rerenderDialog: (nextProps: Partial<ComponentProps<typeof GroupDialog>>) =>
      result.rerender(view({ ...props, ...nextProps })),
  };
}

function fillCustomTarget() {
  fireEvent.change(screen.getByLabelText('分组名称'), {
    target: { value: '测试分组' },
  });
  fireEvent.click(screen.getByRole('checkbox'));
}

function manualInput() {
  return screen.getByRole('spinbutton', { name: 'TTL（秒）' });
}

function save() {
  fireEvent.click(screen.getByRole('button', { name: '保存' }));
}

describe('Cloudflare group TTL', () => {
  it.each(['A', 'AAAA', 'CNAME'])(
    'submits custom %s without requiring nodes',
    (recordType) => {
      const { onSubmit } = renderDialog({ nodes: [] });
      fireEvent.change(screen.getByLabelText('分组名称'), {
        target: { value: 'custom' },
      });
      fireEvent.click(screen.getByRole('radio', { name: '自定义 DNS' }));
      fireEvent.click(screen.getByRole('radio', { name: recordType }));
      fireEvent.change(screen.getByLabelText('记录值'), {
        target: { value: 'origin.example.net' },
      });
      save();
      expect(onSubmit).toHaveBeenCalledWith(
        expect.objectContaining({
          target_mode: 'custom',
          record_type: recordType,
          record_content: 'origin.example.net',
          nodes: [],
          ttl: 1,
        }),
      );
    },
  );

  it('restores custom target fields and preserves node selection while switching modes', () => {
    const { onSubmit } = renderDialog({
      group: {
        ...group,
        target_mode: 'custom',
        record_type: 'CNAME',
        record_content: 'origin.example.net',
      },
    });
    expect(screen.getByLabelText('记录值')).toHaveValue('origin.example.net');
    fireEvent.click(screen.getByRole('radio', { name: '多节点' }));
    expect(screen.getByRole('checkbox')).toBeChecked();
    save();
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({
        target_mode: 'node',
        nodes: [{ node_id: 1, priority: 0 }],
        record_content: '',
      }),
    );
  });

  it('prevents closing and resubmitting while pending', () => {
    const { onOpenChange, onSubmit } = renderDialog({ group, pending: true });
    fireEvent.click(screen.getByRole('button', { name: '取消' }));
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' });
    save();
    expect(onOpenChange).not.toHaveBeenCalled();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('defaults new groups to Auto and sends ttl=1', () => {
    const { onSubmit } = renderDialog();
    expect(screen.getByRole('radiogroup', { name: '分组 TTL' })).toBeVisible();
    expect(screen.getByRole('radio', { name: '自动' })).toBeChecked();
    expect(
      screen.queryByRole('spinbutton', { name: 'TTL（秒）' }),
    ).not.toBeInTheDocument();
    expect(screen.getByText(/Auto 当前约为 300 秒/)).toBeVisible();

    fillCustomTarget();
    save();
    expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ ttl: 1 }));
  });

  it('preserves multiple node selection while saving manual TTL', () => {
    const { onSubmit } = renderDialog({ group });
    fireEvent.change(manualInput(), { target: { value: '600' } });
    save();
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({
        ttl: 600,
        nodes: [{ node_id: 1, priority: 0 }],
      }),
    );
  });

  it('allows a new custom group to save a manual TTL', () => {
    const { onSubmit } = renderDialog();
    fillCustomTarget();
    fireEvent.click(screen.getByRole('radio', { name: '手动' }));
    fireEvent.change(manualInput(), { target: { value: '1200' } });
    save();
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ ttl: 1200 }),
    );
  });

  it('reflects Auto on edit and does not allow deselecting it', () => {
    const { onSubmit } = renderDialog({ group: { ...group, ttl: 1 } });
    fireEvent.click(screen.getByRole('radio', { name: '自动' }));
    expect(screen.getByRole('radio', { name: '自动' })).toBeChecked();
    expect(
      screen.queryByRole('spinbutton', { name: 'TTL（秒）' }),
    ).not.toBeInTheDocument();
    save();
    expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ ttl: 1 }));
  });

  it.each(['', '60.5', '59', '86401', '0', '-1'])(
    'rejects invalid manual TTL %j with an accessible error',
    (value) => {
      const { onSubmit } = renderDialog({ group });
      fireEvent.change(manualInput(), { target: { value } });
      expect(manualInput()).toHaveAttribute('aria-invalid', 'true');
      expect(manualInput()).toHaveAccessibleDescription(
        /请输入 60 至 86400 之间的整数秒数/,
      );
      expect(screen.getByRole('alert')).toHaveTextContent(
        '请输入 60 至 86400 之间的整数秒数。',
      );
      expect(screen.getByRole('button', { name: '保存' })).toBeDisabled();
      save();
      expect(onSubmit).not.toHaveBeenCalled();
    },
  );

  it.each([60, 86400])(
    'accepts the inclusive manual TTL boundary %i',
    (ttl) => {
      const { onSubmit } = renderDialog({ group });
      fireEvent.change(manualInput(), { target: { value: String(ttl) } });
      expect(manualInput()).toHaveAttribute('aria-invalid', 'false');
      save();
      expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ ttl }));
    },
  );

  it('switches invalid manual input to Auto and sends 1 without stale errors', () => {
    const { onSubmit } = renderDialog({ group });
    fireEvent.change(manualInput(), { target: { value: '' } });
    expect(screen.getByRole('alert')).toBeVisible();
    fireEvent.click(screen.getByRole('radio', { name: '自动' }));
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expect(
      screen.queryByRole('spinbutton', { name: 'TTL（秒）' }),
    ).not.toBeInTheDocument();
    save();
    expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ ttl: 1 }));
  });

  it('resets a new group to Auto after closing and reopening', () => {
    const { rerenderDialog, onOpenChange } = renderDialog();
    fireEvent.click(screen.getByRole('radio', { name: '手动' }));
    fireEvent.change(manualInput(), { target: { value: '' } });
    fireEvent.click(screen.getByRole('button', { name: '取消' }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
    rerenderDialog({ open: false });
    rerenderDialog({ open: true });
    expect(screen.getByRole('radio', { name: '自动' })).toBeChecked();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('radio', { name: '手动' }));
    expect(manualInput()).toHaveValue(300);
  });

  it('restores saved TTL after reopening and reflects a different group', () => {
    const { rerenderDialog } = renderDialog({ group });
    fireEvent.change(manualInput(), { target: { value: '600' } });
    rerenderDialog({ open: false });
    rerenderDialog({ open: true });
    expect(manualInput()).toHaveValue(900);
    rerenderDialog({ group: { ...group, id: 8, ttl: 1800 } });
    expect(manualInput()).toHaveValue(1800);
    rerenderDialog({ group: { ...group, id: 9, ttl: 1 } });
    expect(screen.getByRole('radio', { name: '自动' })).toBeChecked();
    expect(
      screen.queryByRole('spinbutton', { name: 'TTL（秒）' }),
    ).not.toBeInTheDocument();
  });

  it('uses 300 seconds for legacy responses without ttl', () => {
    const { ttl: _ttl, ...legacyGroup } = group;
    const { onSubmit } = renderDialog({
      group: legacyGroup as CloudflareGroup,
    });
    expect(manualInput()).toHaveValue(300);
    save();
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ ttl: 300 }),
    );
  });
});
