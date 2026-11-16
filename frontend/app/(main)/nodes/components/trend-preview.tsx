import { useTranslations } from 'next-intl';

export function TrendPreview({
  label,
  values,
}: {
  label: string;
  values: number[];
}) {
  const t = useTranslations('nodes');
  const maximum = Math.max(1, ...values);
  const points = values
    .map(
      (value, index) =>
        `${(index / Math.max(1, values.length - 1)) * 200},${48 - (value / maximum) * 44}`,
    )
    .join(' ');
  return (
    <div className='min-w-0 rounded-lg border p-3'>
      <p className='text-xs text-muted-foreground [overflow-wrap:anywhere]'>
        {label}
      </p>
      {values.length > 1 ? (
        <svg
          viewBox='0 0 200 52'
          role='img'
          aria-label={label}
          className='mt-2 h-14 w-full text-primary'
        >
          <polyline
            points={points}
            fill='none'
            stroke='currentColor'
            strokeWidth='2'
          />
        </svg>
      ) : (
        <p className='mt-2 flex h-14 items-center text-xs text-muted-foreground'>
          {t('trendEmpty')}
        </p>
      )}
    </div>
  );
}
