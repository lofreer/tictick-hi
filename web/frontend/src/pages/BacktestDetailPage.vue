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
    <div v-else-if="task" class="backtest-detail-workspace">
      <section class="surface kline-chart-frame backtest-chart-panel">
        <div class="kline-chart-frame__viewport backtest-chart-viewport" data-chart-viewport="fixed">
          <ErrorState v-if="candlesError" :title="candlesError" retryable @retry="loadCandles" />
          <LoadingState v-else-if="candlesLoading" />
          <TradingViewChart v-else :data="candles" :markers="chartMarkers" :empty-title="t('backtests.chartEmpty')" />
        </div>
      </section>

      <div class="backtest-detail-lower-grid">
        <section class="surface backtest-side-section backtest-summary-panel">
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

        <section class="surface backtest-side-section backtest-detail-tabs">
          <NTabs type="segment" animated>
            <NTabPane name="parameters" :tab="t('backtests.parameters')">
              <dl class="backtest-summary-list">
                <template v-for="row in paramRows" :key="row.label">
                  <dt>{{ row.label }}</dt>
                  <dd>{{ row.value }}</dd>
                </template>
              </dl>
            </NTabPane>

            <NTabPane name="intents" :tab="t('backtests.intents')">
              <LoadingState v-if="intentsLoading" />
              <ErrorState v-else-if="intentsError" :title="intentsError" retryable @retry="loadIntents" />
              <EmptyState v-else-if="intents.length === 0" :title="t('backtests.noIntents')" />
              <div v-else class="intents-list">
                <div v-for="intent in intents" :key="intent.id" class="intents-list__item">
                  <NTag :type="intent.intentType === 'order' ? 'info' : 'warning'" size="small">
                    {{ intent.intentType }}
                  </NTag>
                  <div>
                    <strong>{{ intent.status }}</strong>
                    <span>{{ intent.policy }} / {{ formatDate(intent.createdAt) }}</span>
                  </div>
                </div>
              </div>
            </NTabPane>

            <NTabPane name="orders" :tab="t('backtests.orders')">
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
            </NTabPane>
          </NTabs>
        </section>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ArrowLeft } from "@lucide/vue";
import { NButton, NTabPane, NTabs, NTag, NText } from "naive-ui";
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
import { appColors } from "@/theme/tokens";
import type { BacktestOrder, BacktestTask, ChartCandle, ChartMarker, StrategyIntent } from "@/types/app";

import "./detailChartLayout.css";
import "./klineChartLayout.css";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const task = ref<BacktestTask | null>(null);
const intents = ref<StrategyIntent[]>([]);
const orders = ref<BacktestOrder[]>([]);
const candles = ref<ChartCandle[]>([]);
const taskLoading = ref(false);
const intentsLoading = ref(false);
const ordersLoading = ref(false);
const candlesLoading = ref(false);
const taskError = ref("");
const intentsError = ref("");
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
    { label: t("backtests.executionInterval"), value: summaryValue("executionInterval") },
    { label: t("backtests.candleSource"), value: summaryValue("candleSource") },
    { label: t("backtests.finalEquity"), value: summaryValue("finalEquity") },
    { label: t("backtests.returnPct"), value: summaryValue("returnPct") },
    { label: t("backtests.totalIntents"), value: summaryValue("totalIntents") },
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
const chartMarkers = computed<ChartMarker[]>(() =>
  orders.value.map((order) => ({
    id: order.id,
    time: Math.floor(new Date(order.occurredAt).getTime() / 1000),
    position: order.side === "buy" ? "belowBar" : "aboveBar",
    shape: order.side === "buy" ? "arrowUp" : "arrowDown",
    color: order.side === "buy" ? appColors.success : appColors.danger,
    text: `${order.side} ${order.quantity}`,
    size: 1.2,
  })),
);

onMounted(() => {
  void loadDetail();
});

async function loadDetail() {
  taskLoading.value = true;
  taskError.value = "";
  try {
    task.value = await backtestsApi.getBacktest(backtestID.value);
    await Promise.all([loadIntents(), loadOrders(), loadCandles()]);
  } catch (loadError) {
    task.value = null;
    taskError.value = errorMessage(loadError, t("backtests.detailLoadFailed"));
  } finally {
    taskLoading.value = false;
  }
}

async function loadIntents() {
  if (!task.value) return;

  intentsLoading.value = true;
  intentsError.value = "";
  try {
    intents.value = await backtestsApi.listIntents(task.value.id);
  } catch (loadError) {
    intents.value = [];
    intentsError.value = errorMessage(loadError, t("backtests.intentsLoadFailed"));
  } finally {
    intentsLoading.value = false;
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
.backtest-detail-workspace {
  display: grid;
  gap: 16px;
  width: 100%;
  min-width: 0;
}

.backtest-detail-lower-grid {
  display: grid;
  grid-template-columns: minmax(220px, 260px) minmax(0, 1fr);
  gap: 16px;
  align-items: start;
  min-width: 0;
}

.backtest-side-section {
  min-width: 0;
  padding: 16px;
}

.backtest-summary-panel {
  align-self: start;
}

.backtest-detail-tabs {
  min-height: 380px;
  align-self: stretch;
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

.intents-list,
.orders-list {
  display: grid;
  gap: 10px;
  max-height: min(560px, 54vh);
  max-height: min(560px, 54dvh);
  overflow: auto;
  padding-right: 4px;
  padding-top: 12px;
}

.intents-list__item,
.orders-list__item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 4px 10px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--tt-line);
}

.intents-list__item:last-child,
.orders-list__item:last-child {
  padding-bottom: 0;
  border-bottom: 0;
}

.intents-list__item strong,
.intents-list__item span,
.orders-list__item strong,
.orders-list__item span {
  display: block;
  min-width: 0;
  line-height: 1.45;
}

.intents-list__item span,
.orders-list__item span {
  color: var(--tt-muted);
  font-size: 12px;
}

.orders-list__item .n-text {
  grid-column: 2;
}

@media (max-width: 980px) {
  .backtest-detail-lower-grid {
    grid-template-columns: 1fr;
  }
}
</style>
