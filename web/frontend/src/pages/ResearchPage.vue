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
          <NText depth="3">
            {{ t("research.source") }}: {{ exchange }} / {{ symbol }} / {{ interval }}
          </NText>
        </div>
        <ErrorState
          v-if="candlesError"
          :title="candlesError"
          retryable
          @retry="loadCandles"
        />
        <LoadingState v-else-if="candlesLoading" />
        <TradingViewChart v-else :data="candles" :empty-title="t('research.chartEmpty')" />
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
  NText,
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
</script>

<style scoped>
.research-workspace {
  display: grid;
  gap: 16px;
}

.research-tasks-panel {
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
  min-height: 560px;
}

.research-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px;
  border-bottom: 1px solid var(--tt-line);
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
}
</style>
