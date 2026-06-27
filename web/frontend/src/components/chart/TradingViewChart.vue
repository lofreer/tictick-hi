<template>
  <div ref="rootRef" class="trading-chart">
    <div ref="containerRef" class="trading-chart__canvas" />
    <div v-if="data.length === 0" class="trading-chart__empty">
      <EmptyState :title="emptyTitle" />
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  CandlestickSeries,
  createChart,
  createSeriesMarkers,
  type CandlestickData,
  type IChartApi,
  type ISeriesApi,
  type ISeriesMarkersPluginApi,
  type SeriesMarker,
  type Time,
} from "lightweight-charts";
import { onBeforeUnmount, onMounted, ref, watch } from "vue";

import EmptyState from "@/components/common/EmptyState.vue";
import { useThemeStore } from "@/stores/theme";
import { appColors, chartTheme } from "@/theme/tokens";
import type { ChartCandle, ChartMarker } from "@/types/app";
import "./TradingViewChart.css";

const props = defineProps<{
  data: ChartCandle[];
  emptyTitle: string;
  markers?: ChartMarker[];
}>();

const themeStore = useThemeStore();
const rootRef = ref<HTMLDivElement | null>(null);
const containerRef = ref<HTMLDivElement | null>(null);
let chart: IChartApi | null = null;
let series: ISeriesApi<"Candlestick"> | null = null;
let markerPlugin: ISeriesMarkersPluginApi<Time> | null = null;
let resizeObserver: ResizeObserver | null = null;
let observedResizeHost: HTMLElement | null = null;
let observedResizeHostSize: { width: number; height: number } | null = null;
let resizeFrame = 0;
let lastSize = { width: 0, height: 0 };
const fallbackSize = { width: 1, height: 360 };
const maxRenderedChartHeight = 1200;

onMounted(() => {
  if (!rootRef.value || !containerRef.value) return;

  const initialSize = readHostSize() ?? fallbackSize;
  lastSize = initialSize;
  chart = createChart(containerRef.value, {
    ...chartTheme(themeStore.mode),
    width: initialSize.width,
    height: initialSize.height,
    localization: {
      priceFormatter: (price: number) => price.toFixed(2),
    },
  });
  series = chart.addSeries(CandlestickSeries, {
    upColor: appColors.success,
    downColor: appColors.danger,
    borderVisible: false,
    wickUpColor: appColors.success,
    wickDownColor: appColors.danger,
  });
  markerPlugin = createSeriesMarkers(series, []);

  observedResizeHost = readResizeHost();
  if (observedResizeHost) {
    resizeObserver = new ResizeObserver(handleObservedResize);
    resizeObserver.observe(observedResizeHost);
  }
  window.addEventListener("resize", scheduleResize);

  syncData();
  scheduleResize();
});

onBeforeUnmount(() => {
  if (resizeFrame > 0) {
    window.cancelAnimationFrame(resizeFrame);
    resizeFrame = 0;
  }
  resizeObserver?.disconnect();
  resizeObserver = null;
  observedResizeHost = null;
  observedResizeHostSize = null;
  window.removeEventListener("resize", scheduleResize);
  chart?.remove();
  chart = null;
  series = null;
  markerPlugin = null;
});

watch(
  () => props.data,
  () => syncData(),
  { deep: true },
);

watch(
  () => props.markers,
  () => syncMarkers(),
  { deep: true },
);

watch(
  () => themeStore.mode,
  (mode) => chart?.applyOptions(chartTheme(mode)),
);

function syncData() {
  const candleData: CandlestickData[] = props.data.map((item) => ({
    time: item.time as Time,
    open: item.open,
    high: item.high,
    low: item.low,
    close: item.close,
  }));

  series?.setData(candleData);
  syncMarkers();
  chart?.timeScale().fitContent();
}

function syncMarkers() {
  const markerData: SeriesMarker<Time>[] = (props.markers ?? []).map((marker) => ({
    ...marker,
    time: marker.time as Time,
  }));
  markerPlugin?.setMarkers(markerData);
}

function handleObservedResize(entries: ResizeObserverEntry[]) {
  for (const entry of entries) {
    if (entry.target === observedResizeHost) {
      observedResizeHostSize = readObserverContentSize(entry);
      break;
    }
  }
  scheduleResize();
}

function scheduleResize() {
  if (resizeFrame > 0) return;
  resizeFrame = window.requestAnimationFrame(resizeChart);
}

function resizeChart() {
  resizeFrame = 0;
  if (!chart) return;

  const nextSize = readHostSize();
  if (!nextSize) return;
  if (nextSize.width === lastSize.width && nextSize.height === lastSize.height) return;

  lastSize = nextSize;
  chart.resize(nextSize.width, nextSize.height);
}

function readHostSize() {
  const host = readResizeHost();
  if (!host) return null;

  const bounds = host.getBoundingClientRect();
  const measuredWidth = readClientWidth(host) ?? readObservedWidth(host) ?? positiveFloor(bounds.width);
  const measuredHeight =
    readClientHeight(host) ?? readObservedHeight(host) ?? readPixelHeight(host) ?? positiveFloor(bounds.height);
  const width = measuredWidth ?? fallbackSize.width;
  const height = measuredHeight ? clampRenderedHeight(measuredHeight) : fallbackSize.height;
  if (width <= 0 || height <= 0) return null;

  return { width, height };
}

function clampRenderedHeight(height: number) {
  const viewportHeight = window.innerHeight > 0 ? window.innerHeight : fallbackSize.height;
  const safeMaxHeight = Math.min(maxRenderedChartHeight, Math.max(fallbackSize.height, viewportHeight));
  return Math.min(Math.floor(height), safeMaxHeight);
}

function readClientWidth(element: HTMLElement) {
  return element.clientWidth > 0 ? element.clientWidth : null;
}

function readClientHeight(element: HTMLElement) {
  return element.clientHeight > 0 ? element.clientHeight : null;
}

function readPixelHeight(element: HTMLElement) {
  const value = Number.parseFloat(window.getComputedStyle(element).height);
  if (!Number.isFinite(value) || value <= 0) return null;
  return value;
}

function readObservedWidth(element: HTMLElement) {
  if (element !== observedResizeHost || !observedResizeHostSize) return null;
  return observedResizeHostSize.width > 0 ? observedResizeHostSize.width : null;
}

function readObservedHeight(element: HTMLElement) {
  if (element !== observedResizeHost || !observedResizeHostSize) return null;
  return observedResizeHostSize.height > 0 ? observedResizeHostSize.height : null;
}

function readObserverContentSize(entry: ResizeObserverEntry) {
  const box = Array.isArray(entry.contentBoxSize) ? entry.contentBoxSize[0] : entry.contentBoxSize;
  const width = box?.inlineSize ?? entry.contentRect.width;
  const height = box?.blockSize ?? entry.contentRect.height;
  return {
    width: Math.floor(width),
    height: Math.floor(height),
  };
}

function readResizeHost() {
  return containerRef.value;
}

function positiveFloor(value: number) {
  const floored = Math.floor(value);
  return floored > 0 ? floored : null;
}
</script>
