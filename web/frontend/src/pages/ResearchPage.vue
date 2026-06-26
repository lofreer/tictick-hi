<template>
  <section class="page">
    <header class="page-header">
      <div>
        <h1 class="page-title">{{ t("page.research.title") }}</h1>
        <p class="page-subtitle">{{ t("page.research.subtitle") }}</p>
      </div>
      <NButton type="primary">
        <template #icon>
          <Plus :size="17" />
        </template>
        {{ t("research.createTask") }}
      </NButton>
    </header>

    <div class="workspace-grid">
      <section class="surface chart-panel">
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
        <TradingViewChart :data="candles" :empty-title="t('research.chartEmpty')" />
      </section>

      <aside class="side-panel">
        <NCard :title="t('research.syncTasks')" :bordered="false" class="surface">
          <DataSyncTaskTable
            v-if="tasks.length > 0"
            :tasks="tasks"
            @view="selectTask"
            @delete="deleteTask"
            @toggle-realtime="toggleRealtime"
            @toggle-sync="toggleSync"
          />
          <EmptyState v-else :title="t('research.noTasks')" />
        </NCard>
      </aside>
    </div>
  </section>
</template>

<script setup lang="ts">
import { Plus } from "@lucide/vue";
import { NButton, NCard, NSelect, NText, useMessage, type SelectOption } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";

import TradingViewChart from "@/components/chart/TradingViewChart.vue";
import EmptyState from "@/components/common/EmptyState.vue";
import DataSyncTaskTable from "@/components/tables/DataSyncTaskTable.vue";
import type { ChartCandle, DataSyncTask } from "@/types/app";

const route = useRoute();
const router = useRouter();
const message = useMessage();
const { t } = useI18n();

const exchange = ref(readQuery("exchange", "binance"));
const symbol = ref(readQuery("symbol", "BTCUSDT"));
const interval = ref(readQuery("interval", "1m"));
const tasks = ref<DataSyncTask[]>([]);
const candles = ref<ChartCandle[]>([]);

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

watch([exchange, symbol, interval], () => {
  router.replace({
    name: "research",
    query: { exchange: exchange.value, symbol: symbol.value, interval: interval.value },
  });
});

function readQuery(key: string, fallback: string) {
  const value = route.query[key];
  return typeof value === "string" && value.length > 0 ? value : fallback;
}

function selectTask(task: DataSyncTask) {
  exchange.value = task.exchange;
  symbol.value = task.symbol;
  interval.value = task.interval;
}

function deleteTask() {
  message.info(t("page.placeholder"));
}

function toggleRealtime() {
  message.info(t("page.placeholder"));
}

function toggleSync() {
  message.info(t("page.placeholder"));
}
</script>

<style scoped>
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

@media (max-width: 760px) {
  .research-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
