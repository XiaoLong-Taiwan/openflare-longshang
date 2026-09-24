'use client';

import { useEffect, useMemo } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { z } from 'zod';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  TlsCertificateService,
  ZoneDomainService,
  ZoneService,
  zoneQueryKey,
  type ZoneDomainItem,
  type ZoneItem,
} from '@/lib/services/openflare';

import { useTranslations } from 'next-intl';

import {
  previewZoneDomainInput,
  resolveZoneDomainInput,
  type ZoneDomainInputError,
} from './resolve-zone-domain-input';

type Values = {
  zone_id: string;
  domain_input: string;
  cert_id: string;
  proxy_connect_timeout: string;
  proxy_send_timeout: string;
  proxy_read_timeout: string;
  client_max_body_size: string;
  websocket_enabled: 'inherit' | 'on' | 'off';
  proxy_request_buffering: 'inherit' | 'on' | 'off';
  proxy_buffering_enabled: 'inherit' | 'on' | 'off';
  custom_headers: string;
};

const headersToText = (headers?: Array<{ key: string; value: string }>) =>
  (headers ?? []).map(({ key, value }) => `${key}: ${value}`).join('\n');

const textToHeaders = (value: string) =>
  value
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const separator = line.indexOf(':');
      if (separator <= 0) throw new Error('invalidHeader');
      return {
        key: line.slice(0, separator).trim(),
        value: line.slice(separator + 1).trim(),
      };
    });

export function QuickCreateZoneDomainDialog({
  open,
  onOpenChange,
  fixedZoneId,
  fixedZoneRoot,
  zones: zonesProp,
  editingDomain,
  onCreated,
}: {
  open: boolean;
  onOpenChange(open: boolean): void;
  fixedZoneId?: number;
  fixedZoneRoot?: string;
  zones?: ZoneItem[];
  editingDomain?: ZoneDomainItem;
  onCreated(domain: ZoneDomainItem): void | Promise<void>;
}) {
  const t = useTranslations('websites');
  const tc = useTranslations('common');
  const queryClient = useQueryClient();
  const domainErrorMessage = (
    error: ZoneDomainInputError | undefined,
    root?: string,
  ) => {
    if (!error) return t('invalidFormat');
    if (error === 'mustBelongToZone') {
      return t('mustBelongToZone', { root: root ?? '' });
    }
    return t(error);
  };
  const zonesQuery = useQuery({
    queryKey: zoneQueryKey,
    queryFn: () => ZoneService.list(),
    enabled: open && !fixedZoneId && !zonesProp,
  });
  const certificatesQuery = useQuery({
    queryKey: ['openflare', 'tls-certificates'],
    queryFn: () => TlsCertificateService.list(),
    enabled: open,
  });

  const zones = useMemo(
    () => zonesProp ?? zonesQuery.data ?? [],
    [zonesProp, zonesQuery.data],
  );
  const fixedZone = useMemo(() => {
    if (!fixedZoneId) {
      return undefined;
    }
    return (
      zones.find((zone) => zone.id === fixedZoneId) ??
      (fixedZoneRoot
        ? ({
            id: fixedZoneId,
            domain: fixedZoneRoot,
            created_at: '',
            updated_at: '',
          } as ZoneItem)
        : undefined)
    );
  }, [fixedZoneId, fixedZoneRoot, zones]);

  const optionalSeconds = z
    .string()
    .refine(
      (value) => value === '' || /^[1-9]\d*$/.test(value),
      t('positiveSeconds'),
    );
  const schema = z.object({
    zone_id: z.string().min(1, t('selectZone')),
    domain_input: z.string().trim().min(1, t('enterDomain')),
    cert_id: z.string(),
    proxy_connect_timeout: optionalSeconds,
    proxy_send_timeout: optionalSeconds,
    proxy_read_timeout: optionalSeconds,
    client_max_body_size: z
      .string()
      .refine(
        (value) => value === '' || /^[1-9]\d*[kKmMgG]?$/.test(value),
        t('bodySizeInvalid'),
      ),
    websocket_enabled: z.enum(['inherit', 'on', 'off']),
    proxy_request_buffering: z.enum(['inherit', 'on', 'off']),
    proxy_buffering_enabled: z.enum(['inherit', 'on', 'off']),
    custom_headers: z.string(),
  });
  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      zone_id: fixedZoneId ? String(fixedZoneId) : '',
      domain_input: '',
      cert_id: '',
      proxy_connect_timeout: '',
      proxy_send_timeout: '',
      proxy_read_timeout: '',
      client_max_body_size: '',
      websocket_enabled: 'inherit',
      proxy_request_buffering: 'inherit',
      proxy_buffering_enabled: 'inherit',
      custom_headers: '',
    },
  });

  useEffect(() => {
    if (!open) {
      return;
    }
    const config = editingDomain?.nginx_config ?? {};
    form.reset({
      zone_id: fixedZoneId ? String(fixedZoneId) : '',
      domain_input: editingDomain?.domain ?? '',
      cert_id: editingDomain?.cert_id ? String(editingDomain.cert_id) : '',
      proxy_connect_timeout: config.proxy_connect_timeout
        ? String(config.proxy_connect_timeout)
        : '',
      proxy_send_timeout: config.proxy_send_timeout
        ? String(config.proxy_send_timeout)
        : '',
      proxy_read_timeout: config.proxy_read_timeout
        ? String(config.proxy_read_timeout)
        : '',
      client_max_body_size: config.client_max_body_size ?? '',
      websocket_enabled:
        config.websocket_enabled == null
          ? 'inherit'
          : config.websocket_enabled
            ? 'on'
            : 'off',
      proxy_request_buffering:
        config.proxy_request_buffering == null
          ? 'inherit'
          : config.proxy_request_buffering
            ? 'on'
            : 'off',
      proxy_buffering_enabled:
        config.proxy_buffering_enabled == null
          ? 'inherit'
          : config.proxy_buffering_enabled
            ? 'on'
            : 'off',
      custom_headers: headersToText(config.custom_headers),
    });
  }, [editingDomain, fixedZoneId, form, open]);

  const watchedZoneId = form.watch('zone_id');
  const watchedInput = form.watch('domain_input');
  const selectedZone = useMemo(() => {
    if (fixedZone) {
      return fixedZone;
    }
    const id = Number(watchedZoneId);
    return zones.find((zone) => zone.id === id);
  }, [fixedZone, watchedZoneId, zones]);

  const preview = selectedZone
    ? previewZoneDomainInput(watchedInput, selectedZone.domain)
    : '';

  const mutation = useMutation({
    mutationFn: async (values: Values) => {
      const zoneId = Number(values.zone_id);
      const zone = fixedZone ?? zones.find((item) => item.id === zoneId);
      if (!zone) {
        throw new Error(t('selectZone'));
      }
      const resolved = resolveZoneDomainInput(values.domain_input, zone.domain);
      if (resolved.error || !resolved.domain) {
        throw new Error(domainErrorMessage(resolved.error, zone.domain));
      }
      let customHeaders: Array<{ key: string; value: string }>;
      try {
        customHeaders = textToHeaders(values.custom_headers);
      } catch {
        throw new Error(t('domainHeadersInvalid'));
      }
      const payload = {
        domain: resolved.domain,
        cert_id: values.cert_id ? Number(values.cert_id) : null,
        nginx_config: {
          proxy_connect_timeout: Number(values.proxy_connect_timeout) || 0,
          proxy_send_timeout: Number(values.proxy_send_timeout) || 0,
          proxy_read_timeout: Number(values.proxy_read_timeout) || 0,
          client_max_body_size: values.client_max_body_size.trim(),
          ...(values.websocket_enabled === 'inherit'
            ? {}
            : { websocket_enabled: values.websocket_enabled === 'on' }),
          ...(values.proxy_request_buffering === 'inherit'
            ? {}
            : {
                proxy_request_buffering:
                  values.proxy_request_buffering === 'on',
              }),
          ...(values.proxy_buffering_enabled === 'inherit'
            ? {}
            : {
                proxy_buffering_enabled:
                  values.proxy_buffering_enabled === 'on',
              }),
          custom_headers: customHeaders,
        },
      };
      return editingDomain
        ? ZoneDomainService.update(zone.id, editingDomain.id, payload)
        : ZoneDomainService.create(zone.id, payload);
    },
    onSuccess: async (domain) => {
      toast.success(t(editingDomain ? 'domainUpdated' : 'domainAdded'), {
        description: domain.domain,
      });
      await Promise.all([
        onCreated(domain),
        queryClient.invalidateQueries({ queryKey: zoneQueryKey }),
        queryClient.invalidateQueries({
          queryKey: [...zoneQueryKey, 'all-domains'],
        }),
      ]);
      onOpenChange(false);
    },
    onError: (error) =>
      toast.error(error instanceof Error ? error.message : t('addFailed')),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {t(editingDomain ? 'editDomainTitle' : 'quickCreateTitle')}
          </DialogTitle>
          <DialogDescription>
            {t(editingDomain ? 'editDomainDesc' : 'quickCreateDesc')}
          </DialogDescription>
        </DialogHeader>

        <form
          id='quick-create-zone-domain'
          className='space-y-4'
          onSubmit={form.handleSubmit((values) => {
            if (!selectedZone) {
              form.setError('zone_id', { message: t('selectZone') });
              return;
            }
            const resolved = resolveZoneDomainInput(
              values.domain_input,
              selectedZone.domain,
            );
            if (resolved.error) {
              form.setError('domain_input', {
                message: domainErrorMessage(
                  resolved.error,
                  selectedZone.domain,
                ),
              });
              return;
            }
            mutation.mutate(values);
          })}
        >
          {!fixedZoneId ? (
            <div className='space-y-1.5'>
              <Label>Zone</Label>
              <Select
                value={form.watch('zone_id') || undefined}
                onValueChange={(value) => form.setValue('zone_id', value)}
              >
                <SelectTrigger>
                  <SelectValue placeholder={t('selectRegisteredRoot')} />
                </SelectTrigger>
                <SelectContent>
                  {zones.map((zone) => (
                    <SelectItem key={zone.id} value={String(zone.id)}>
                      {zone.domain}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {form.formState.errors.zone_id ? (
                <p className='text-xs text-destructive'>
                  {form.formState.errors.zone_id.message}
                </p>
              ) : null}
            </div>
          ) : (
            <div className='rounded-md border bg-muted/30 px-3 py-2 text-sm'>
              {t('zoneLabel')}
              <span className='ml-1 font-medium'>
                {fixedZoneRoot || fixedZone?.domain || `#${fixedZoneId}`}
              </span>
            </div>
          )}

          <div className='space-y-1.5'>
            <Label htmlFor='domain-input'>{t('domain')}</Label>
            <Input
              id='domain-input'
              placeholder={
                selectedZone
                  ? t('domainPlaceholderWithZone', {
                      domain: selectedZone.domain,
                    })
                  : t('domainPlaceholder')
              }
              {...form.register('domain_input')}
            />
            {preview ? (
              <p className='text-xs text-muted-foreground'>
                {t('willCreate')}
                <code className='ml-1 rounded bg-muted px-1 py-0.5 font-mono text-[11px]'>
                  {preview}
                </code>
              </p>
            ) : (
              <p className='text-xs text-muted-foreground'>
                {t.rich('domainExample', {
                  api: (chunks) => <code className='font-mono'>{chunks}</code>,
                  fqdn: (chunks) => <code className='font-mono'>{chunks}</code>,
                  apex: (chunks) => <code className='font-mono'>{chunks}</code>,
                })}
              </p>
            )}
            {form.formState.errors.domain_input ? (
              <p className='text-xs text-destructive'>
                {form.formState.errors.domain_input.message}
              </p>
            ) : null}
          </div>

          <div className='space-y-1.5'>
            <Label>{t('certOptional')}</Label>
            <Select
              value={form.watch('cert_id') || '__none'}
              onValueChange={(value) =>
                form.setValue('cert_id', value === '__none' ? '' : value)
              }
            >
              <SelectTrigger>
                <SelectValue placeholder={t('noCert')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='__none'>{t('noCert')}</SelectItem>
                {(certificatesQuery.data ?? []).map((certificate) => (
                  <SelectItem
                    key={certificate.id}
                    value={String(certificate.id)}
                  >
                    {certificate.name} · {certificate.primary_domain}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className='flex flex-col gap-3 border-t pt-4'>
            <div>
              <Label>{t('domainNginxRules')}</Label>
              <p className='mt-1 text-xs text-muted-foreground'>
                {t('domainNginxRulesDesc')}
              </p>
            </div>
            <div className='grid grid-cols-1 gap-3 sm:grid-cols-2'>
              {(
                [
                  ['proxy_connect_timeout', 'proxyConnectTimeout'],
                  ['proxy_send_timeout', 'proxySendTimeout'],
                  ['proxy_read_timeout', 'proxyReadTimeout'],
                ] as const
              ).map(([field, label]) => (
                <div key={field} className='flex flex-col gap-1.5'>
                  <Label htmlFor={field}>{t(label)}</Label>
                  <Input
                    id={field}
                    inputMode='numeric'
                    placeholder={t('inheritGlobal')}
                    {...form.register(field)}
                  />
                  {form.formState.errors[field] ? (
                    <p className='text-xs text-destructive'>
                      {form.formState.errors[field]?.message}
                    </p>
                  ) : null}
                </div>
              ))}
              <div className='flex flex-col gap-1.5'>
                <Label htmlFor='client_max_body_size'>
                  {t('clientMaxBodySize')}
                </Label>
                <Input
                  id='client_max_body_size'
                  placeholder={t('bodySizePlaceholder')}
                  {...form.register('client_max_body_size')}
                />
                {form.formState.errors.client_max_body_size ? (
                  <p className='text-xs text-destructive'>
                    {form.formState.errors.client_max_body_size.message}
                  </p>
                ) : null}
              </div>
            </div>
            {(
              [
                ['websocket_enabled', 'websocketEnabled'],
                ['proxy_request_buffering', 'proxyRequestBuffering'],
                ['proxy_buffering_enabled', 'proxyBufferingEnabled'],
              ] as const
            ).map(([field, label]) => (
              <div
                key={field}
                className='flex items-center justify-between gap-3'
              >
                <Label htmlFor={field}>{t(label)}</Label>
                <Select
                  value={form.watch(field)}
                  onValueChange={(value) =>
                    form.setValue(field, value as Values[typeof field])
                  }
                >
                  <SelectTrigger id={field} className='w-36'>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value='inherit'>
                      {t('inheritGlobal')}
                    </SelectItem>
                    <SelectItem value='on'>{t('enabled')}</SelectItem>
                    <SelectItem value='off'>{t('disabled')}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            ))}
            <div className='flex flex-col gap-1.5'>
              <Label htmlFor='domain-custom-headers'>
                {t('domainCustomHeaders')}
              </Label>
              <Textarea
                id='domain-custom-headers'
                className='min-h-20 font-mono'
                placeholder={'X-Header: value'}
                {...form.register('custom_headers')}
              />
              <p className='text-xs text-muted-foreground'>
                {t('domainCustomHeadersDesc')}
              </p>
            </div>
          </div>
        </form>

        <DialogFooter>
          <Button variant='outline' onClick={() => onOpenChange(false)}>
            {tc('cancel')}
          </Button>
          <Button
            type='submit'
            form='quick-create-zone-domain'
            disabled={mutation.isPending}
          >
            {mutation.isPending ? (
              <Loader2 className='mr-1 size-4 animate-spin' />
            ) : null}
            {t(editingDomain ? 'saveChanges' : 'addDomain')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
