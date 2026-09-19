'use client';

import ReactEChartsCore from 'echarts-for-react/lib/core';
import type { EChartsCoreOption } from 'echarts/core';
import * as echarts from 'echarts/core';
import { MapChart, ScatterChart } from 'echarts/charts';
import { GeoComponent, TooltipComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import { useTheme } from 'next-themes';
import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useEffect, useMemo, useRef, useState } from 'react';

import { EmptyState } from '@/components/layout/empty';
import type {
  DashboardNodeHealth,
  DistributionItem,
} from '@/lib/services/openflare';

import {
  getNodeStatusLabel,
  getOpenrestyStatusLabel,
  type NodeMessageT,
} from '../../nodes/components/node-utils';
import countryCentroidsJson from './data/country-centroids.json';
import worldGeoJson from './data/world-geo.json';

echarts.use([
  GeoComponent,
  TooltipComponent,
  MapChart,
  ScatterChart,
  CanvasRenderer,
]);

const fallbackCoordinates = [
  [-122.4194, 37.7749],
  [-46.6333, -23.5505],
  [-0.1276, 51.5072],
  [2.3522, 48.8566],
  [77.209, 28.6139],
  [121.4737, 31.2304],
  [103.8198, 1.3521],
  [151.2093, -33.8688],
  [28.0473, -26.2041],
  [139.6917, 35.6895],
] as const;

type Tone = 'healthy' | 'warning' | 'danger';

type WorldGeoJson = {
  type: string;
  features?: WorldFeature[];
};

type WorldFeature = {
  properties?: {
    name?: string;
  };
  geometry?: {
    type?: 'Polygon' | 'MultiPolygon';
    coordinates?: number[][][] | number[][][][];
  };
};

type MapNodeDatum = {
  name: string;
  geoName: string;
  route: string;
  derivedFromGeo: boolean;
  nodes: DashboardNodeHealth[];
  requestCount: number;
  errorCount: number;
  activeEventCount: number;
  value: [number, number];
  itemStyle: {
    color: string;
    borderColor: string;
    borderWidth: number;
  };
  emphasis: {
    itemStyle: {
      borderColor: string;
      borderWidth: number;
    };
  };
};

type CountryRegionDatum = {
  name: string;
  value: number;
  itemStyle: {
    areaColor: string;
    borderColor: string;
  };
  emphasis: {
    itemStyle: {
      areaColor: string;
      borderColor: string;
    };
  };
};

let worldMapRegistrationAttempted = false;
let worldMapRegistrationSucceeded = false;

const baseWorldMapLayoutSizePercent = 118;
const baseWorldMapZoom = 1;
const worldMapAspectRatio = 5 / 3;

export function buildNodeMapHref(nodes: DashboardNodeHealth[]) {
  if (nodes.length !== 1) {
    return '/nodes';
  }
  return `/nodes/detail?id=${nodes[0].id}`;
}

export function getNodeTone(node: DashboardNodeHealth): Tone {
  if (
    node.status === 'offline' ||
    node.openresty_status === 'unhealthy' ||
    node.active_event_count > 0
  ) {
    return 'danger';
  }

  if (
    node.cpu_usage_percent >= 80 ||
    node.memory_usage_percent >= 85 ||
    node.storage_usage_percent >= 85
  ) {
    return 'warning';
  }

  return 'healthy';
}

const countryCentroids = countryCentroidsJson as Record<string, number[]>;

type CountryCentroid = [number, number];

function toCountryCentroid(
  coords: number[] | undefined,
): CountryCentroid | null {
  if (!coords || coords.length < 2) {
    return null;
  }
  return [coords[0], coords[1]];
}

function resolveCountryCentroid(geoName: string): CountryCentroid | null {
  const trimmed = geoName.trim();
  if (!trimmed) {
    return null;
  }
  const candidates = new Set<string>([trimmed]);
  // ipinfo-style names: "Hong Kong, Hong Kong, HK"
  for (const part of trimmed.split(',').map((item) => item.trim())) {
    if (part) {
      candidates.add(part);
    }
  }
  // trailing ISO2 often present after city/region
  const isoMatch = trimmed.match(/\b([A-Z]{2})\b/g);
  if (isoMatch) {
    for (const code of isoMatch) {
      candidates.add(code);
    }
  }

  for (const candidate of candidates) {
    const direct = toCountryCentroid(countryCentroids[candidate]);
    if (direct) {
      return direct;
    }
    const lower = candidate.toLowerCase();
    for (const [name, coordinates] of Object.entries(countryCentroids)) {
      if (name.toLowerCase() === lower) {
        return toCountryCentroid(coordinates);
      }
    }
  }

  // Soft contains match for longer composite labels (prefer longer country names).
  const lowerFull = trimmed.toLowerCase();
  let best: { length: number; coords: CountryCentroid } | null = null;
  for (const [name, coordinates] of Object.entries(countryCentroids)) {
    const lowerName = name.toLowerCase();
    if (lowerFull.includes(lowerName) && lowerName.length >= 4) {
      const coords = toCountryCentroid(coordinates);
      if (!coords) {
        continue;
      }
      if (!best || lowerName.length > best.length) {
        best = { length: lowerName.length, coords };
      }
    }
  }
  return best?.coords ?? null;
}

function getNodeCoordinates(node: DashboardNodeHealth, index: number) {
  if (
    typeof node.geo_latitude === 'number' &&
    typeof node.geo_longitude === 'number'
  ) {
    return {
      coordinates: [node.geo_longitude, node.geo_latitude] as [number, number],
      derivedFromGeo: true,
    };
  }

  const countryCentroid = resolveCountryCentroid(node.geo_name || '');
  if (countryCentroid) {
    return {
      coordinates: countryCentroid,
      derivedFromGeo: true,
    };
  }

  return {
    coordinates: [
      ...fallbackCoordinates[index % fallbackCoordinates.length],
    ] as [number, number],
    derivedFromGeo: false,
  };
}

export type NodeMapGroup = {
  coordinates: [number, number];
  derivedFromGeo: boolean;
  geoName: string;
  nodes: DashboardNodeHealth[];
  tone: Tone;
};

const tonePriority: Record<Tone, number> = {
  healthy: 0,
  warning: 1,
  danger: 2,
};

function coordinateKey(coordinates: [number, number]) {
  return `${coordinates[0].toFixed(6)},${coordinates[1].toFixed(6)}`;
}

function geoNameKey(geoName: string) {
  return geoName.trim().toLocaleLowerCase();
}

export function groupDashboardNodes(nodes: DashboardNodeHealth[]) {
  return nodes.reduce<NodeMapGroup[]>((groups, node, index) => {
    const { coordinates, derivedFromGeo } = getNodeCoordinates(node, index);
    const locationKey = coordinateKey(coordinates);
    const nameKey = geoNameKey(node.geo_name || '');
    const group = groups.find(
      (item) =>
        coordinateKey(item.coordinates) === locationKey ||
        (nameKey && geoNameKey(item.geoName) === nameKey),
    );
    const tone = getNodeTone(node);

    if (!group) {
      groups.push({
        coordinates,
        derivedFromGeo,
        geoName: node.geo_name || node.name,
        nodes: [node],
        tone,
      });
      return groups;
    }

    group.nodes.push(node);
    group.derivedFromGeo = group.derivedFromGeo || derivedFromGeo;
    if (tonePriority[tone] > tonePriority[group.tone]) {
      group.tone = tone;
    }
    return groups;
  }, []);
}

function escapeTooltipText(value: string) {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

export function buildNodeMapTooltip(
  data: Pick<
    MapNodeDatum,
    | 'activeEventCount'
    | 'derivedFromGeo'
    | 'errorCount'
    | 'geoName'
    | 'nodes'
    | 'requestCount'
  >,
  t: NodeMessageT,
  tn: NodeMessageT,
) {
  const locationLine = escapeTooltipText(
    data.derivedFromGeo ? data.geoName : `${data.geoName} · ${t('presetPin')}`,
  );
  const nodeLines = data.nodes
    .map(
      (node) =>
        `<div>${escapeTooltipText(node.name)} · ${escapeTooltipText(getNodeStatusLabel(node.status, tn))} · ${escapeTooltipText(getOpenrestyStatusLabel(node.openresty_status, tn))}</div>`,
    )
    .join('');

  return [
    `<div style="font-weight:600;margin-bottom:6px;">${locationLine}</div>`,
    `<div>${t('nodeCount', { count: data.nodes.length })}</div>`,
    nodeLines,
    `<div>${t('requestsErrors', {
      requests: data.requestCount.toLocaleString('zh-CN'),
      errors: data.errorCount.toLocaleString('zh-CN'),
    })}</div>`,
    `<div>${t('eventsStatus', {
      events: data.activeEventCount,
    })}</div>`,
  ].join('');
}

function ensureWorldMapRegistered() {
  if (worldMapRegistrationSucceeded || echarts.getMap('world')) {
    worldMapRegistrationSucceeded = true;
    return true;
  }

  if (worldMapRegistrationAttempted) {
    return false;
  }

  worldMapRegistrationAttempted = true;

  try {
    const geoJson = worldGeoJson as WorldGeoJson;

    if (
      geoJson.type !== 'FeatureCollection' ||
      !Array.isArray(geoJson.features)
    ) {
      throw new Error('invalid world geojson payload');
    }

    echarts.registerMap(
      'world',
      geoJson as unknown as Parameters<typeof echarts.registerMap>[1],
    );

    if (!echarts.getMap('world')) {
      throw new Error('world map registration failed');
    }

    worldMapRegistrationSucceeded = true;
    return true;
  } catch (error) {
    const registrationError =
      error instanceof Error
        ? error
        : new Error('unknown world map registration error');
    console.error('Failed to register ECharts world map', registrationError);
    return false;
  }
}

export function WorldStageMap({
  nodes,
  sourceCountries,
}: {
  nodes: DashboardNodeHealth[];
  sourceCountries: DistributionItem[];
}) {
  const t = useTranslations('dashboard.map');
  const tn = useTranslations('nodes');
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === 'dark';
  const router = useRouter();
  const chartContainerRef = useRef<HTMLDivElement | null>(null);
  const [mapReady, setMapReady] = useState(false);
  const [mapFailed, setMapFailed] = useState(false);
  const [containerSize, setContainerSize] = useState({ width: 0, height: 0 });

  useEffect(() => {
    const ready = ensureWorldMapRegistered();
    setMapReady(ready);
    setMapFailed(!ready);
  }, []);

  useEffect(() => {
    if (!mapReady) {
      return;
    }

    const container = chartContainerRef.current;
    if (!container) {
      return;
    }

    const updateSize = () => {
      const nextWidth = container.clientWidth;
      const nextHeight = container.clientHeight;
      if (nextWidth <= 0 || nextHeight <= 0) {
        return;
      }
      setContainerSize((previous) =>
        previous.width === nextWidth && previous.height === nextHeight
          ? previous
          : { width: nextWidth, height: nextHeight },
      );
    };

    updateSize();
    const frameId = window.requestAnimationFrame(updateSize);

    if (typeof ResizeObserver === 'undefined') {
      window.addEventListener('resize', updateSize);
      return () => {
        window.cancelAnimationFrame(frameId);
        window.removeEventListener('resize', updateSize);
      };
    }

    const observer = new ResizeObserver(() => {
      updateSize();
    });
    observer.observe(container);

    return () => {
      window.cancelAnimationFrame(frameId);
      observer.disconnect();
    };
  }, [mapReady]);

  const mapPalette = useMemo(
    () =>
      isDark
        ? {
            areaColor: '#1e293b',
            borderColor: 'rgba(148,163,184,0.32)',
            labelColor: '#e2e8f0',
            healthyColor: '#34d399',
            warningColor: '#fbbf24',
            dangerColor: '#f87171',
            healthyBorder: '#6ee7b7',
            warningBorder: '#fcd34d',
            dangerBorder: '#fca5a5',
          }
        : {
            areaColor: '#e2e8f0',
            borderColor: 'rgba(100,116,139,0.45)',
            labelColor: '#18181b',
            healthyColor: '#10b981',
            warningColor: '#f59e0b',
            dangerColor: '#ef4444',
            healthyBorder: '#6ee7b7',
            warningBorder: '#fcd34d',
            dangerBorder: '#fca5a5',
          },
    [isDark],
  );

  const mapNodes = useMemo<MapNodeDatum[]>(
    () =>
      groupDashboardNodes(nodes).map((group) => {
        const { coordinates, derivedFromGeo, tone } = group;
        const toneColor =
          tone === 'healthy'
            ? mapPalette.healthyColor
            : tone === 'warning'
              ? mapPalette.warningColor
              : mapPalette.dangerColor;
        const toneBorder =
          tone === 'healthy'
            ? mapPalette.healthyBorder
            : tone === 'warning'
              ? mapPalette.warningBorder
              : mapPalette.dangerBorder;

        return {
          name: group.nodes.map((node) => node.name).join(', '),
          geoName: group.geoName,
          route: buildNodeMapHref(group.nodes),
          derivedFromGeo,
          nodes: group.nodes,
          requestCount: group.nodes.reduce(
            (total, node) => total + node.request_count,
            0,
          ),
          errorCount: group.nodes.reduce(
            (total, node) => total + node.error_count,
            0,
          ),
          activeEventCount: group.nodes.reduce(
            (total, node) => total + node.active_event_count,
            0,
          ),
          value: [coordinates[0], coordinates[1]],
          itemStyle: {
            color: toneColor,
            borderColor: toneBorder,
            borderWidth: 1.5,
          },
          emphasis: {
            itemStyle: {
              borderColor: toneBorder,
              borderWidth: 2,
            },
          },
        };
      }),
    [mapPalette, nodes],
  );

  const countryRegions = useMemo<CountryRegionDatum[]>(() => {
    const maxRequests = Math.max(
      1,
      ...sourceCountries.map((item) => item.value || 0),
    );

    return sourceCountries
      .filter((item) => item.key && item.value > 0)
      .map((item) => {
        const intensity = Math.max(0.18, item.value / maxRequests);
        const areaOpacity = Number((0.14 + intensity * 0.58).toFixed(3));
        const borderOpacity = Number((0.22 + intensity * 0.48).toFixed(3));
        const areaColor = isDark
          ? `rgba(96, 165, 250, ${areaOpacity})`
          : `rgba(59, 130, 246, ${areaOpacity})`;
        const borderColor = isDark
          ? `rgba(147, 197, 253, ${borderOpacity})`
          : `rgba(59, 130, 246, ${borderOpacity})`;

        return {
          name: item.key,
          value: item.value,
          itemStyle: {
            areaColor,
            borderColor,
          },
          emphasis: {
            itemStyle: {
              areaColor,
              borderColor,
            },
          },
        };
      });
  }, [isDark, sourceCountries]);

  const responsiveMapScale = useMemo(() => {
    const { width, height } = containerSize;
    if (width <= 0 || height <= 0) {
      return 1;
    }

    const expectedWidth = height * worldMapAspectRatio;
    const fitWidth = Math.min(width, expectedWidth);
    const widthScale = Math.min(Math.max(fitWidth / 180, 0.72), 1.08);
    const heightScale = Math.min(Math.max(height / 252, 0.72), 1.08);

    return Number(Math.min(widthScale, heightScale).toFixed(3));
  }, [containerSize]);

  const computedLayoutSize = useMemo(
    () => `${Math.round(baseWorldMapLayoutSizePercent * responsiveMapScale)}%`,
    [responsiveMapScale],
  );
  const computedZoom = useMemo(
    () => Number((baseWorldMapZoom * responsiveMapScale).toFixed(3)),
    [responsiveMapScale],
  );

  const mapOption = useMemo<EChartsCoreOption>(
    () => ({
      animation: false,
      backgroundColor: 'transparent',
      tooltip: {
        trigger: 'item',
        transitionDuration: 0,
        backgroundColor: isDark
          ? 'rgba(24,24,27,0.96)'
          : 'rgba(255,255,255,0.98)',
        borderColor: isDark ? 'rgba(63,63,70,0.8)' : 'rgba(228,228,231,0.9)',
        borderWidth: 1,
        textStyle: {
          color: isDark ? '#fafafa' : '#18181b',
          fontSize: 12,
        },
        formatter: (params: unknown) => {
          const payload = params as {
            data?: MapNodeDatum | CountryRegionDatum;
            seriesType?: string;
            name?: string;
          };
          const data = payload.data;
          if (
            payload.seriesType !== 'scatter' &&
            data &&
            'value' in data &&
            typeof data.value === 'number'
          ) {
            return [
              `<div style="font-weight:600;margin-bottom:6px;">${payload.name ?? data.name}</div>`,
              `<div>${t('sourceRequests', { count: data.value.toLocaleString('zh-CN') })}</div>`,
            ].join('');
          }

          if (!data || !('requestCount' in data)) {
            return '';
          }

          return buildNodeMapTooltip(data, t, tn);
        },
      },
      geo: {
        map: 'world',
        roam: true,
        silent: true,
        layoutCenter: ['50%', '50%'],
        layoutSize: computedLayoutSize,
        zoom: computedZoom,
        scaleLimit: {
          min: Math.max(Number((computedZoom * 0.78).toFixed(3)), 0.35),
          max: Math.max(Number((computedZoom * 2.2).toFixed(3)), 1.4),
        },
        regions: countryRegions,
        itemStyle: {
          areaColor: mapPalette.areaColor,
          borderColor: mapPalette.borderColor,
          borderWidth: 0.7,
        },
        emphasis: {
          disabled: true,
        },
      },
      series: [
        {
          type: 'map',
          map: 'world',
          geoIndex: 0,
          data: countryRegions,
          silent: true,
          z: 1,
          emphasis: {
            disabled: true,
          },
        },
        {
          type: 'scatter',
          coordinateSystem: 'geo',
          data: mapNodes,
          z: 3,
          symbolSize: 11,
          label: {
            show: false,
            color: mapPalette.labelColor,
            fontSize: 11,
          },
          emphasis: {
            scale: 1.12,
            label: {
              show: mapNodes.length > 0 && mapNodes.length <= 5,
              position: 'right',
              distance: 8,
              formatter: '{b}',
              backgroundColor: isDark
                ? 'rgba(24,24,27,0.88)'
                : 'rgba(255,255,255,0.94)',
              borderColor: isDark
                ? 'rgba(63,63,70,0.7)'
                : 'rgba(228,228,231,0.9)',
              borderWidth: 1,
              borderRadius: 999,
              padding: [4, 8],
            },
          },
        },
      ],
    }),
    [
      computedLayoutSize,
      computedZoom,
      countryRegions,
      isDark,
      mapNodes,
      mapPalette,
      t,
      tn,
    ],
  );

  if (!mapReady) {
    return (
      <div className='flex h-full items-center justify-center'>
        <EmptyState
          title={mapFailed ? t('loadFailed') : t('loading')}
          description={mapFailed ? t('loadFailedDesc') : t('loadingDesc')}
        />
      </div>
    );
  }

  const chartReady = containerSize.width > 0 && containerSize.height > 0;

  return (
    <div ref={chartContainerRef} className='h-full min-h-0 w-full'>
      {chartReady ? (
        <ReactEChartsCore
          echarts={echarts}
          option={mapOption}
          notMerge
          lazyUpdate
          opts={{ renderer: 'canvas' }}
          onEvents={{
            click: (params: { data?: MapNodeDatum }) => {
              if (params.data?.route) {
                router.push(params.data.route);
              }
            },
          }}
          style={{
            height: containerSize.height,
            width: containerSize.width,
          }}
        />
      ) : (
        <div className='flex h-full items-center justify-center'>
          <EmptyState
            title={t('loading')}
            description={t('measuring')}
            iconSize='sm'
          />
        </div>
      )}
    </div>
  );
}
