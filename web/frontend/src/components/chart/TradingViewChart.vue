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
let resizeFrame = 0;
let lastSize = { width: 0, height: 0 };
let lastWindowSize = readWindowSize();
const fallbackSize = { width: 1, height: 360 };
const maxRenderedChartHeight = 1200;

onMounted(() => {
  if (!rootRef.value || !containerRef.value) return;

  const initialSize = readHostSize() ?? { ...fallbackSize, fixedViewport: false };
  lastSize = { width: initialSize.width, height: initialSize.height };
  lastWindowSize = readWindowSize();
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
  if (entries.some((entry) => entry.target === observedResizeHost)) {
    scheduleResize();
  }
}

function scheduleResize() {
  if (resizeFrame > 0) return;
  resizeFrame = window.requestAnimationFrame(resizeChart);
}

function resizeChart() {
  resizeFrame = 0;
  if (!chart) return;

  const nextMeasurement = readHostSize();
  if (!nextMeasurement) return;

  const nextWindowSize = readWindowSize();
  const nextSize = guardFixedViewportSize(nextMeasurement, nextWindowSize);
  if (nextSize.width === lastSize.width && nextSize.height === lastSize.height) return;

  lastSize = nextSize;
  lastWindowSize = nextWindowSize;
  chart.resize(nextSize.width, nextSize.height);
}

function readHostSize() {
  const host = readResizeHost();
  if (!host) return null;

  const bounds = host.getBoundingClientRect();
  const fixedViewport = isFixedViewportHost(host);
  const measuredWidth =
    readClientWidth(host) ?? readPixelSize(host, "width") ?? positiveFloor(bounds.width);
  const measuredHeight = fixedViewport
    ? readFixedViewportHeight(host, bounds)
    : readClientHeight(host) ?? readPixelSize(host, "height") ?? positiveFloor(bounds.height);
  const width = measuredWidth ?? fallbackSize.width;
  const height = measuredHeight ? clampRenderedHeight(measuredHeight) : fallbackSize.height;
  if (width <= 0 || height <= 0) return null;

  return { width, height, fixedViewport };
}

function readFixedViewportHeight(element: HTMLElement, bounds: DOMRect) {
  const declaredHeight = readPixelSize(element, "height");
  if (declaredHeight) return declaredHeight;

  const declaredMaxHeight = readPixelSize(element, "maxHeight");
  if (declaredMaxHeight) return declaredMaxHeight;

  const boundedHeight = positiveFloor(bounds.height);
  if (boundedHeight && boundedHeight <= fixedViewportHeightCap()) return boundedHeight;

  return fallbackSize.height;
}

function guardFixedViewportSize(
  nextSize: { width: number; height: number; fixedViewport: boolean },
  nextWindowSize: { width: number; height: number },
) {
  const windowUnchanged =
    nextWindowSize.width === lastWindowSize.width && nextWindowSize.height === lastWindowSize.height;
  if (
    nextSize.fixedViewport &&
    windowUnchanged &&
    nextSize.height !== lastSize.height
  ) {
    return { width: nextSize.width, height: lastSize.height };
  }

  return { width: nextSize.width, height: nextSize.height };
}

function clampRenderedHeight(height: number) {
  return Math.min(Math.floor(height), fixedViewportHeightCap());
}

function fixedViewportHeightCap() {
  const viewportHeight = window.innerHeight > 0 ? window.innerHeight : fallbackSize.height;
  return Math.min(maxRenderedChartHeight, Math.max(fallbackSize.height, viewportHeight));
}

function readClientWidth(element: HTMLElement) {
  return element.clientWidth > 0 ? element.clientWidth : null;
}

function readClientHeight(element: HTMLElement) {
  return element.clientHeight > 0 ? element.clientHeight : null;
}

function readPixelSize(element: HTMLElement, property: "width" | "height" | "maxHeight") {
  const value = Number.parseFloat(window.getComputedStyle(element)[property]);
  if (!Number.isFinite(value) || value <= 0) return null;
  return value;
}

function readResizeHost() {
  const root = rootRef.value;
  if (!root) return null;
  const fixedChartSlot = root.closest<HTMLElement>('[data-chart-viewport="fixed"]');
  if (fixedChartSlot && fixedChartSlot !== root) return fixedChartSlot;
  const parent = root.parentElement;
  if (!parent) return root;
  if (parent.hasAttribute("data-v-app") && parent.parentElement) return parent.parentElement;
  return parent;
}

function isFixedViewportHost(element: HTMLElement) {
  return element.getAttribute("data-chart-viewport") === "fixed";
}

function readWindowSize() {
  return {
    width: window.innerWidth > 0 ? window.innerWidth : 0,
    height: window.innerHeight > 0 ? window.innerHeight : 0,
  };
}

function positiveFloor(value: number) {
  const floored = Math.floor(value);
  return floored > 0 ? floored : null;
}
</script>
