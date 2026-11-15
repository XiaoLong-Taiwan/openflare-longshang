'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { AccessLogService } from '@/lib/services/openflare';
import { Button } from '@/components/ui/button';
import { NodeSectionCard } from './node-detail-primitives';
import { DistributionList } from './distribution-list';
import { formatBytes, formatMetricCount, getErrorMessage } from './node-utils';

export function NodeDomainTraffic({ nodeId }: { nodeId: string }) {
  const t = useTranslations('nodes.detailSections');
  const [host, setHost] = useState('');
  const hostsQuery = useQuery({
    queryKey: ['openflare', 'node-domains', nodeId],
    queryFn: () => AccessLogService.getOverview({ node_id: nodeId, hours: 24 }),
    enabled: nodeId.trim().length > 0,
    refetchInterval: 30000,
  });
  const detailQuery = useQuery({
    queryKey: ['openflare', 'node-domain-traffic', nodeId, host],
    queryFn: () => AccessLogService.getOverview({ node_id: nodeId, host, hours: 24 }),
    enabled: nodeId.trim().length > 0 && host.length > 0,
    refetchInterval: 30000,
  });
  const query = host ? detailQuery : hostsQuery;
  const data = query.data;

  return (
    <NodeSectionCard title={t('domains')} description={t('domainScope')}>
      <div className='flex flex-col gap-6'>
        <div className='flex flex-wrap gap-2'>
          <Button variant={host ? 'outline' : 'secondary'} onClick={() => setHost('')}>{t('allDomains')}</Button>
          {(hostsQuery.data?.top_hosts ?? []).map((item) => (
            <Button key={item.key} variant={host === item.key ? 'secondary' : 'outline'} onClick={() => setHost(item.key)}>{item.key}</Button>
          ))}
        </div>
        {query.isError ? <p role='alert'>{getErrorMessage(query.error, t('failed'))}</p> : query.isPending ? <p>{t('loading')}</p> : data ? <>
          <div className='grid gap-4 sm:grid-cols-3'>
            <div><p>{t('requestCount')}</p><p>{formatMetricCount(data.summary.total_requests)}</p></div>
            <div><p>{t('visitors')}</p><p>{formatMetricCount(data.summary.total_visits)}</p></div>
            <div><p>{t('bandwidth')}</p><p>{formatBytes(data.summary.bandwidth_served)}</p></div>
          </div>
          <div className='grid gap-6 md:grid-cols-2'>
            <div><p className='mb-3'>{t('paths')}</p><DistributionList items={data.top_paths.map((item) => ({ label: item.key, value: item.value }))} emptyMessage={t('empty')} /></div>
            <div><p className='mb-3'>{t('statusCodes')}</p><DistributionList items={data.status_codes.map((item) => ({ label: item.key, value: item.value }))} emptyMessage={t('empty')} /></div>
          </div>
        </> : <p>{t('empty')}</p>}
      </div>
    </NodeSectionCard>
  );
}
