import { act, cleanup, fireEvent, render, screen } from '@testing-library/react';
import { NextIntlClientProvider } from 'next-intl';
import { afterEach, expect, it, vi } from 'vitest';
import messages from '@/messages/zh-CN.json';
import UsersPage from './page';

const { getUserDetail, updateUserStatus } = vi.hoisted(() => ({
  getUserDetail: vi.fn().mockResolvedValue({ id: 42 }),
  updateUserStatus: vi.fn().mockResolvedValue(undefined),
}));

vi.mock('@/contexts/admin-users-context', () => ({
  useAdminUsers: () => ({
    users: [{ id: 42, username: 'operator', nickname: 'Operator', email: 'operator@example.com', is_active: true, is_admin: false }],
    loading: false,
    error: null,
    fetchUsers: vi.fn(),
    getUserDetail,
    updateUserStatus,
  }),
}));
vi.mock('./components/user-filter-bar', () => ({ UserFilterBar: () => null }));
vi.mock('./components/create-user-modal', () => ({ CreateUserModal: () => null }));
vi.mock('./components/edit-user-modal', () => ({ EditUserModal: () => null }));
vi.mock('./components/user-detail-sheet', () => ({ UserDetailSheet: () => null }));

afterEach(() => { cleanup(); vi.clearAllMocks(); });

it('keeps identity and status in the mobile summary and provides named detail and status controls', async () => {
  render(<NextIntlClientProvider locale='zh-CN' messages={messages} timeZone='Asia/Shanghai'><UsersPage /></NextIntlClientProvider>);
  expect(screen.getByText('42 · operator@example.com')).toBeInTheDocument();
  expect(screen.getByText(`${messages.admin.users.status}: ${messages.admin.users.statusActive}`)).toBeInTheDocument();
  await act(async () => { fireEvent.click(screen.getByRole('button', { name: messages.admin.users.viewDetail })); });
  expect(getUserDetail).toHaveBeenCalledWith(42);
  await act(async () => { fireEvent.click(screen.getByRole('switch', { name: messages.admin.users.disableAccount })); });
  expect(updateUserStatus).toHaveBeenCalledWith(expect.objectContaining({ id: 42 }));
});
