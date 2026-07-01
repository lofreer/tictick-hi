<template>
  <section class="page">
    <header class="page-header">
      <div>
        <h1 class="page-title">{{ t("page.research.title") }}</h1>
        <p class="page-subtitle">{{ t("page.research.subtitle") }}</p>
      </div>
      <NButton type="primary" @click="openCreateTask">
        <template #icon>
          <Plus :size="17" />
        </template>
        {{ t("research.createTask") }}
      </NButton>
    </header>

    <div class="research-workspace">
      <section class="surface research-tasks-panel">
        <div class="research-section-header">
          <h2>{{ t("research.syncTasks") }}</h2>
        </div>
        <LoadingState v-if="tasksLoading" />
        <ErrorState
          v-else-if="tasksError"
          :title="tasksError"
          retryable
          @retry="loadTasks"
        />
        <DataSyncTaskTable
          v-else-if="tasks.length > 0"
          :repairing-task-id="repairTaskGapsLoadingId"
          :tasks="tasks"
          @view="selectTask"
          @view-gaps="viewTaskGaps"
          @view-invalid="viewTaskInvalidIssues"
          @delete="deleteTask"
          @repair-gaps="repairTaskGapsAndPoll"
          @retry="retryTask"
          @toggle-realtime="toggleRealtime"
          @toggle-sync="toggleSync"
        />
        <EmptyState v-else :title="t('research.noTasks')" />
      </section>

      <section class="surface research-chart-panel">
        <div class="research-toolbar">
          <div class="research-toolbar-main">
            <div class="research-source-controls">
              <NSelect
                v-model:value="exchange"
                class="research-select research-select--exchange"
                :options="exchangeOptions"
                size="small"
              />
              <MarketSymbolAutoComplete
                v-model:value="symbol"
                class="research-symbol-input"
                :exchange="exchange"
                :show-sync-button="false"
                size="small"
                @synced="loadMarketInstrumentSyncStatuses"
              />
              <NButton
                class="research-refresh-button"
                circle
                secondary
                size="small"
                :aria-label="t('research.refreshChart')"
                :loading="candlesLoading"
                :title="t('research.refreshChart')"
                @click="refreshChartCandles"
              >
                <template #icon>
                  <RefreshCw :size="15" />
                </template>
              </NButton>
              <NSelect
                v-model:value="interval"
                class="research-select research-select--compact"
                :options="intervalOptions"
                size="small"
              />
              <ResearchWindowControls
                :can-load-next="canLoadNextCandles"
                :can-load-previous="canLoadPreviousCandles"
                :loading="candlesLoading"
                @next="loadNextChartCandles"
                @previous="loadPreviousChartCandles"
                @range="applyChartTimeRange"
              />
            </div>
          </div>
          <div class="research-toolbar-status">
            <NText class="research-current-source" depth="3">
              {{ t("research.currentDataSource") }}: {{ exchange }} / {{ symbol }} / {{ interval }}
            </NText>
            <div v-if="candleResult || currentMarketInstrumentSyncStatus || marketInstrumentSyncStatusError" class="research-meta">
              <NTag
                v-if="currentMarketInstrumentSyncStatus"
                :bordered="false"
                size="small"
                :title="catalogStatusDetail(currentMarketInstrumentSyncStatus)"
                :type="catalogStatusTagType(currentMarketInstrumentSyncStatus)"
              >
                {{ t("research.instrumentCatalog") }}: {{ catalogStatusLabel(currentMarketInstrumentSyncStatus) }}
              </NTag>
              <NTag
                v-else-if="marketInstrumentSyncStatusError"
                :bordered="false"
                size="small"
                type="warning"
                :title="marketInstrumentSyncStatusError"
              >
                {{ t("research.instrumentCatalog") }}: {{ t("research.instrumentCatalogUnknown") }}
              </NTag>
              <NTag v-if="candleResult" :bordered="false" size="small" :type="sourceTagType">
                {{ t("research.candleSource") }}: {{ sourceLabel }}
              </NTag>
              <NTag v-if="candleResult" :bordered="false" size="small" :type="healthTagType">
                {{ t("research.dataHealth") }}: {{ healthLabel }}
              </NTag>
              <NTag v-if="firstCandleIssue" :bordered="false" size="small" type="error" :title="candleIssueTitle">
                {{ candleIssueLabel }}
              </NTag>
              <NTag v-if="candleResult" :bordered="false" size="small">
                {{ t("research.baseInterval") }}: {{ baseIntervalText }}
              </NTag>
              <NTag v-if="windowLabel" :bordered="false" size="small">
                {{ windowLabel }}
              </NTag>
              <NTag v-if="candleResult && candleResult.gaps.length > 0" :bordered="false" size="small" type="warning">
                {{ gapCountLabel }}
              </NTag>
              <MarketCandleGapTag :exchange="exchange" :interval="interval" :symbol="symbol" :tasks="tasks" @repaired="startRepairPollingForResult" />
              <MarketCandleInvalidIssueTag :exchange="exchange" :interval="interval" :symbol="symbol" :tasks="tasks" @repaired="startRepairPollingForResult" @quarantined="refreshAfterMarketCandleQuarantine" />
              <NTag v-if="coverageVisible" :bordered="false" size="small" :type="coverageTagType">
                {{ coverageLabel }}
              </NTag>
              <NButton
                v-if="canRepairGap"
                size="tiny"
                secondary
                type="warning"
                :loading="repairGapLoading"
                @click="repairFirstChartGap"
              >
                {{ t("research.repairFirstGap") }}
              </NButton>
              <MarketRepairResultTags :candle-result="candleResult" :result="chartGapRepairResult" :tasks="tasks" />
              <ChartInvalidIssueRepairAction :candle-result="candleResult" :exchange="exchange" :interval="candleResult?.baseInterval || interval" :issue="firstCandleIssue" :load-candles="loadCandles" :load-tasks="loadTasks" :symbol="symbol" :tasks="tasks" @repaired="(result) => startRepairPollingForResult(result, { immediate: false })" />
            </div>
          </div>
        </div>
        <div class="kline-chart-frame research-chart-body">
          <div class="kline-chart-frame__viewport research-chart-viewport" data-chart-viewport="fixed">
            <ErrorState
              v-if="candlesError"
              :title="candlesError"
              retryable
              @retry="loadCandles"
            />
            <LoadingState v-else-if="candlesLoading" />
            <TradingViewChart v-else :data="candles" :markers="chartMarkers" :empty-title="t('research.chartEmpty')" />
          </div>
        </div>
      </section>
    </div>

    <NModal
      v-model:show="createModalOpen"
      preset="card"
      :title="t('research.createTaskTitle')"
      class="research-modal"
    >
      <NForm>
        <NFormItem :label="t('research.exchange')">
          <NSelect v-model:value="createForm.exchange" :options="exchangeOptions" />
        </NFormItem>
        <NFormItem :label="t('research.symbol')">
          <MarketSymbolAutoComplete
            v-model:value="createForm.symbol"
            :exchange="createForm.exchange"
            @synced="loadMarketInstrumentSyncStatuses"
          />
        </NFormItem>
        <NFormItem v-if="createMarketInstrumentSyncStatus || marketInstrumentSyncStatusError" :label="t('research.instrumentCatalog')">
          <NTag
            v-if="createMarketInstrumentSyncStatus"
            :bordered="false"
            :title="catalogStatusDetail(createMarketInstrumentSyncStatus)"
            :type="catalogStatusTagType(createMarketInstrumentSyncStatus)"
          >
            {{ catalogStatusDetail(createMarketInstrumentSyncStatus) }}
          </NTag>
          <NTag v-else :bordered="false" type="warning" :title="marketInstrumentSyncStatusError">
            {{ t("research.instrumentCatalogUnknown") }}
          </NTag>
        </NFormItem>
        <NFormItem :label="t('research.interval')">
          <NSelect v-model:value="createForm.interval" :options="intervalOptions" />
        </NFormItem>
        <NFormItem :label="t('research.startTime')">
          <NDatePicker v-model:value="createForm.startTime" type="datetime" clearable />
        </NFormItem>
        <NFormItem :label="t('research.endTime')">
          <NDatePicker v-model:value="createForm.endTime" type="datetime" clearable />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="createModalOpen = false">{{ t("common.cancel") }}</NButton>
          <NButton
            type="primary"
            :loading="createLoading"
            :disabled="!canCreateTask"
            @click="createTask"
          >
            {{ t("common.create") }}
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <ResearchTaskGapDetailsModal
      v-model:show="gapDetailsModalOpen"
      :details="gapDetails"
      :error="gapDetailsError"
      :loading="gapDetailsLoading"
      :repair-loading="repairTaskGapsLoadingId === gapDetailsTask?.id"
      :repair-notice="taskGapRepairNotice"
      :repair-notice-type="taskGapRepairNoticeType"
      :repair-result="taskGapRepairResult"
      :task="gapDetailsTask"
      :tasks="tasks"
      @repair="gapDetailsTask && repairTaskGapsAndPoll(gapDetailsTask)"
      @retry="gapDetailsTask && viewTaskGaps(gapDetailsTask)"
    />

    <ResearchTaskInvalidIssueModal ref="invalidIssueModal" :tasks="tasks" @repaired="startRepairPollingForResult" />
  </section>
</template>

<script setup lang="ts">
import { Plus, RefreshCw } from "@lucide/vue";
import {
  NButton,
  NDatePicker,
  NForm,
  NFormItem,
  NModal,
  NSelect,
  NSpace,
  NTag,
  NText,
  type SelectOption,
  type TagProps,
} from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

import TradingViewChart from "@/components/chart/TradingViewChart.vue";
import EmptyState from "@/components/common/EmptyState.vue";
import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import MarketSymbolAutoComplete from "@/components/market/MarketSymbolAutoComplete.vue";
import ChartInvalidIssueRepairAction from "@/components/research/ChartInvalidIssueRepairAction.vue";
import MarketCandleGapTag from "@/components/research/MarketCandleGapTag.vue";
import MarketCandleInvalidIssueTag from "@/components/research/MarketCandleInvalidIssueTag.vue";
import MarketRepairResultTags from "@/components/research/MarketRepairResultTags.vue";
import ResearchTaskGapDetailsModal from "@/components/research/ResearchTaskGapDetailsModal.vue";
import ResearchTaskInvalidIssueModal from "@/components/research/ResearchTaskInvalidIssueModal.vue";
import ResearchWindowControls from "@/components/research/ResearchWindowControls.vue";
import DataSyncTaskTable from "@/components/tables/DataSyncTaskTable.vue";
import { useResearchRepairTaskPolling } from "@/composables/useResearchRepairTaskPolling";
import { useResearchWorkspace } from "@/composables/useResearchWorkspace";
import type { CandleIssue, DataSyncGapRepairResult, DataSyncTask, MarketInstrumentSyncStatus } from "@/types/app";
import { candleCoverageLabel, candleCoverageTagType, shouldShowCandleCoverage } from "./researchCandleCoverage";
import "./ResearchPage.css";
import "./klineChartLayout.css";

const { t, te } = useI18n();
const invalidIssueModal = ref<InstanceType<typeof ResearchTaskInvalidIssueModal> | null>(null);
const chartGapRepairResult = ref<DataSyncGapRepairResult | null>(null);
const {
  canCreateTask,
  canLoadNextCandles,
  canLoadPreviousCandles,
  canRepairGap,
  candleResult,
  candles,
  candlesError,
  candlesLoading,
  chartMarkers,
  createForm,
  createLoading,
  createModalOpen,
  createMarketInstrumentSyncStatus,
  createTask,
  currentMarketInstrumentSyncStatus,
  deleteTask,
  exchange,
  applyTimeRange,
  gapDetails,
  gapDetailsError,
  gapDetailsLoading,
  gapDetailsModalOpen,
  gapDetailsTask,
  interval,
  loadCandles,
  loadMarketInstrumentSyncStatuses,
  loadNextCandles,
  loadPreviousCandles,
  loadTasks,
  openCreateTask,
  repairFirstGap,
  repairGapLoading,
  repairTaskGaps,
  repairTaskGapsLoadingId,
  retryTask,
  selectTask,
  symbol,
  marketInstrumentSyncStatusError,
  tasks,
  tasksError,
  tasksLoading,
  taskGapRepairNotice,
  taskGapRepairNoticeType,
  taskGapRepairResult,
  toggleRealtime,
  toggleSync,
  viewTaskGaps,
} = useResearchWorkspace();
const { startRepairTaskPolling } = useResearchRepairTaskPolling(loadTasks);

const exchangeOptions = computed<SelectOption[]>(() => [
  { label: "Binance", value: "binance" },
  { label: "OKX", value: "okx" },
]);

const intervalOptions = computed<SelectOption[]>(() => [
  { label: "1m", value: "1m" },
  { label: "5m", value: "5m" },
  { label: "15m", value: "15m" },
  { label: "1h", value: "1h" },
  { label: "4h", value: "4h" },
  { label: "1d", value: "1d" },
]);

function viewTaskInvalidIssues(task: DataSyncTask) { invalidIssueModal.value?.open(task); }

async function repairFirstChartGap() {
  chartGapRepairResult.value = await repairFirstGap() ?? null;
  if (chartGapRepairResult.value) startRepairPollingForResult(chartGapRepairResult.value, { immediate: false });
}

async function repairTaskGapsAndPoll(task: DataSyncTask) {
  await repairTaskGaps(task);
  if (taskGapRepairResult.value) startRepairPollingForResult(taskGapRepairResult.value, { immediate: false });
}

async function refreshChartCandles() {
  resetChartRepairResults();
  await loadCandles();
}

function loadPreviousChartCandles() { resetChartRepairResults(); loadPreviousCandles(); }

function loadNextChartCandles() { resetChartRepairResults(); loadNextCandles(); }

function applyChartTimeRange(...args: Parameters<typeof applyTimeRange>) { resetChartRepairResults(); applyTimeRange(...args); }

watch([exchange, symbol, interval], () => resetChartRepairResults());

function resetChartRepairResults() { chartGapRepairResult.value = null; }

function startRepairPollingForResult(result: DataSyncGapRepairResult, options: { immediate?: boolean } = {}) {
  startRepairTaskPolling({
    immediate: options.immediate ?? true,
    onExhausted: loadCandles,
    onSettled: loadCandles,
    repairTaskIds: result.createdTasks.map((task) => task.id),
  });
}

async function refreshAfterMarketCandleQuarantine() { await Promise.all([loadTasks(), loadCandles()]); }

const sourceLabel = computed(() => t(`research.candleSource.${candleResult.value?.source ?? "none"}`));
const healthLabel = computed(() => t(`research.dataHealth.${candleResult.value?.health ?? "insufficient"}`));
const firstCandleIssue = computed<CandleIssue | null>(() => candleResult.value?.issues[0] ?? null);
const candleIssueReason = computed(() => invalidIssueLabel(firstCandleIssue.value));
const candleIssueTitle = computed(() => firstCandleIssue.value?.message || candleIssueReason.value);
const candleIssueLabel = computed(() => {
  const openTime = firstCandleIssue.value?.openTime;
  if (!openTime) return t("research.candleIssueNoTime", { reason: candleIssueReason.value });
  return t("research.candleIssue", { time: formatWindowTime(openTime), reason: candleIssueReason.value });
});
const baseIntervalText = computed(() => candleResult.value?.baseInterval ?? "-");
const gapCountLabel = computed(() => t("research.gapCount", { count: candleResult.value?.gaps.length ?? 0 }));
const coverageVisible = computed(() => shouldShowCandleCoverage(candleResult.value));
const coverageTagType = computed<TagProps["type"]>(() => candleCoverageTagType(candleResult.value));
const coverageLabel = computed(() => candleCoverageLabel(candleResult.value, t));
const windowLabel = computed(() => {
  const window = candleResult.value?.window;
  if (!window?.from || !window.to || window.count === 0) return "";
  return t("research.candleWindow", { from: formatWindowTime(window.from), to: formatWindowTime(window.to), count: window.count });
});
const sourceTagType = computed<TagProps["type"]>(() => {
  if (candleResult.value?.source === "native") return "success";
  if (candleResult.value?.source === "aggregated") return "info";
  return "default";
});
const healthTagType = computed<TagProps["type"]>(() => {
  if (candleResult.value?.health === "ok") return "success";
  if (candleResult.value?.health === "gap") return "warning";
  if (candleResult.value?.health === "invalid") return "error";
  return "default";
});

function catalogStatusTagType(status: MarketInstrumentSyncStatus): TagProps["type"] {
  return status.lastError ? "warning" : "success";
}

function catalogStatusLabel(status: MarketInstrumentSyncStatus) {
  if (status.lastError) return t("research.instrumentCatalogFailed");
  return t("research.instrumentCatalogOK");
}

function catalogStatusDetail(status: MarketInstrumentSyncStatus) {
  if (status.lastError) {
    return t("research.instrumentCatalogFailedDetail", {
      exchange: status.exchange,
      error: status.lastError,
      time: formatWindowTime(status.lastAttemptAt),
    });
  }
  return t("research.instrumentCatalogOKDetail", {
    exchange: status.exchange,
    time: formatWindowTime(status.lastSuccessAt ?? status.lastAttemptAt),
  });
}

function invalidIssueLabel(issue: CandleIssue | null) {
  if (!issue?.code) return issue?.message || t("research.invalidCandleIssue.unknown");
  const key = `research.invalidCandleIssue.${issue.code}`;
  return te(key) ? t(key) : issue.message || t("research.invalidCandleIssue.unknown");
}

function formatWindowTime(value: string) {
  return value.replace("T", " ").replace(/(?:\.\d+)?Z$/, " UTC");
}
</script>
