<template>
  <div class="trading-chart">
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
  type CandlestickData,
  type IChartApi,
  type ISeriesApi,
  type Time,
} from "lightweight-charts";
import { onBeforeUnmount, onMounted, ref, watch } from "vue";

import EmptyState from "@/components/common/EmptyState.vue";
import { useThemeStore } from "@/stores/theme";
import { appColors, chartTheme } from "@/theme/tokens";
import type { ChartCandle } from "@/types/app";
import "./TradingViewChart.css";

const props = defineProps<{
  data: ChartCandle[];
  emptyTitle: string;
}>();

const themeStore = useThemeStore();
const containerRef = ref<HTMLDivElement | null>(null);
let chart: IChartApi | null = null;
let series: ISeriesApi<"Candlestick"> | null = null;
let observer: ResizeObserver | null = null;

onMounted(() => {
  if (!containerRef.value) return;

  chart = createChart(containerRef.value, {
    ...chartTheme(themeStore.mode),
    autoSize: true,
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

  observer = new ResizeObserver(() => chart?.timeScale().fitContent());
  observer.observe(containerRef.value);
  syncData();
});

onBeforeUnmount(() => {
  observer?.disconnect();
  chart?.remove();
});

watch(
  () => props.data,
  () => syncData(),
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
  chart?.timeScale().fitContent();
}
</script>
