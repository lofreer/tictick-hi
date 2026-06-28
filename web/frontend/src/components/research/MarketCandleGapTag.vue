<template>
  <NTag v-if="loading" :bordered="false" size="small">
    {{ t("research.marketGapScanLoading") }}
  </NTag>
  <NTag v-else-if="error" :bordered="false" size="small" type="error">
    {{ t("research.marketGapScanFailed") }}
  </NTag>
  <NTag v-else-if="scan" :bordered="false" role="button" size="small" tabindex="0" :title="title" :type="tagType" @click="detailsOpen = true" @keydown.enter="detailsOpen = true">
    {{ label }}
  </NTag>
  <NModal v-model:show="detailsOpen" preset="card" :title="t('research.marketGapDetailsTitle')" class="research-modal">
    <NText v-if="scan" depth="3">{{ props.exchange }} / {{ props.symbol }} / {{ props.interval }}</NText>
    <NDataTable
      v-if="scan && scan.gaps.length > 0"
      class="market-gap-table"
      :columns="columns"
      :data="scan.gaps"
      :bordered="false"
      size="small"
    />
    <NText v-else depth="3">{{ t("research.marketGapDetailsEmpty") }}</NText>
    <template #footer>
      <NSpace justify="end">
        <NTag v-if="scan?.limited" :bordered="false" type="warning">
          {{ t("research.marketGapDetailsLimited", { returned: scan.returnedCount, total: scan.totalCount }) }}
        </NTag>
        <NButton
          v-if="scan?.gaps.length"
          :loading="repairingKey === gapKey(scan.gaps[0])"
          secondary
          type="warning"
          @click="repairGap(scan.gaps[0])"
        >
          {{ t("research.marketGapRepairFirst") }}
        </NButton>
        <NButton @click="detailsOpen = false">{{ t("common.close") }}</NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<script setup lang="ts">
import { NButton, NDataTable, NModal, NSpace, NTag, NText, useMessage, type DataTableColumns, type TagProps } from "naive-ui";
import { computed, h, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

import { dataApi } from "@/services/api/data";
import type { CandleGap, MarketCandleGapScan } from "@/types/app";

const props = defineProps<{
  exchange: string;
  interval: string;
  symbol: string;
}>();
const emit = defineEmits<{ repaired: [] }>();

const { t } = useI18n();
const message = useMessage();
const loading = ref(false);
const error = ref("");
const scan = ref<MarketCandleGapScan | null>(null);
const detailsOpen = ref(false);
const repairingKey = ref("");
let requestSeq = 0;

const columns = computed<DataTableColumns<CandleGap>>(() => [
  { title: t("research.gapFrom"), key: "from", minWidth: 180, render: (row) => formatWindowTime(row.from) },
  { title: t("research.gapTo"), key: "to", minWidth: 180, render: (row) => formatWindowTime(row.to) },
  { title: t("research.missingCandles"), key: "missingCandles", width: 120 },
  {
    title: t("research.actions"),
    key: "actions",
    width: 120,
    render: (row) =>
      h(
        NButton,
        {
          loading: repairingKey.value === gapKey(row),
          secondary: true,
          size: "tiny",
          type: "warning",
          onClick: () => void repairGap(row),
        },
        () => t("research.marketGapRepair"),
      ),
  },
]);

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

async function repairGap(gap: CandleGap) {
  const key = gapKey(gap);
  repairingKey.value = key;
  try {
    const result = await dataApi.repairMarketCandleGap({
      exchange: props.exchange,
      symbol: props.symbol,
      interval: props.interval,
      from: gap.from,
      to: gap.to,
    });
    if (result.createdTasks.length > 0) {
      message.success(t("research.marketGapRepairQueued", { count: result.createdTasks.length }));
      emit("repaired");
    } else {
      message.success(t("research.taskGapRepairAlreadyQueued"));
    }
    await loadScan();
  } catch (repairError) {
    message.error(t("research.marketGapRepairFailed"));
  } finally {
    repairingKey.value = "";
  }
}

function gapKey(gap: CandleGap) {
  return `${gap.from}:${gap.to}`;
}

function formatWindowTime(value: string) {
  return value.replace("T", " ").replace(/(?:\.\d+)?Z$/, " UTC");
}
</script>
