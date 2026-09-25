'use client';

import { useEffect } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useQuery } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import type {
  ProxyRouteItem,
  ProxyRouteLoadBalancing,
  ProxyRouteProxyConfig,
  ProxyRouteUpstreamTarget,
} from '@/lib/services/openflare';
import { NodeService, PagesService } from '@/lib/services/openflare';

import { useTranslations } from 'next-intl';

import {
  customHeadersToText,
  parseCustomHeadersText,
  parseOriginUrl,
  parseOriginUrls,
  validateOriginHost,
} from '../../components/helpers';
import { UpstreamTargetEditor } from '../../components/upstream-target-editor';
import { proxyRouteFormIds } from '../helpers';
import { useRouteSectionSave } from '../hooks/use-route-section-save';
import { SectionShell } from './section-shell';

type ProxyBooleanValue = 'inherit' | 'on' | 'off';

type ReverseProxyValues = {
  upstream_type: 'direct' | 'tunnel' | 'pages';
  upstream_targets: ProxyRouteUpstreamTarget[];
  load_balancing: ProxyRouteLoadBalancing;
  origin_host: string;
  tunnel_id?: string;
  tunnel_target_addr?: string;
  tunnel_target_protocol?: 'http' | 'https';
  pages_project_id?: string;
  custom_headers_text: string;
  proxy_connect_timeout: string;
  proxy_send_timeout: string;
  proxy_read_timeout: string;
  client_header_timeout: string;
  client_body_timeout: string;
  send_timeout: string;
  client_max_body_size: string;
  websocket_enabled: ProxyBooleanValue;
  proxy_request_buffering: ProxyBooleanValue;
  proxy_buffering_enabled: ProxyBooleanValue;
};

interface ProxySectionProps {
  route: ProxyRouteItem;
  onRouteUpdate: (route: ProxyRouteItem) => void;
  onSavingChange?: (saving: boolean) => void;
}

export function ProxySection({
  route,
  onRouteUpdate,
  onSavingChange,
}: ProxySectionProps) {
  const t = useTranslations('proxyRoutes');
  const proxyTimeoutPattern = /^[1-9]\d*(?:ms|s|m|h|d|w)$/;
  const reverseProxySchema = z
    .object({
      upstream_type: z.enum(['direct', 'tunnel', 'pages']),
      upstream_targets: z.array(
        z.object({ url: z.string(), priority: z.number().int().min(0) }),
      ),
      load_balancing: z.enum(['round_robin', 'least_conn']),
      origin_host: z.string(),
      tunnel_id: z.string().optional(),
      tunnel_target_addr: z.string().trim().optional(),
      tunnel_target_protocol: z.enum(['http', 'https']).optional(),
      pages_project_id: z.string().optional(),
      custom_headers_text: z.string(),
      proxy_connect_timeout: z.string(),
      proxy_send_timeout: z.string(),
      proxy_read_timeout: z.string(),
      client_header_timeout: z.string(),
      client_body_timeout: z.string(),
      send_timeout: z.string(),
      client_max_body_size: z.string(),
      websocket_enabled: z.enum(['inherit', 'on', 'off']),
      proxy_request_buffering: z.enum(['inherit', 'on', 'off']),
      proxy_buffering_enabled: z.enum(['inherit', 'on', 'off']),
    })
    .superRefine((value, context) => {
      if (value.upstream_type === 'direct') {
        const originUrls = value.upstream_targets
          .map((target) => target.url.trim())
          .filter(Boolean);
        if (originUrls.length === 0) {
          context.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['upstream_targets'],
            message: t('validation.enterAtLeastOneUpstream'),
          });
        } else {
          const { error } = parseOriginUrls(originUrls.join('\n'), t);
          if (error) {
            context.addIssue({
              code: z.ZodIssueCode.custom,
              path: ['upstream_targets'],
              message: error,
            });
          }
        }
      } else if (value.upstream_type === 'tunnel') {
        if (!value.tunnel_id) {
          context.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['tunnel_id'],
            message: t('validation.selectTunnel'),
          });
        }
        if (!value.tunnel_target_addr) {
          context.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['tunnel_target_addr'],
            message: t('validation.enterTunnelTarget'),
          });
        }
      } else if (!value.pages_project_id) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['pages_project_id'],
          message: t('validation.selectPagesProject'),
        });
      }

      const originHostError = validateOriginHost(value.origin_host, t);
      if (originHostError) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['origin_host'],
          message: originHostError,
        });
      }

      const { error: headerError } = parseCustomHeadersText(
        value.custom_headers_text,
        t,
      );
      if (headerError) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['custom_headers_text'],
          message: headerError,
        });
      }

      for (const field of [
        'proxy_connect_timeout',
        'proxy_send_timeout',
        'proxy_read_timeout',
        'client_header_timeout',
        'client_body_timeout',
        'send_timeout',
      ] as const) {
        if (value[field] && !proxyTimeoutPattern.test(value[field].trim())) {
          context.addIssue({
            code: z.ZodIssueCode.custom,
            path: [field],
            message: t('validation.timeoutInvalid'),
          });
        }
      }

      if (
        value.client_max_body_size &&
        !/^[1-9]\d*[kKmMgG]?$/.test(value.client_max_body_size)
      ) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['client_max_body_size'],
          message: t('validation.bodySizeInvalid'),
        });
      }
    });
  const { saving, save } = useRouteSectionSave(
    route,
    onRouteUpdate,
    onSavingChange,
  );

  const tunnelsQuery = useQuery({
    queryKey: ['openflare', 'nodes'],
    queryFn: () => NodeService.listNodes(),
  });

  const pagesProjectsQuery = useQuery({
    queryKey: ['openflare', 'pages-projects'],
    queryFn: () => PagesService.listProjects(),
  });

  const tunnelClients = (tunnelsQuery.data ?? []).filter(
    (node) => node.node_type === 'tunnel_client',
  );
  const pagesProjects = (pagesProjectsQuery.data ?? []).filter(
    (project) => project.enabled && project.active_deployment_id,
  );

  const form = useForm<ReverseProxyValues>({
    resolver: zodResolver(reverseProxySchema),
    defaultValues: {
      upstream_type: route.upstream_type || 'direct',
      upstream_targets:
        route.upstream_targets?.length > 0
          ? route.upstream_targets
          : route.upstream_list.map((url) => ({ url, priority: 0 })),
      load_balancing: route.load_balancing || 'round_robin',
      origin_host: route.origin_host || '',
      tunnel_id: route.tunnel_node_id ? String(route.tunnel_node_id) : '',
      tunnel_target_addr: route.tunnel_target_addr || '',
      tunnel_target_protocol:
        (route.tunnel_target_protocol as 'http' | 'https') || 'http',
      pages_project_id: route.pages_project_id
        ? String(route.pages_project_id)
        : '',
      custom_headers_text: customHeadersToText(route.custom_header_list),
      proxy_connect_timeout: String(
        route.proxy_config?.proxy_connect_timeout || '',
      ),
      proxy_send_timeout: String(route.proxy_config?.proxy_send_timeout || ''),
      proxy_read_timeout: String(route.proxy_config?.proxy_read_timeout || ''),
      client_header_timeout: String(
        route.proxy_config?.client_header_timeout || '',
      ),
      client_body_timeout: String(
        route.proxy_config?.client_body_timeout || '',
      ),
      send_timeout: String(route.proxy_config?.send_timeout || ''),
      client_max_body_size: route.proxy_config?.client_max_body_size || '',
      websocket_enabled:
        route.proxy_config?.websocket_enabled == null
          ? 'inherit'
          : route.proxy_config.websocket_enabled
            ? 'on'
            : 'off',
      proxy_request_buffering:
        route.proxy_config?.proxy_request_buffering == null
          ? 'inherit'
          : route.proxy_config.proxy_request_buffering
            ? 'on'
            : 'off',
      proxy_buffering_enabled:
        route.proxy_config?.proxy_buffering_enabled == null
          ? 'inherit'
          : route.proxy_config.proxy_buffering_enabled
            ? 'on'
            : 'off',
    },
  });

  useEffect(() => {
    form.reset({
      upstream_type: route.upstream_type || 'direct',
      upstream_targets:
        route.upstream_targets?.length > 0
          ? route.upstream_targets
          : route.upstream_list.map((url) => ({ url, priority: 0 })),
      load_balancing: route.load_balancing || 'round_robin',
      origin_host: route.origin_host || '',
      tunnel_id: route.tunnel_node_id ? String(route.tunnel_node_id) : '',
      tunnel_target_addr: route.tunnel_target_addr || '',
      tunnel_target_protocol:
        (route.tunnel_target_protocol as 'http' | 'https') || 'http',
      pages_project_id: route.pages_project_id
        ? String(route.pages_project_id)
        : '',
      custom_headers_text: customHeadersToText(route.custom_header_list),
      proxy_connect_timeout: String(
        route.proxy_config?.proxy_connect_timeout || '',
      ),
      proxy_send_timeout: String(route.proxy_config?.proxy_send_timeout || ''),
      proxy_read_timeout: String(route.proxy_config?.proxy_read_timeout || ''),
      client_header_timeout: String(
        route.proxy_config?.client_header_timeout || '',
      ),
      client_body_timeout: String(
        route.proxy_config?.client_body_timeout || '',
      ),
      send_timeout: String(route.proxy_config?.send_timeout || ''),
      client_max_body_size: route.proxy_config?.client_max_body_size || '',
      websocket_enabled:
        route.proxy_config?.websocket_enabled == null
          ? 'inherit'
          : route.proxy_config.websocket_enabled
            ? 'on'
            : 'off',
      proxy_request_buffering:
        route.proxy_config?.proxy_request_buffering == null
          ? 'inherit'
          : route.proxy_config.proxy_request_buffering
            ? 'on'
            : 'off',
      proxy_buffering_enabled:
        route.proxy_config?.proxy_buffering_enabled == null
          ? 'inherit'
          : route.proxy_config.proxy_buffering_enabled
            ? 'on'
            : 'off',
    });
  }, [form, route]);

  const upstreamType = form.watch('upstream_type');

  return (
    <SectionShell
      title={t('reverseProxy')}
      description={t('reverseProxyDesc')}
      formId={proxyRouteFormIds.proxy}
      saving={saving}
    >
      <Form {...form}>
        <form
          id={proxyRouteFormIds.proxy}
          className='space-y-5'
          onSubmit={form.handleSubmit(async (values) => {
            let originUrl = '';
            let originScheme: 'http' | 'https' = 'http';
            let originAddress = '';
            let originPort = '';
            let originUri = '';
            let upstreams: string[] = [];
            const upstreamTargets = values.upstream_targets
              .filter((target) => target.url.trim())
              .map((target) => ({ ...target, url: target.url.trim() }));

            if (values.upstream_type === 'direct') {
              const { urls } = parseOriginUrls(
                upstreamTargets.map((target) => target.url).join('\n'),
                t,
              );
              const primaryOrigin = parseOriginUrl(urls[0]);
              originUrl = urls[0];
              originScheme = primaryOrigin.scheme;
              originAddress = primaryOrigin.address;
              originPort = primaryOrigin.port;
              originUri = primaryOrigin.uri;
              upstreams = urls.slice(1);
            } else if (values.upstream_type === 'tunnel') {
              originUrl = `${values.tunnel_target_protocol}://${values.tunnel_target_addr}`;
              originScheme = values.tunnel_target_protocol as 'http' | 'https';
              originAddress = values.tunnel_target_addr || '';
            } else {
              originUrl = 'http://127.0.0.1';
              originScheme = 'http';
              originAddress = '127.0.0.1';
              originPort = '80';
            }

            const { headers } = parseCustomHeadersText(
              values.custom_headers_text,
              t,
            );
            const proxyConfig: ProxyRouteProxyConfig = {
              proxy_connect_timeout: values.proxy_connect_timeout.trim(),
              proxy_send_timeout: values.proxy_send_timeout.trim(),
              proxy_read_timeout: values.proxy_read_timeout.trim(),
              client_header_timeout: values.client_header_timeout.trim(),
              client_body_timeout: values.client_body_timeout.trim(),
              send_timeout: values.send_timeout.trim(),
              client_max_body_size: values.client_max_body_size.trim(),
              websocket_enabled:
                values.websocket_enabled === 'inherit'
                  ? null
                  : values.websocket_enabled === 'on',
              proxy_request_buffering:
                values.proxy_request_buffering === 'inherit'
                  ? null
                  : values.proxy_request_buffering === 'on',
              proxy_buffering_enabled:
                values.proxy_buffering_enabled === 'inherit'
                  ? null
                  : values.proxy_buffering_enabled === 'on',
              custom_headers: headers,
            };

            await save(
              {
                origin_id: null,
                origin_url: originUrl,
                origin_scheme: originScheme,
                origin_address: originAddress,
                origin_port: originPort,
                origin_uri: originUri,
                origin_host: values.origin_host.trim(),
                upstreams,
                upstream_targets: upstreamTargets,
                load_balancing: values.load_balancing,
                proxy_config: proxyConfig,
                custom_headers: headers,
                upstream_type: values.upstream_type,
                tunnel_node_id:
                  values.upstream_type === 'tunnel' && values.tunnel_id
                    ? Number(values.tunnel_id)
                    : null,
                tunnel_target_addr:
                  values.upstream_type === 'tunnel'
                    ? values.tunnel_target_addr
                    : '',
                tunnel_target_protocol:
                  values.upstream_type === 'tunnel'
                    ? values.tunnel_target_protocol
                    : '',
                pages_project_id:
                  values.upstream_type === 'pages' && values.pages_project_id
                    ? Number(values.pages_project_id)
                    : null,
              },
              t('proxySaved'),
            );
          })}
        >
          <FormField
            control={form.control}
            name='upstream_type'
            render={({ field }) => (
              <FormItem className='space-y-3'>
                <FormLabel>{t('upstreamType')}</FormLabel>
                <div className='flex flex-wrap gap-4'>
                  {(
                    [
                      ['direct', t('upstreamDirect')],
                      ['tunnel', t('upstreamTunnel')],
                      ['pages', t('upstreamPages')],
                    ] as const
                  ).map(([value, label]) => (
                    <label
                      key={value}
                      className='flex cursor-pointer items-center gap-2 text-sm'
                    >
                      <input
                        type='radio'
                        value={value}
                        checked={field.value === value}
                        onChange={() => field.onChange(value)}
                        className='size-4 accent-primary'
                      />
                      <Label className='font-normal'>{label}</Label>
                    </label>
                  ))}
                </div>
                <FormMessage />
              </FormItem>
            )}
          />

          {upstreamType === 'direct' ? (
            <FormField
              control={form.control}
              name='upstream_targets'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('upstreamAddresses')}</FormLabel>
                  <FormControl>
                    <UpstreamTargetEditor
                      value={field.value}
                      loadBalancing={form.watch('load_balancing')}
                      onChange={field.onChange}
                      onLoadBalancingChange={(value) =>
                        form.setValue('load_balancing', value)
                      }
                      labels={{
                        address: t('upstreamAddress'),
                        priority: t('upstreamPriority'),
                        loadBalancing: t('loadBalancing'),
                        roundRobin: t('roundRobin'),
                        leastConn: t('leastConn'),
                        add: t('addUpstream'),
                        remove: t('removeUpstream'),
                        hint: t('upstreamStructuredHint'),
                      }}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          ) : null}

          {upstreamType === 'tunnel' ? (
            <div className='space-y-4 rounded-lg border border-dashed bg-muted/30 p-4'>
              <FormField
                control={form.control}
                name='tunnel_id'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('selectTunnel')}</FormLabel>
                    <Select
                      value={field.value || 'none'}
                      onValueChange={(value) =>
                        field.onChange(value === 'none' ? '' : value)
                      }
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder={t('pleaseSelect')} />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value='none'>
                          {t('pleaseSelect')}
                        </SelectItem>
                        {tunnelClients.map((tunnel) => (
                          <SelectItem key={tunnel.id} value={String(tunnel.id)}>
                            {tunnel.name} (
                            {tunnel.status === 'online'
                              ? t('online')
                              : t('offline')}
                            )
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FormDescription>{t('tunnelForwardHint')}</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='tunnel_target_protocol'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('tunnelProtocol')}</FormLabel>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value='http'>HTTP</SelectItem>
                        <SelectItem value='https'>HTTPS</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='tunnel_target_addr'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('tunnelAddress')}</FormLabel>
                    <FormControl>
                      <Input placeholder='127.0.0.1:8080' {...field} />
                    </FormControl>
                    <FormDescription>{t('tunnelAddressHint')}</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          ) : null}

          {upstreamType === 'pages' ? (
            <div className='rounded-lg border border-dashed bg-muted/30 p-4'>
              <FormField
                control={form.control}
                name='pages_project_id'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('selectPagesProject')}</FormLabel>
                    <Select
                      value={field.value || 'none'}
                      onValueChange={(value) =>
                        field.onChange(value === 'none' ? '' : value)
                      }
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder={t('pleaseSelect')} />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value='none'>
                          {t('pleaseSelect')}
                        </SelectItem>
                        {pagesProjects.map((project) => (
                          <SelectItem
                            key={project.id}
                            value={String(project.id)}
                          >
                            {project.name} ({project.slug})
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FormDescription>{t('pagesProjectHint')}</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          ) : null}

          <FormField
            control={form.control}
            name='origin_host'
            render={({ field }) => (
              <FormItem>
                <FormLabel>Origin Host Header</FormLabel>
                <FormControl>
                  <Input placeholder='origin.example.internal' {...field} />
                </FormControl>
                <FormDescription>{t('originHostHint')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <div className='space-y-4 rounded-lg border border-dashed bg-muted/30 p-4'>
            <div>
              <h3 className='text-sm font-medium'>{t('proxyConfig')}</h3>
              <p className='mt-1 text-xs text-muted-foreground'>
                {t('proxyConfigHint')}
              </p>
            </div>
            <div className='grid grid-cols-1 gap-4 sm:grid-cols-2'>
              {(
                [
                  ['proxy_connect_timeout', 'proxyConnectTimeout'],
                  ['proxy_send_timeout', 'proxySendTimeout'],
                  ['proxy_read_timeout', 'proxyReadTimeout'],
                  ['client_header_timeout', 'clientHeaderTimeout'],
                  ['client_body_timeout', 'clientBodyTimeout'],
                  ['send_timeout', 'sendTimeout'],
                ] as const
              ).map(([name, label]) => (
                <FormField
                  key={name}
                  control={form.control}
                  name={name}
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t(label)}</FormLabel>
                      <FormControl>
                        <Input
                          placeholder={t('timeoutPlaceholder')}
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              ))}
              <FormField
                control={form.control}
                name='client_max_body_size'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('clientMaxBodySize')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder={t('bodySizePlaceholder')}
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            {(
              [
                ['websocket_enabled', 'websocketEnabled'],
                ['proxy_request_buffering', 'proxyRequestBuffering'],
                ['proxy_buffering_enabled', 'proxyBufferingEnabled'],
              ] as const
            ).map(([name, label]) => (
              <FormField
                key={name}
                control={form.control}
                name={name}
                render={({ field }) => (
                  <FormItem className='flex items-center justify-between gap-3'>
                    <FormLabel>{t(label)}</FormLabel>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger className='w-36'>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value='inherit'>
                          {t('inheritGlobal')}
                        </SelectItem>
                        <SelectItem value='on'>{t('enabled')}</SelectItem>
                        <SelectItem value='off'>{t('disabled')}</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
            ))}
          </div>

          <FormField
            control={form.control}
            name='custom_headers_text'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('customHeaders')}</FormLabel>
                <FormControl>
                  <Textarea
                    className='min-h-32 font-mono text-xs'
                    placeholder={'X-Trace-Id: $request_id\nX-Site: marketing'}
                    {...field}
                  />
                </FormControl>
                <FormDescription>{t('customHeadersHint')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </form>
      </Form>
    </SectionShell>
  );
}
