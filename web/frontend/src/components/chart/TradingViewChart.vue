<template>
  <div ref="rootRef" class="trading-chart">
    <div ref="containerRef" class="trading-chart__canvas" />
    <div v-if="readout" class="trading-chart__readout" :class="`trading-chart__readout--${readout.direction}`">
      <span class="trading-chart__readout-time">{{ readout.timeLabel }}</span>
      <span>O {{ readout.open }}</span>
      <span>H {{ readout.high }}</span>
      <span>L {{ readout.low }}</span>
      <span>C {{ readout.close }}</span>
      <span>V {{ readout.volume }}</span>
      <span>{{ readout.change }} / {{ readout.changePct }}</span>
    </div>
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
  type MouseEventParams,
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
import { appColors, chartAxisFontSize, chartRightPriceScaleWidth, chartTheme } from "@/theme/tokens";
import type { ChartCandle, ChartMarker } from "@/types/app";
import { chartCandleForTime, chartReadoutFromCandle, type ChartReadout } from "./chartReadout";
import { positiveFloor, readClientHeight, readClientWidth, readPixelSize } from "./chartSizing";
import "./TradingViewChart.css";

const props = defineProps<{
  data: ChartCandle[];
  emptyTitle: string;
  markers?: ChartMarker[];
}>();

const themeStore = useThemeStore();
const rootRef = ref<HTMLDivElement | null>(null);
const containerRef = ref<HTMLDivElement | null>(null);
const readout = ref<ChartReadout | null>(null);
let chart: IChartApi | null = null;
let series: ISeriesApi<"Candlestick"> | null = null;
let volumeSeries: ISeriesApi<"Histogram"> | null = null;
let markerPlugin: ISeriesMarkersPluginApi<Time> | null = null;
let resizeObserver: ResizeObserver | null = null;
let observedResizeHost: HTMLElement | null = null;
let resizeFrame = 0;
let lastSize = { width: 0, height: 0 };
const fallbackSize = { width: 1, height: 640 };
const minRenderHeight = 360;
const maxRenderHeight = 860;
const maxInitialVisibleBars = 320;
const minInitialVisibleBars = 80;
const targetInitialBarSpacingPixels = 4;
const minTimeAxisEdgePaddingBars = 1;
const minTimeAxisEdgePaddingPixels = 12;
const maxTimeAxisEdgePaddingPixels = 18;
const timeAxisLabelInsetBars = 0;
const timeAxisEdgePaddingRatio = 0.012;
const volumePriceScaleId = "volume";
const volumeUpColor = "rgba(14, 203, 129, 0.28)";
const volumeDownColor = "rgba(246, 70, 93, 0.28)";

onMounted(() => {
  if (!rootRef.value || !containerRef.value) return;

  const initialSize = readHostSize() ?? { ...fallbackSize };
  lastSize = { width: initialSize.width, height: initialSize.height };
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
    resizeObserver = new ResizeObserver(handleObservedResize);
    resizeObserver.observe(observedResizeHost);
  }
  chart.subscribeCrosshairMove(handleCrosshairMove);
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
  window.removeEventListener("resize", handleWindowResize);
  chart?.unsubscribeCrosshairMove(handleCrosshairMove);
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
  updateLatestReadout();
  syncMarkers();
  fitChartContent(candleData.length);
}

function updateLatestReadout() {
  readout.value = chartReadoutFromCandle(props.data.at(-1));
}

function handleCrosshairMove(param: MouseEventParams<Time>) {
  readout.value = chartReadoutFromCandle(chartCandleForTime(props.data, param.time) ?? props.data.at(-1));
}

function configurePriceScales() {
  chart?.priceScale("right").applyOptions({
    scaleMargins: {
      top: 0.08,
      bottom: 0.24,
    },
  });
  chart?.priceScale(volumePriceScaleId).applyOptions({
    visible: false,
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
  return Math.max(1, lastSize.width - rightPriceScaleMinimumWidth());
}

function responsiveChartOptions(mode = themeStore.mode) {
  const theme = chartTheme(mode);
  return {
    ...theme,
    layout: {
      ...theme.layout,
      fontSize: chartFontSize(),
    },
    localization: {
      priceFormatter: formatChartPrice,
    },
    rightPriceScale: {
      ...theme.rightPriceScale,
      alignLabels: true,
      entireTextOnly: true,
      ticksVisible: false,
      ensureEdgeTickMarksVisible: false,
      minimumWidth: rightPriceScaleMinimumWidth(),
    },
    timeScale: {
      ...theme.timeScale,
      barSpacing: 5,
      minBarSpacing: 0.75,
      rightOffsetPixels: 0,
      secondsVisible: false,
      tickMarkMaxCharacterLength: 8,
      tickMarkFormatter: formatChartTickMark,
    },
  };
}

function chartFontSize() {
  return chartAxisFontSize;
}

function rightPriceScaleMinimumWidth() {
  if (lastSize.width <= 480) return chartRightPriceScaleWidth.mobile;
  if (lastSize.width <= 980) return chartRightPriceScaleWidth.narrowDesktop;
  return chartRightPriceScaleWidth.desktop;
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

  scheduleResize();
}

function handleWindowResize() {
  scheduleResize();
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

  if (nextMeasurement.width === lastSize.width && nextMeasurement.height === lastSize.height) {
    return;
  }

  lastSize = { width: nextMeasurement.width, height: nextMeasurement.height };
  chart.applyOptions(responsiveChartOptions());
  chart.resize(nextMeasurement.width, nextMeasurement.height);
  fitChartContent(props.data.length);
}

function readHostSize(): { width: number; height: number } | null {
  const host = readResizeHost();
  if (!host) return null;

  const bounds = host.getBoundingClientRect();
  const measuredWidth = readHostWidth(host, bounds);
  const measuredHeight = readHostHeight(host, bounds);
  const width = measuredWidth === null ? null : positiveFloor(measuredWidth);
  const height = measuredHeight === null ? null : positiveFloor(measuredHeight);
  if (width === null && height === null) return { ...fallbackSize };
  const renderWidth = positiveFloor(width === null ? fallbackSize.width : width);
  const renderHeight = positiveFloor(height === null ? fallbackSize.height : height);
  if (renderWidth === null || renderHeight === null) return null;

  return normalizeRenderSize(renderWidth, renderHeight);
}

function readHostWidth(element: HTMLElement, bounds = element.getBoundingClientRect()) {
  return readClientWidth(element) ?? readPixelSize(element, "width") ?? positiveFloor(bounds.width);
}

function readHostHeight(element: HTMLElement, bounds = element.getBoundingClientRect()) {
  const declaredHeight = readPixelSize(element, "height");
  const declaredMaxHeight = readPixelSize(element, "maxHeight");
  if (declaredHeight !== null && declaredMaxHeight !== null) {
    return Math.min(declaredHeight, declaredMaxHeight);
  }
  return declaredHeight ?? declaredMaxHeight ?? readClientHeight(element) ?? positiveFloor(bounds.height);
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

function normalizeRenderSize(width: number, height: number) {
  return {
    width,
    height: Math.min(maxRenderHeight, Math.max(minRenderHeight, height)),
  };
}

</script>
