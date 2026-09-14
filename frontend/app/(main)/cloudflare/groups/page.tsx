'use client';

import { useRouter } from 'next/navigation';
import { useEffect } from 'react';

export default function CloudflareGroupsPage() {
  const router = useRouter();

  useEffect(() => {
    router.replace('/cloudflare');
  }, [router]);

  return null;
}
