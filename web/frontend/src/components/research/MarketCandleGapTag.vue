<template>
  <NTag v-if="loading" :bordered="false" size="small">
    {{ t("research.marketGapScanLoading") }}
  </NTag>
  <NTag v-else-if="error" :bordered="false" size="small" type="error">
    {{ t("research.marketGapScanFailed") }}
  </NTag>
  <NTag v-else-if="scan" :bordered="false" size="small" :title="title" :type="tagType">
    {{ label }}
  </NTag>
</template>

<script setup lang="ts">
import { NTag, type TagProps } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

import { dataApi } from "@/services/api/data";
import type { MarketCandleGapScan } from "@/types/app";

const props = defineProps<{
  exchange: string;
  interval: string;
  symbol: string;
}>();

const { t } = useI18n();
const loading = ref(false);
const error = ref("");
const scan = ref<MarketCandleGapScan | null>(null);
let requestSeq = 0;

const tagType = computed<TagProps["type"]>(() => {
  if (!scan.value || scan.value.totalCount === 0) return "success";
  return "warning";
});

const label = computed(() => {
  if (!scan.value) return "";
  if (scan.value.totalCount === 0) {
    return t("research.marketGapScanOK", { count: scan.value.window.count });
  }
  return t("research.marketGapScanGap", { count: scan.value.totalCount });
});

const title = computed(() => {
  const firstGap = scan.value?.gaps[0];
  if (!scan.value || !firstGap) return "";
  return t("research.marketGapScanFirst", {
    from: formatWindowTime(firstGap.from),
    missing: firstGap.missingCandles,
    to: formatWindowTime(firstGap.to),
  });
});

watch(
  () => [props.exchange, props.symbol, props.interval],
  () => void loadScan(),
  { immediate: true },
);

async function loadScan() {
  const seq = ++requestSeq;
  loading.value = true;
  error.value = "";
  scan.value = null;
  try {
    const result = await dataApi.scanMarketCandleGaps({
      exchange: props.exchange,
      interval: props.interval,
      limit: 20,
      symbol: props.symbol,
    });
    if (seq === requestSeq) scan.value = result;
  } catch (scanError) {
    if (seq === requestSeq) error.value = scanError instanceof Error ? scanError.message : String(scanError);
  } finally {
    if (seq === requestSeq) loading.value = false;
  }
}

function formatWindowTime(value: string) {
  return value.replace("T", " ").replace(/(?:\.\d+)?Z$/, " UTC");
}
</script>
