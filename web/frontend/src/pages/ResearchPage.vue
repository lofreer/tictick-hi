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
          @delete="deleteTask"
          @repair-gaps="repairTaskGaps"
          @retry="retryTask"
          @toggle-realtime="toggleRealtime"
          @toggle-sync="toggleSync"
        />
        <EmptyState v-else :title="t('research.noTasks')" />
      </section>

      <section class="surface research-chart-panel">
        <div class="research-toolbar">
          <div class="toolbar-row">
            <NSelect v-model:value="exchange" class="research-select" :options="exchangeOptions" />
            <MarketSymbolAutoComplete v-model:value="symbol" class="research-symbol-input" :exchange="exchange" />
            <NSelect v-model:value="interval" class="research-select research-select--compact" :options="intervalOptions" />
            <ResearchWindowControls
              :can-load-next="canLoadNextCandles"
              :can-load-previous="canLoadPreviousCandles"
              :loading="candlesLoading"
              @next="loadNextCandles"
              @previous="loadPreviousCandles"
            />
          </div>
          <div class="research-context">
            <NText depth="3">
              {{ t("research.currentDataSource") }}: {{ exchange }} / {{ symbol }} / {{ interval }}
            </NText>
            <div v-if="candleResult" class="research-meta">
              <NTag :bordered="false" size="small" :type="sourceTagType">
                {{ t("research.candleSource") }}: {{ sourceLabel }}
              </NTag>
              <NTag :bordered="false" size="small" :type="healthTagType">
                {{ t("research.dataHealth") }}: {{ healthLabel }}
              </NTag>
              <NTag :bordered="false" size="small">
                {{ t("research.baseInterval") }}: {{ baseIntervalText }}
              </NTag>
              <NTag v-if="windowLabel" :bordered="false" size="small">
                {{ windowLabel }}
              </NTag>
              <NTag v-if="candleResult.gaps.length > 0" :bordered="false" size="small" type="warning">
                {{ gapCountLabel }}
              </NTag>
              <MarketCandleGapTag :exchange="exchange" :interval="interval" :symbol="symbol" @repaired="loadTasks" />
              <NTag v-if="coverageLimited" :bordered="false" size="small" type="warning">
                {{ coverageLabel }}
              </NTag>
              <NButton
                v-if="canRepairGap"
                size="tiny"
                secondary
                type="warning"
                :loading="repairGapLoading"
                @click="repairFirstGap"
              >
                {{ t("research.repairFirstGap") }}
              </NButton>
            </div>
          </div>
        </div>
        <div class="research-chart-body" data-chart-viewport="fixed">
          <ErrorState
            v-if="candlesError"
            :title="candlesError"
            retryable
            @retry="loadCandles"
          />
          <LoadingState v-else-if="candlesLoading" />
          <TradingViewChart v-else :data="candles" :empty-title="t('research.chartEmpty')" />
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
          <MarketSymbolAutoComplete v-model:value="createForm.symbol" :exchange="createForm.exchange" />
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

    <NModal
      v-model:show="gapDetailsModalOpen"
      preset="card"
      :title="t('research.gapDetailsTitle')"
      class="research-modal"
    >
      <div v-if="gapDetailsTask" class="research-gap-context">
        <NText depth="3">
          {{ gapDetailsTask.exchange }} / {{ gapDetailsTask.symbol }} / {{ gapDetailsTask.interval }}
        </NText>
      </div>
      <LoadingState v-if="gapDetailsLoading" />
      <ErrorState
        v-else-if="gapDetailsError"
        :title="gapDetailsError"
        retryable
        @retry="gapDetailsTask && viewTaskGaps(gapDetailsTask)"
      />
      <EmptyState
        v-else-if="!gapDetails || gapDetails.gaps.length === 0"
        :title="t('research.noGapDetails')"
      />
      <NDataTable
        v-else
        :columns="gapDetailColumns"
        :data="gapDetails.gaps"
        :bordered="false"
        size="small"
      />
      <template #footer>
        <NSpace justify="end">
          <NTag v-if="gapDetails?.limited" :bordered="false" type="warning">
            {{
              t("research.gapDetailsLimited", {
                returned: gapDetails.returnedCount,
                total: gapDetails.totalCount,
                limit: gapDetails.repairLimit,
              })
            }}
          </NTag>
          <NButton @click="gapDetailsModalOpen = false">{{ t("common.close") }}</NButton>
        </NSpace>
      </template>
    </NModal>
  </section>
</template>

<script setup lang="ts">
import { Plus } from "@lucide/vue";
import {
  NButton,
  NDataTable,
  NDatePicker,
  NForm,
  NFormItem,
  NModal,
  NSelect,
  NSpace,
  NTag,
  NText,
  type DataTableColumns,
  type SelectOption,
  type TagProps,
} from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import TradingViewChart from "@/components/chart/TradingViewChart.vue";
import EmptyState from "@/components/common/EmptyState.vue";
import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import MarketSymbolAutoComplete from "@/components/market/MarketSymbolAutoComplete.vue";
import MarketCandleGapTag from "@/components/research/MarketCandleGapTag.vue";
import ResearchWindowControls from "@/components/research/ResearchWindowControls.vue";
import DataSyncTaskTable from "@/components/tables/DataSyncTaskTable.vue";
import { useResearchWorkspace } from "@/composables/useResearchWorkspace";
import type { CandleGap } from "@/types/app";
import "./ResearchPage.css";

const { t } = useI18n();
const {
  canCreateTask,
  canLoadNextCandles,
  canLoadPreviousCandles,
  canRepairGap,
  candleResult,
  candles,
  candlesError,
  candlesLoading,
  createForm,
  createLoading,
  createModalOpen,
  createTask,
  deleteTask,
  exchange,
  gapDetails,
  gapDetailsError,
  gapDetailsLoading,
  gapDetailsModalOpen,
  gapDetailsTask,
  interval,
  loadCandles,
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
  tasks,
  tasksError,
  tasksLoading,
  toggleRealtime,
  toggleSync,
  viewTaskGaps,
} = useResearchWorkspace();

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

const gapDetailColumns = computed<DataTableColumns<CandleGap>>(() => [{ title: t("research.gapFrom"), key: "from", minWidth: 180 }, { title: t("research.gapTo"), key: "to", minWidth: 180 }, { title: t("research.missingCandles"), key: "missingCandles", width: 120 }]);

const sourceLabel = computed(() => t(`research.candleSource.${candleResult.value?.source ?? "none"}`));
const healthLabel = computed(() => t(`research.dataHealth.${candleResult.value?.health ?? "insufficient"}`));
const baseIntervalText = computed(() => candleResult.value?.baseInterval ?? "-");
const gapCountLabel = computed(() => t("research.gapCount", { count: candleResult.value?.gaps.length ?? 0 }));
const coverageLimited = computed(() => candleResult.value?.coverage.limitedByBaseWindow ?? false);
const coverageLabel = computed(() =>
  t("research.coverageLimited", {
    requested: candleResult.value?.coverage.requestedLimit ?? 0,
    returned: candleResult.value?.coverage.returnedCandles ?? 0,
  }),
);
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
  return "default";
});

function formatWindowTime(value: string) {
  return value.replace("T", " ").replace(/(?:\.\d+)?Z$/, " UTC");
}
</script>
