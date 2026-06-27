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
          :tasks="tasks"
          @view="selectTask"
          @delete="deleteTask"
          @retry="retryTask"
          @toggle-realtime="toggleRealtime"
          @toggle-sync="toggleSync"
        />
        <EmptyState v-else :title="t('research.noTasks')" />
      </section>

      <section class="surface chart-panel research-chart-panel">
        <div class="research-toolbar">
          <div class="toolbar-row">
            <NSelect v-model:value="exchange" class="research-select" :options="exchangeOptions" />
            <NSelect v-model:value="symbol" class="research-select" :options="symbolOptions" />
            <NSelect v-model:value="interval" class="research-select research-select--compact" :options="intervalOptions" />
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
              <NTag v-if="candleResult.gaps.length > 0" :bordered="false" size="small" type="warning">
                {{ gapCountLabel }}
              </NTag>
              <NTag v-if="coverageLimited" :bordered="false" size="small" type="warning">
                {{ coverageLabel }}
              </NTag>
            </div>
          </div>
        </div>
        <div class="research-chart-body">
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
          <NSelect v-model:value="createForm.symbol" :options="symbolOptions" filterable tag />
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
  </section>
</template>

<script setup lang="ts">
import { Plus } from "@lucide/vue";
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
  type TagProps,
  type SelectOption,
} from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import TradingViewChart from "@/components/chart/TradingViewChart.vue";
import EmptyState from "@/components/common/EmptyState.vue";
import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import DataSyncTaskTable from "@/components/tables/DataSyncTaskTable.vue";
import { useResearchWorkspace } from "@/composables/useResearchWorkspace";

const { t } = useI18n();
const {
  canCreateTask,
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
  interval,
  loadCandles,
  loadTasks,
  openCreateTask,
  retryTask,
  selectTask,
  symbol,
  tasks,
  tasksError,
  tasksLoading,
  toggleRealtime,
  toggleSync,
} = useResearchWorkspace();

const exchangeOptions = computed<SelectOption[]>(() => [
  { label: "Binance", value: "binance" },
  { label: "OKX", value: "okx" },
]);

const symbolOptions = computed<SelectOption[]>(() => [
  { label: "BTCUSDT", value: "BTCUSDT" },
  { label: "ETHUSDT", value: "ETHUSDT" },
  { label: "BTC-USDT", value: "BTC-USDT" },
  { label: "ETH-USDT", value: "ETH-USDT" },
]);

const intervalOptions = computed<SelectOption[]>(() => [
  { label: "1m", value: "1m" },
  { label: "5m", value: "5m" },
  { label: "15m", value: "15m" },
  { label: "1h", value: "1h" },
  { label: "4h", value: "4h" },
  { label: "1d", value: "1d" },
]);

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
</script>

<style scoped>
.research-workspace {
  display: grid;
  grid-auto-rows: max-content;
  align-items: start;
  gap: 16px;
}

.research-tasks-panel {
  max-height: 360px;
  overflow: hidden;
}

.research-section-header {
  padding: 16px 16px 8px;
}

.research-section-header h2 {
  margin: 0;
  font-size: 18px;
  line-height: 1.3;
  font-weight: 760;
}

.research-chart-panel {
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  height: clamp(560px, calc(100vh - 220px), 760px);
  max-height: 760px;
  min-height: 0;
  overflow: hidden;
  align-self: start;
}

.research-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px;
  border-bottom: 1px solid var(--tt-line);
}

.research-context {
  display: flex;
  min-width: 0;
  align-items: flex-end;
  flex-direction: column;
  gap: 6px;
}

.research-meta {
  display: flex;
  justify-content: flex-end;
  gap: 6px;
  flex-wrap: wrap;
}

.research-chart-body {
  position: relative;
  display: block;
  flex: 1 1 0;
  box-sizing: border-box;
  width: 100%;
  max-width: 100%;
  height: 0;
  max-height: 100%;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  overflow: clip;
  contain: strict;
}

.research-chart-body :deep(.state-block),
.research-chart-body :deep(.trading-chart) {
  width: 100%;
  min-height: 0;
  height: 100%;
}

.research-select {
  width: 136px;
}

.research-select--compact {
  width: 96px;
}

:global(.research-modal) {
  width: min(560px, calc(100vw - 32px));
}

@media (max-width: 760px) {
  .research-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .research-context {
    align-items: flex-start;
  }

  .research-meta {
    justify-content: flex-start;
  }
}
</style>
