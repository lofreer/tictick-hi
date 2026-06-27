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

onMounted(() => {
  if (!rootRef.value || !containerRef.value) return;

  const initialSize = readHostSize() ?? { width: 1, height: 360 };
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

  resizeObserver = new ResizeObserver(scheduleResize);
  resizeObserver.observe(rootRef.value);
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
  chart.resize(nextSize.width, nextSize.height);
}

function readHostSize() {
  if (!rootRef.value) return null;

  const bounds = rootRef.value.getBoundingClientRect();
  const width = Math.floor(bounds.width);
  const height = Math.floor(bounds.height);
  if (width <= 0 || height <= 0) return null;

  return { width, height };
}
</script>
