import { Globe } from 'lucide-react';
import { getTranslations } from 'next-intl/server';
import { Suspense } from 'react';

import { LoadingStateWithBorder } from '@/components/layout/loading';

import { ZonePageClient } from '../[zoneId]/page-client';

export default async function ZonePage() {
  const t = await getTranslations('websites');
  return (
    <Suspense
      fallback={
        <div className='py-6 px-1'>
          <LoadingStateWithBorder
            icon={Globe}
            description={t('loadingDetail')}
          />
        </div>
      }
    >
      <ZonePageClient />
    </Suspense>
  );
}
