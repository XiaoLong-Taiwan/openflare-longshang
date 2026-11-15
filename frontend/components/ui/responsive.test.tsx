import * as React from 'react';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { Button } from './button';
import { Input } from './input';
import {
  Dialog, DialogContent, DialogDescription, DialogFooter,
  DialogHeader, DialogTitle, DialogTrigger,
} from './dialog';
import { Sheet, SheetContent, SheetDescription, SheetTitle, SheetTrigger } from './sheet';
import { Select, SelectTrigger, SelectValue } from './select';
import { Sidebar, SidebarContent, SidebarFooter, SidebarProvider, SidebarTrigger } from './sidebar';
import { Tabs, TabsContent, TabsList, TabsTrigger } from './tabs';

const viewport = vi.hoisted(() => ({ mobile: true }));
vi.mock('@/hooks/use-mobile', () => ({ useIsMobile: () => viewport.mobile }));

afterEach(() => {
  cleanup();
  viewport.mobile = true;
});

describe('responsive shared controls', () => {
  it('keeps form semantics, disabled behavior and asChild icon sizing metadata', async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(
      <>
        <Input aria-label='Name' />
        <Button disabled onClick={onClick}>Save</Button>
        <Button asChild size='icon-sm'><a href='#settings' aria-label='Settings'>S</a></Button>
        <Select disabled><SelectTrigger aria-label='Region'><SelectValue placeholder='Region' /></SelectTrigger></Select>
      </>,
    );
    await user.type(screen.getByRole('textbox'), 'Mobile');
    expect(screen.getByRole('textbox')).toHaveValue('Mobile');
    await user.click(screen.getByRole('button', { name: 'Save' }));
    expect(onClick).not.toHaveBeenCalled();
    expect(screen.getByRole('link')).toHaveAttribute('data-size', 'icon-sm');
    expect(screen.getByRole('combobox')).toBeDisabled();
  });

  it('applies mobile control sizes outside the utility layer so compact caller classes cannot shrink targets', () => {
    const css = readFileSync(resolve(process.cwd(), 'app/globals.css'), 'utf8');
    const mobileStyles = css.slice(css.indexOf('@media (max-width: 767px)'));
    expect(mobileStyles).toContain("[data-slot='input']");
    expect(mobileStyles).toContain("[data-slot='button'][data-size^='icon']");
    expect(mobileStyles).toContain('min-width: 2.75rem');
    expect(mobileStyles).toContain('min-height: 2.75rem');
    expect(mobileStyles).toContain('font-size: 1rem');
  });
});

describe('responsive overlays', () => {
  it('bounds long dialogs and preserves Escape dismissal and trigger focus', async () => {
    const user = userEvent.setup();
    render(
      <Dialog>
        <DialogTrigger>Open dialog</DialogTrigger>
        <DialogContent>
          <DialogHeader><DialogTitle>Preferences</DialogTitle><DialogDescription>Long form</DialogDescription></DialogHeader>
          <Input aria-label='Value' />
          <DialogFooter><Button>Cancel</Button><Button>Save</Button></DialogFooter>
        </DialogContent>
      </Dialog>,
    );
    const trigger = screen.getByRole('button', { name: 'Open dialog' });
    await user.click(trigger);
    const dialog = screen.getByRole('dialog');
    expect(dialog.className).toContain('100dvh');
    expect(dialog).toHaveClass('overflow-y-auto', 'overscroll-contain');
    expect(dialog.querySelector('[data-slot=dialog-footer]')).toHaveClass('flex-col-reverse', 'sm:flex-row');
    await user.keyboard('{Escape}');
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });

  it.each(['left', 'right', 'top', 'bottom'] as const)('constrains the %s sheet and keeps its close button usable', async (side) => {
    const user = userEvent.setup();
    render(
      <Sheet>
        <SheetTrigger>Open sheet</SheetTrigger>
        <SheetContent side={side}><SheetTitle>Details</SheetTitle><SheetDescription>Long content</SheetDescription></SheetContent>
      </Sheet>,
    );
    await user.click(screen.getByRole('button', { name: 'Open sheet' }));
    expect(screen.getByRole('dialog')).toHaveClass('max-h-dvh', 'max-w-full', 'overflow-y-auto');
    await user.click(screen.getByRole('button', { name: 'Close' }));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('resets a mobile sidebar when crossing the desktop breakpoint', async () => {
    const user = userEvent.setup();
    function Shell() {
      return (
        <SidebarProvider>
          <SidebarTrigger />
          <Sidebar><SidebarContent>Navigation</SidebarContent><SidebarFooter>Account</SidebarFooter></Sidebar>
        </SidebarProvider>
      );
    }
    const { rerender } = render(<Shell />);
    await user.click(screen.getByRole('button', { name: 'Toggle Sidebar' }));
    expect(screen.getByRole('dialog')).toHaveAttribute('data-mobile', 'true');
    expect(screen.getByRole('dialog')).toHaveClass('overflow-hidden', 'gap-0');
    await user.keyboard('{Escape}');
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Toggle Sidebar' }));
    viewport.mobile = false;
    rerender(<Shell />);
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    viewport.mobile = true;
    rerender(<Shell />);
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});

describe('responsive tabs', () => {
  it.each(['default', 'line'] as const)('keeps %s tabs bounded and keyboard navigation skips disabled tabs', async (variant) => {
    const user = userEvent.setup();
    render(
      <Tabs defaultValue='first'>
        <TabsList variant={variant}>
          <TabsTrigger value='first'>First long tab</TabsTrigger>
          <TabsTrigger value='disabled' disabled>Disabled</TabsTrigger>
          <TabsTrigger value='last'>Last long tab</TabsTrigger>
        </TabsList>
        <TabsContent value='first'>First panel</TabsContent>
        <TabsContent value='last'>Last panel</TabsContent>
      </Tabs>,
    );
    expect(screen.getByRole('tablist')).toHaveClass('max-w-full', 'group-data-[orientation=horizontal]/tabs:overflow-x-auto');
    await user.tab();
    await user.keyboard('{ArrowRight}');
    expect(screen.getByRole('tab', { name: 'Last long tab' })).toHaveFocus();
    expect(screen.getByRole('tab', { name: 'Last long tab' })).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByRole('tabpanel')).toHaveTextContent('Last panel');
  });

  it('preserves vertical tab orientation and arrow key behavior', async () => {
    const user = userEvent.setup();
    render(
      <Tabs orientation='vertical' defaultValue='one'>
        <TabsList><TabsTrigger value='one'>One</TabsTrigger><TabsTrigger value='two'>Two</TabsTrigger></TabsList>
        <TabsContent value='one'>Panel one</TabsContent><TabsContent value='two'>Panel two</TabsContent>
      </Tabs>,
    );
    expect(screen.getByRole('tablist')).toHaveAttribute('aria-orientation', 'vertical');
    await user.tab();
    await user.keyboard('{ArrowDown}');
    expect(screen.getByRole('tabpanel')).toHaveTextContent('Panel two');
  });
});
