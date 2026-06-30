<template>
  <NTag v-if="loading" :bordered="false" size="small">
    {{ t("research.marketInvalidScanLoading") }}
  </NTag>
  <NTag v-else-if="error" :bordered="false" size="small" type="error" :title="error">
    {{ t("research.marketInvalidScanFailed") }}
  </NTag>
  <NTag
    v-else-if="scan"
    :bordered="false"
    role="button"
    size="small"
    tabindex="0"
    :title="title"
    :type="tagType"
    @click="detailsOpen = true"
    @keydown.enter="detailsOpen = true"
    @keydown.space.prevent="detailsOpen = true"
  >
    {{ label }}
  </NTag>
  <NModal v-model:show="detailsOpen" preset="card" :title="t('research.marketInvalidDetailsTitle')" class="research-modal">
    <NText v-if="scan" depth="3">{{ props.exchange }} / {{ props.symbol }} / {{ props.interval }}</NText>
    <NDataTable
      v-if="scan && scan.issues.length > 0"
      class="market-invalid-table"
      :columns="columns"
      :data="scan.issues"
      :bordered="false"
      size="small"
    />
    <NText v-else depth="3">{{ t("research.marketInvalidDetailsEmpty") }}</NText>
    <template #footer>
      <NSpace justify="end">
        <NTag v-if="scan?.limited" :bordered="false" type="warning">
          {{ t("research.marketInvalidDetailsLimited", { returned: scan.returnedCount, total: scan.totalCount }) }}
        </NTag>
        <MarketRepairResultTags :result="repairResult" />
        <NTag v-if="repairError" :bordered="false" type="error">
          {{ t("research.marketInvalidRepairFailed") }}
        </NTag>
        <NButton
          v-if="repairableOpenTimes.length > 0"
          secondary
          type="warning"
          :loading="repairing"
          @click="repairReturnedIssues"
        >
          {{ t("research.marketInvalidRepairReturned") }}
        </NButton>
        <NButton @click="detailsOpen = false">{{ t("common.close") }}</NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<script setup lang="ts">
import { NButton, NDataTable, NModal, NSpace, NTag, NText, type DataTableColumns, type TagProps } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

import { dataApi } from "@/services/api/data";
import type { CandleIssue, DataSyncGapRepairResult, MarketCandleInvalidIssueScan } from "@/types/app";
import MarketRepairResultTags from "./MarketRepairResultTags.vue";

const props = defineProps<{
  exchange: string;
  interval: string;
  symbol: string;
}>();
const emit = defineEmits<{
  repaired: [];
}>();

const { t, te } = useI18n();
const loading = ref(false);
const error = ref("");
const repairError = ref(false);
const repairing = ref(false);
const repairResult = ref<DataSyncGapRepairResult | null>(null);
const scan = ref<MarketCandleInvalidIssueScan | null>(null);
const detailsOpen = ref(false);
let requestSeq = 0;

const columns = computed<DataTableColumns<CandleIssue>>(() => [
  { title: t("research.invalidIssueOpenTime"), key: "openTime", minWidth: 180, render: (row) => formatWindowTime(row.openTime) },
  { title: t("research.invalidIssueType"), key: "code", minWidth: 170, render: (row) => invalidIssueLabel(row.code, row.message) },
  { title: t("research.invalidIssueMessage"), key: "message", minWidth: 220 },
]);

const tagType = computed<TagProps["type"]>(() => {
  if (!scan.value || scan.value.totalCount === 0) return "success";
  return "error";
});

const label = computed(() => {
  if (!scan.value) return "";
  if (scan.value.totalCount === 0) {
    return t("research.marketInvalidScanOK", { count: scan.value.window.count });
  }
  return t("research.marketInvalidScanIssue", { count: scan.value.totalCount });
});

const title = computed(() => {
  const firstIssue = scan.value?.issues[0];
  if (!scan.value || !firstIssue) return "";
  return t("research.marketInvalidScanFirst", {
    reason: invalidIssueLabel(firstIssue.code, firstIssue.message),
    time: formatWindowTime(firstIssue.openTime),
  });
});

const repairableOpenTimes = computed(() => scan.value?.issues
  .map((issue) => issue.openTime)
  .filter((openTime): openTime is string => Boolean(openTime)) ?? []);

watch(
  () => [props.exchange, props.symbol, props.interval],
  () => void loadScan(),
  { immediate: true },
);

async function loadScan(options: { clearRepairResult?: boolean } = {}) {
  const seq = ++requestSeq;
  loading.value = true;
  error.value = "";
  repairError.value = false;
  if (options.clearRepairResult ?? true) {
    repairResult.value = null;
  }
  scan.value = null;
  try {
    const result = await dataApi.scanMarketCandleInvalidIssues({
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

async function repairReturnedIssues() {
  if (repairableOpenTimes.value.length === 0) return;
  repairing.value = true;
  repairError.value = false;
  repairResult.value = null;
  try {
    const result = await dataApi.repairMarketCandleInvalidIssues({
      exchange: props.exchange,
      interval: props.interval,
      openTimes: repairableOpenTimes.value,
      symbol: props.symbol,
    });
    repairResult.value = result;
    emit("repaired");
    await loadScan({ clearRepairResult: false });
  } catch (repairFailure) {
    repairError.value = true;
  } finally {
    repairing.value = false;
  }
}

function invalidIssueLabel(code: string, fallback?: string) {
  if (!code) return fallback || t("research.invalidCandleIssue.unknown");
  const key = `research.invalidCandleIssue.${code}`;
  return te(key) ? t(key) : fallback || t("research.invalidCandleIssue.unknown");
}

function formatWindowTime(value?: string) {
  if (!value) return "-";
  return value.replace("T", " ").replace(/(?:\.\d+)?Z$/, " UTC");
}
</script>
