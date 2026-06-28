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
  HistogramSeries,
  TickMarkType,
  type CandlestickData,
  type HistogramData,
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
import { positiveFloor, readChartGutter, readClientHeight, readClientWidth, readPixelSize } from "./chartSizing";
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
let volumeSeries: ISeriesApi<"Histogram"> | null = null;
let markerPlugin: ISeriesMarkersPluginApi<Time> | null = null;
let resizeObserver: ResizeObserver | null = null;
let observedResizeHost: HTMLElement | null = null;
let resizeFrame = 0;
let lastSize = { width: 0, height: 0 };
let lastObservedHostWidth = 0;
let fixedViewportHeightSnapshot: number | null = null;
let pendingFixedViewportHeightRefresh = false;
const fallbackSize = { width: 1, height: 360 };
const maxRenderedChartHeight = 1200;
const maxInitialVisibleBars = 360;
const minInitialVisibleBars = 80;
const targetInitialBarSpacingPixels = 3;
const minTimeAxisEdgePaddingBars = 12;
const minTimeAxisEdgePaddingPixels = 48;
const maxTimeAxisEdgePaddingPixels = 96;
const timeAxisLabelInsetBars = 10.5;
const timeAxisEdgePaddingRatio = 0.12;
const volumePriceScaleId = "";
const volumeUpColor = "rgba(14, 203, 129, 0.28)";
const volumeDownColor = "rgba(246, 70, 93, 0.28)";

onMounted(() => {
  if (!rootRef.value || !containerRef.value) return;

  const initialSize = readHostSize() ?? { ...fallbackSize };
  lastSize = { width: initialSize.width, height: initialSize.height };
  lockRenderedViewport(lastSize);
  chart = createChart(containerRef.value, {
    ...responsiveChartOptions(),
    autoSize: false,
    width: initialSize.width,
    height: initialSize.height,
  });
  series = chart.addSeries(CandlestickSeries, {
    upColor: appColors.success,
    downColor: appColors.danger,
    borderVisible: false,
    wickUpColor: appColors.success,
    wickDownColor: appColors.danger,
  });
  volumeSeries = chart.addSeries(HistogramSeries, {
    priceFormat: { type: "volume" },
    priceLineVisible: false,
    lastValueVisible: false,
    priceScaleId: volumePriceScaleId,
    color: volumeUpColor,
    base: 0,
  });
  configurePriceScales();
  markerPlugin = createSeriesMarkers(series, []);

  observedResizeHost = readResizeHost();
  if (observedResizeHost) {
    lastObservedHostWidth = readHostWidth(observedResizeHost) ?? initialSize.width;
    resizeObserver = new ResizeObserver(handleObservedResize);
    resizeObserver.observe(observedResizeHost);
  }
  window.addEventListener("resize", handleWindowResize);

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
  fixedViewportHeightSnapshot = null;
  pendingFixedViewportHeightRefresh = false;
  lastObservedHostWidth = 0;
  window.removeEventListener("resize", handleWindowResize);
  chart?.remove();
  chart = null;
  series = null;
  volumeSeries = null;
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
  (mode) => {
    chart?.applyOptions(responsiveChartOptions(mode));
    configurePriceScales();
  },
);

function syncData() {
  const candleData: CandlestickData[] = props.data.map((item) => ({
    time: item.time as Time,
    open: item.open,
    high: item.high,
    low: item.low,
    close: item.close,
  }));
  const volumeData: HistogramData[] = props.data.map((item) => ({
    time: item.time as Time,
    value: item.volume,
    color: item.close >= item.open ? volumeUpColor : volumeDownColor,
  }));

  series?.setData(candleData);
  volumeSeries?.setData(volumeData);
  syncMarkers();
  fitChartContent(candleData.length);
}

function configurePriceScales() {
  chart?.priceScale("right").applyOptions({
    scaleMargins: {
      top: 0.08,
      bottom: 0.24,
    },
  });
  chart?.priceScale(volumePriceScaleId).applyOptions({
    scaleMargins: {
      top: 0.8,
      bottom: 0,
    },
  });
}

function syncMarkers() {
  const markerData: SeriesMarker<Time>[] = (props.markers ?? []).map((marker) => ({
    ...marker,
    time: marker.time as Time,
  }));
  markerPlugin?.setMarkers(markerData);
}

function fitChartContent(dataLength: number) {
  const timeScale = chart?.timeScale();
  if (!timeScale) return;
  timeScale.fitContent();
  if (dataLength === 0) return;
  const visibleBars = initialVisibleBars(dataLength);
  const edgePadding = timeAxisEdgePaddingBars(visibleBars);
  const totalPadding = edgePadding + timeAxisLabelInsetBars;
  const left = dataLength > visibleBars ? dataLength - visibleBars - totalPadding : -totalPadding;
  timeScale.setVisibleLogicalRange({
    from: left,
    to: dataLength - 1 + totalPadding,
  });
}

function initialVisibleBars(dataLength: number) {
  if (dataLength <= minInitialVisibleBars) return dataLength;
  const barsForWidth = Math.floor(chartPlotWidth() / targetInitialBarSpacingPixels);
  return Math.min(dataLength, Math.max(minInitialVisibleBars, Math.min(maxInitialVisibleBars, barsForWidth)));
}

function timeAxisEdgePaddingBars(visibleBars: number) {
  const plotWidth = chartPlotWidth();
  const paddingPixels = Math.min(
    maxTimeAxisEdgePaddingPixels,
    Math.max(minTimeAxisEdgePaddingPixels, plotWidth * timeAxisEdgePaddingRatio),
  );
  return Math.max(minTimeAxisEdgePaddingBars, Math.ceil((paddingPixels / plotWidth) * visibleBars));
}

function chartPlotWidth() {
  return Math.max(1, lastSize.width - rightPriceScaleMinimumWidth(lastSize.width));
}

function responsiveChartOptions(mode = themeStore.mode) {
  const theme = chartTheme(mode);
  return {
    ...theme,
    localization: {
      priceFormatter: formatChartPrice,
    },
    rightPriceScale: {
      ...theme.rightPriceScale,
      minimumWidth: rightPriceScaleMinimumWidth(lastSize.width),
    },
    timeScale: {
      ...theme.timeScale,
      secondsVisible: false,
      tickMarkMaxCharacterLength: 8,
      tickMarkFormatter: formatChartTickMark,
    },
  };
}

function rightPriceScaleMinimumWidth(width: number) {
  if (width < 520) return 104;
  if (width < 900) return 128;
  return 144;
}

function formatChartPrice(price: number) {
  if (!Number.isFinite(price)) return "";
  const absolutePrice = Math.abs(price);
  if (absolutePrice === 0) return "0";
  if (absolutePrice >= 1000) return price.toFixed(0);
  if (absolutePrice >= 100) return trimTrailingZeros(price.toFixed(1));
  if (absolutePrice >= 1) return trimTrailingZeros(price.toFixed(2));
  if (absolutePrice >= 0.01) return trimTrailingZeros(price.toFixed(4));
  return price.toPrecision(4);
}

function trimTrailingZeros(value: string) {
  return value.replace(/(\.\d*?[1-9])0+$/, "$1").replace(/\.0+$/, "");
}

function formatChartTickMark(time: Time, tickMarkType: TickMarkType) {
  const date = chartTimeToDate(time);
  if (!date) return null;
  if (tickMarkType === TickMarkType.Year) return `${date.getUTCFullYear()}`;
  if (tickMarkType === TickMarkType.Month) return `${date.getUTCFullYear().toString().slice(2)}-${pad2(date.getUTCMonth() + 1)}`;
  if (tickMarkType === TickMarkType.DayOfMonth) return `${pad2(date.getUTCMonth() + 1)}-${pad2(date.getUTCDate())}`;
  return `${pad2(date.getUTCHours())}:${pad2(date.getUTCMinutes())}`;
}

function chartTimeToDate(time: Time) {
  if (typeof time === "number") return new Date(time * 1000);
  if (typeof time === "string") {
    const parsed = new Date(time);
    return Number.isNaN(parsed.getTime()) ? null : parsed;
  }
  return new Date(Date.UTC(time.year, time.month - 1, time.day));
}

function pad2(value: number) {
  return value.toString().padStart(2, "0");
}

function handleObservedResize(entries: ResizeObserverEntry[]) {
  const host = observedResizeHost;
  const hostEntry = entries.find((entry) => entry.target === host);
  if (!host || !hostEntry) return;

  if (!isFixedViewportHost(host)) {
    scheduleResize();
    return;
  }

  const nextWidth = readResizeEntryWidth(hostEntry) ?? readHostWidth(host);
  if (!nextWidth || nextWidth === lastObservedHostWidth) return;

  lastObservedHostWidth = nextWidth;
  scheduleResize();
}

function handleWindowResize() {
  scheduleResize(true);
}

function scheduleResize(refreshFixedViewportHeight = false) {
  pendingFixedViewportHeightRefresh ||= refreshFixedViewportHeight;
  if (resizeFrame > 0) return;
  resizeFrame = window.requestAnimationFrame(resizeChart);
}

function resizeChart() {
  resizeFrame = 0;
  if (!chart) return;

  const nextMeasurement = readHostSize(pendingFixedViewportHeightRefresh);
  pendingFixedViewportHeightRefresh = false;
  if (!nextMeasurement) return;

  if (nextMeasurement.width === lastSize.width && nextMeasurement.height === lastSize.height) {
    lockRenderedViewport(lastSize);
    return;
  }

  lastSize = { width: nextMeasurement.width, height: nextMeasurement.height };
  lockRenderedViewport(lastSize);
  chart.applyOptions(responsiveChartOptions());
  chart.resize(nextMeasurement.width, nextMeasurement.height);
  fitChartContent(props.data.length);
}

function lockRenderedViewport(size: { width: number; height: number }) {
  const width = `${Math.max(1, Math.floor(size.width))}px`;
  const height = `${Math.max(1, Math.floor(size.height))}px`;
  for (const element of [rootRef.value, containerRef.value]) {
    if (!element) continue;
    element.style.setProperty("--tt-chart-render-width", width);
    element.style.setProperty("--tt-chart-render-height", height);
    element.style.setProperty("width", width, "important");
    element.style.setProperty("height", height, "important");
    element.style.setProperty("max-width", width, "important");
    element.style.setProperty("max-height", height, "important");
    element.style.setProperty("inline-size", width, "important");
    element.style.setProperty("block-size", height, "important");
    element.style.setProperty("max-inline-size", width, "important");
    element.style.setProperty("max-block-size", height, "important");
  }
}

function readHostSize(refreshFixedViewportHeight = false) {
  const host = readResizeHost();
  if (!host) return null;

  const bounds = host.getBoundingClientRect();
  const style = window.getComputedStyle(host);
  const fixedViewport = isFixedViewportHost(host);
  const measuredWidth = (readHostWidth(host, bounds) ?? fallbackSize.width) - readChartGutter(style, "--tt-chart-inline-end-gutter");
  const measuredHeight = fixedViewport
    ? readFixedViewportHeight(host, bounds, refreshFixedViewportHeight)
    : readClientHeight(host) ?? readPixelSize(host, "height") ?? positiveFloor(bounds.height);
  const width = Math.floor(measuredWidth);
  const height = measuredHeight
    ? clampRenderedHeight(measuredHeight - readChartGutter(style, "--tt-chart-block-end-gutter"))
    : fallbackSize.height;
  if (width <= 0 || height <= 0) return null;

  return { width, height };
}

function readFixedViewportHeight(element: HTMLElement, bounds: DOMRect, refresh: boolean) {
  if (fixedViewportHeightSnapshot !== null && !refresh) {
    return fixedViewportHeightSnapshot;
  }

  const declaredHeight = readDeclaredFixedViewportHeight(element, bounds, fixedViewportHeightSnapshot);
  fixedViewportHeightSnapshot = declaredHeight;
  return declaredHeight;
}

function readDeclaredFixedViewportHeight(element: HTMLElement, bounds: DOMRect, previousHeight: number | null) {
  const declaredHeight = readFixedViewportPixelSize(readPixelSize(element, "height"));
  const declaredMaxHeight = readFixedViewportPixelSize(readPixelSize(element, "maxHeight"));
  if (declaredHeight && (!declaredMaxHeight || declaredHeight <= declaredMaxHeight)) return declaredHeight;
  if (declaredMaxHeight) return declaredMaxHeight;

  const boundedHeight = readFixedViewportPixelSize(positiveFloor(bounds.height));
  if (boundedHeight) return boundedHeight;

  return previousHeight ?? fallbackSize.height;
}

function clampRenderedHeight(height: number) {
  return Math.min(Math.floor(height), fixedViewportHeightCap());
}

function readFixedViewportPixelSize(value: number | null) {
  if (!value) return null;
  const height = Math.floor(value);
  if (height <= 0 || height > fixedViewportHeightCap()) return null;
  return height;
}

function fixedViewportHeightCap() {
  const viewportHeight = window.innerHeight > 0 ? window.innerHeight : fallbackSize.height;
  return Math.min(maxRenderedChartHeight, Math.max(fallbackSize.height, viewportHeight));
}

function readHostWidth(element: HTMLElement, bounds = element.getBoundingClientRect()) {
  return readClientWidth(element) ?? readPixelSize(element, "width") ?? positiveFloor(bounds.width);
}

function readResizeEntryWidth(entry: ResizeObserverEntry) {
  const contentBox = Array.isArray(entry.contentBoxSize) ? entry.contentBoxSize[0] : entry.contentBoxSize;
  return positiveFloor(contentBox?.inlineSize ?? entry.contentRect.width);
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
</script>
