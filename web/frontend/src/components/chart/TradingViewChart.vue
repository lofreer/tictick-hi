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
let resizeFrame = 0;
let lastSize = { width: 0, height: 0 };
const fallbackSize = { width: 1, height: 360 };
const maxRenderedChartHeight = 1200;

onMounted(() => {
  if (!rootRef.value || !containerRef.value) return;

  const initialSize = readHostSize() ?? fallbackSize;
  lastSize = initialSize;
  applyContainerSize(initialSize);
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

  const measurementHost = readResizeHost();
  if (measurementHost) {
    resizeObserver = new ResizeObserver(scheduleResize);
    resizeObserver.observe(measurementHost);
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
  applyContainerSize(nextSize);
  chart.resize(nextSize.width, nextSize.height);
}

function readHostSize() {
  if (!rootRef.value) return null;

  const host = readLayoutHost();
  if (!host) return null;

  const bounds = host.getBoundingClientRect();
  const width = Math.floor(readClientWidth(host) ?? bounds.width);
  const height = readStableHostHeight(host, bounds);
  if (width <= 0 || height <= 0) return null;

  return { width, height };
}

function readStableHostHeight(host: HTMLElement, hostBounds: DOMRect) {
  const measuredHeight = readClientHeight(host) ?? Math.floor(hostBounds.height);
  const panel = host.closest<HTMLElement>(".chart-panel");
  if (!panel) return clampRenderedHeight(measuredHeight);

  const panelBounds = panel.getBoundingClientRect();
  const panelHeight = readClientHeight(panel) ?? readPixelHeight(panel) ?? Math.floor(panelBounds.height);
  const offsetTop = host === panel ? 0 : Math.max(0, Math.floor(hostBounds.top - panelBounds.top));
  const availableHeight = Math.floor(panelHeight - offsetTop);
  if (availableHeight <= 0) return clampRenderedHeight(measuredHeight);

  return clampRenderedHeight(Math.min(measuredHeight, availableHeight));
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

function readResizeHost() {
  return readLayoutHost();
}

function readLayoutHost() {
  const root = rootRef.value;
  if (!root) return null;
  const parent = root.parentElement;
  return parent?.closest<HTMLElement>(".research-chart-body, .chart-panel") ?? parent ?? root;
}

function applyContainerSize(size: { width: number; height: number }) {
  if (rootRef.value) {
    rootRef.value.style.width = `${size.width}px`;
    rootRef.value.style.height = `${size.height}px`;
  }
  if (containerRef.value) {
    containerRef.value.style.width = `${size.width}px`;
    containerRef.value.style.height = `${size.height}px`;
  }
}
</script>
