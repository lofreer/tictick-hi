<template>
  <section class="page">
    <header class="page-header">
      <div>
        <h1 class="page-title">{{ task?.name ?? t("page.backtestsDetail.title") }}</h1>
        <p class="page-subtitle">{{ subtitle }}</p>
      </div>
      <NButton @click="router.push({ name: 'backtests' })">
        <template #icon>
          <ArrowLeft :size="17" />
        </template>
        {{ t("backtests.backToList") }}
      </NButton>
    </header>

    <LoadingState v-if="taskLoading" />
    <ErrorState v-else-if="taskError" :title="taskError" retryable @retry="loadDetail" />
    <div v-else-if="task" class="workspace-grid">
      <section class="surface chart-panel backtest-chart-panel">
        <ErrorState v-if="candlesError" :title="candlesError" retryable @retry="loadCandles" />
        <LoadingState v-else-if="candlesLoading" />
        <TradingViewChart v-else :data="candles" :empty-title="t('backtests.chartEmpty')" />
      </section>

      <aside class="side-panel">
        <section class="surface backtest-side-section">
          <div class="backtest-side-section__heading">
            <h2>{{ t("backtests.summary") }}</h2>
            <StatusBadge :status="task.status" />
          </div>
          <dl class="backtest-summary-list">
            <template v-for="row in summaryRows" :key="row.label">
              <dt>{{ row.label }}</dt>
              <dd>{{ row.value }}</dd>
            </template>
          </dl>
        </section>

        <section class="surface backtest-side-section">
          <h2>{{ t("backtests.parameters") }}</h2>
          <dl class="backtest-summary-list">
            <template v-for="row in paramRows" :key="row.label">
              <dt>{{ row.label }}</dt>
              <dd>{{ row.value }}</dd>
            </template>
          </dl>
        </section>

        <section class="surface backtest-side-section">
          <h2>{{ t("backtests.orders") }}</h2>
          <LoadingState v-if="ordersLoading" />
          <ErrorState v-else-if="ordersError" :title="ordersError" retryable @retry="loadOrders" />
          <EmptyState v-else-if="orders.length === 0" :title="t('backtests.noOrders')" />
          <div v-else class="orders-list">
            <div v-for="order in orders" :key="order.id" class="orders-list__item">
              <NTag :type="order.side === 'buy' ? 'success' : 'error'" size="small">
                {{ order.side }}
              </NTag>
              <div>
                <strong>{{ order.price }}</strong>
                <span>{{ formatDate(order.occurredAt) }}</span>
              </div>
              <NText depth="3">{{ order.quantity }} / {{ order.status }}</NText>
            </div>
          </div>
        </section>
      </aside>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ArrowLeft } from "@lucide/vue";
import { NButton, NTag, NText } from "naive-ui";
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";

import TradingViewChart from "@/components/chart/TradingViewChart.vue";
import EmptyState from "@/components/common/EmptyState.vue";
import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import StatusBadge from "@/components/common/StatusBadge.vue";
import { backtestsApi } from "@/services/api/backtests";
import { dataApi } from "@/services/api/data";
import type { BacktestOrder, BacktestTask, ChartCandle } from "@/types/app";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const task = ref<BacktestTask | null>(null);
const orders = ref<BacktestOrder[]>([]);
const candles = ref<ChartCandle[]>([]);
const taskLoading = ref(false);
const ordersLoading = ref(false);
const candlesLoading = ref(false);
const taskError = ref("");
const ordersError = ref("");
const candlesError = ref("");

const backtestID = computed(() => String(route.params.id ?? ""));
const subtitle = computed(() => {
  if (!task.value) {
    return "";
  }
  return `${task.value.exchange} / ${task.value.symbol} / ${task.value.interval}`;
});
const summaryRows = computed(() => {
  if (!task.value) {
    return [];
  }
  return [
    { label: t("research.exchange"), value: task.value.exchange },
    { label: t("research.symbol"), value: task.value.symbol },
    { label: t("research.interval"), value: task.value.interval },
    { label: t("research.startTime"), value: formatDate(task.value.startTime) },
    { label: t("research.endTime"), value: formatDate(task.value.endTime) },
    { label: t("strategy.strategy"), value: task.value.strategyId },
    { label: t("strategy.initialBalance"), value: task.value.initialBalance },
    { label: t("strategy.feeBps"), value: task.value.feeBps },
    { label: t("strategy.slippageBps"), value: task.value.slippageBps },
    { label: t("strategy.triggerMode"), value: triggerModeLabel(task.value.triggerMode) },
    { label: t("backtests.finalEquity"), value: summaryValue("finalEquity") },
    { label: t("backtests.returnPct"), value: summaryValue("returnPct") },
    { label: t("backtests.totalOrders"), value: summaryValue("totalOrders") },
  ];
});
const paramRows = computed(() => {
  if (!task.value) {
    return [];
  }
  const rows = Object.entries(task.value.strategyParams).map(([label, value]) => ({
    label,
    value: String(value),
  }));
  return rows.length > 0 ? rows : [{ label: "-", value: "-" }];
});

onMounted(() => {
  void loadDetail();
});

async function loadDetail() {
  taskLoading.value = true;
  taskError.value = "";
  try {
    task.value = await backtestsApi.getBacktest(backtestID.value);
    await Promise.all([loadOrders(), loadCandles()]);
  } catch (loadError) {
    task.value = null;
    taskError.value = errorMessage(loadError, t("backtests.detailLoadFailed"));
  } finally {
    taskLoading.value = false;
  }
}

async function loadOrders() {
  if (!task.value) return;

  ordersLoading.value = true;
  ordersError.value = "";
  try {
    orders.value = await backtestsApi.listOrders(task.value.id);
  } catch (loadError) {
    orders.value = [];
    ordersError.value = errorMessage(loadError, t("backtests.ordersLoadFailed"));
  } finally {
    ordersLoading.value = false;
  }
}

async function loadCandles() {
  if (!task.value) return;

  candlesLoading.value = true;
  candlesError.value = "";
  try {
    candles.value = await dataApi.listCandles({
      exchange: task.value.exchange,
      symbol: task.value.symbol,
      interval: task.value.interval,
      from: task.value.startTime,
      to: task.value.endTime,
      limit: 1000,
    });
  } catch (loadError) {
    candles.value = [];
    candlesError.value = errorMessage(loadError, t("research.candlesLoadFailed"));
  } finally {
    candlesLoading.value = false;
  }
}

function triggerModeLabel(value: string) {
  return value === "minute_replay" ? t("strategy.minuteReplay") : t("strategy.closedCandle");
}

function summaryValue(key: string) {
  if (!task.value) {
    return "-";
  }
  const value = task.value.resultSummary[key];
  return value === undefined || value === null || value === "" ? "-" : String(value);
}

function formatDate(value?: string) {
  return value ? new Date(value).toLocaleString() : "-";
}

function errorMessage(loadError: unknown, fallback: string) {
  if (loadError instanceof Error && loadError.message) {
    return loadError.message;
  }
  return fallback;
}
</script>

<style scoped>
.backtest-chart-panel {
  min-height: calc(100vh - 180px);
}

.backtest-side-section {
  padding: 16px;
}

.backtest-side-section h2 {
  margin: 0 0 12px;
  font-size: 15px;
  font-weight: 720;
  line-height: 1.35;
}

.backtest-side-section__heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.backtest-summary-list {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.3fr);
  gap: 10px 12px;
  margin: 0;
}

.backtest-summary-list dt,
.backtest-summary-list dd {
  min-width: 0;
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
}

.backtest-summary-list dt {
  color: var(--tt-muted);
}

.backtest-summary-list dd {
  overflow-wrap: anywhere;
  font-weight: 650;
  text-align: right;
}

.orders-list {
  display: grid;
  gap: 10px;
}

.orders-list__item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 4px 10px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--tt-line);
}

.orders-list__item:last-child {
  padding-bottom: 0;
  border-bottom: 0;
}

.orders-list__item strong,
.orders-list__item span {
  display: block;
  min-width: 0;
  line-height: 1.45;
}

.orders-list__item span {
  color: var(--tt-muted);
  font-size: 12px;
}

.orders-list__item .n-text {
  grid-column: 2;
}
</style>
