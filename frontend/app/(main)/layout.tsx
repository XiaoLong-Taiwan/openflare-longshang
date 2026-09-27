'use client';

import { motion } from 'motion/react';
import { usePathname, useRouter } from 'next/navigation';
import { useEffect, useMemo, useState } from 'react';
import { AppSidebar } from '@/components/layout/sidebar';
import { SiteHeader } from '@/components/layout/header';
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar';
import { useAuthRedirect } from '@/hooks/use-auth-redirect';
import { usePublicConfig } from '@/hooks/use-public-config';

export default function MainLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const router = useRouter();
  const [isFullWidth, setIsFullWidth] = useState(false);
  const { config, loading: configLoading } = usePublicConfig();
  const disabledPaths = useMemo(() => {
    if (!config?.menu_display_config) return new Set<string>();
    try {
      const parsed = JSON.parse(config.menu_display_config) as Record<
        string,
        unknown
      >;
      return new Set(
        Object.entries(parsed)
          .filter(([, enabled]) => enabled === false)
          .map(([path]) => path),
      );
    } catch {
      return new Set<string>();
    }
  }, [config?.menu_display_config]);

  useAuthRedirect();

  useEffect(() => {
    if (
      configLoading ||
      pathname === '/' ||
      pathname.startsWith('/admin/settings')
    )
      return;
    const disabled = Array.from(disabledPaths).some(
      (path) => pathname === path || pathname.startsWith(`${path}/`),
    );
    if (disabled) router.replace('/');
  }, [configLoading, disabledPaths, pathname, router]);

  return (
    <SidebarProvider
      className='h-screen'
      style={
        {
          '--header-height': '60px',
        } as React.CSSProperties
      }
    >
      <AppSidebar />
      <SidebarInset className='flex flex-col min-w-0 h-screen'>
        <SiteHeader
          isFullWidth={isFullWidth}
          onToggleFullWidth={setIsFullWidth}
        />
        <div className='flex flex-1 flex-col bg-background overflow-y-auto overflow-x-hidden min-w-0 hide-scrollbar'>
          <div
            className={`w-full mx-auto px-4 sm:px-6 md:px-8 lg:px-12 min-w-0 transition-all duration-300 ease-in-out ${!isFullWidth ? 'max-w-[1320px]' : 'max-w-full'}`}
          >
            <motion.div
              key={pathname}
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{
                duration: 0.5,
                ease: 'easeOut',
              }}
              className='w-full'
            >
              {children}
            </motion.div>
          </div>
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
