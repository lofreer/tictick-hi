<template>
  <section class="page">
    <header class="page-header">
      <div>
        <h1 class="page-title">{{ task?.name ?? t("page.tradingDetail.title") }}</h1>
        <p class="page-subtitle">{{ subtitle }}</p>
      </div>
      <NButton @click="router.push({ name: 'trading' })">
        <template #icon>
          <ArrowLeft :size="17" />
        </template>
        {{ t("trading.backToList") }}
      </NButton>
    </header>

    <LoadingState v-if="taskLoading" />
    <ErrorState v-else-if="taskError" :title="taskError" retryable @retry="loadDetail" />
    <div v-else-if="task" class="trading-detail-workspace">
      <section class="surface kline-chart-frame trading-detail-chart">
        <div class="kline-chart-frame__viewport trading-detail-chart-viewport" data-chart-viewport="fixed">
          <ErrorState v-if="candlesError" :title="candlesError" retryable @retry="loadCandles" />
          <LoadingState v-else-if="candlesLoading" />
          <TradingViewChart v-else :data="candles" :markers="chartMarkers" :empty-title="t('trading.chartEmpty')" />
        </div>
      </section>

      <div class="trading-detail-lower-grid">
        <section class="surface trading-detail-section trading-detail-summary">
          <div class="trading-detail-section__heading">
            <h2>{{ t("trading.summary") }}</h2>
            <StatusBadge :status="task.status" />
          </div>
          <dl class="trading-summary-list">
            <template v-for="row in summaryRows" :key="row.label">
              <dt>{{ row.label }}</dt>
              <dd>{{ row.value }}</dd>
            </template>
          </dl>
        </section>

        <section class="surface trading-detail-section trading-detail-tabs">
          <NTabs type="segment" animated>
            <NTabPane name="positions" :tab="t('trading.positions')">
              <LoadingState v-if="positionsLoading" />
              <ErrorState v-else-if="positionsError" :title="positionsError" retryable @retry="loadPositions" />
              <EmptyState v-else-if="positions.length === 0" :title="t('trading.noPositions')" />
              <div v-else class="detail-list">
                <div v-for="position in positions" :key="position.symbol" class="detail-list__item">
                  <NTag size="small">{{ position.symbol }}</NTag>
                  <strong>{{ position.quantity }}</strong>
                  <span>{{ position.averagePrice }} / {{ formatDate(position.updatedAt) }}</span>
                </div>
              </div>
            </NTabPane>
            <NTabPane name="intents" :tab="t('trading.intents')">
              <LoadingState v-if="intentsLoading" />
              <ErrorState v-else-if="intentsError" :title="intentsError" retryable @retry="loadIntents" />
              <EmptyState v-else-if="intents.length === 0" :title="t('trading.noIntents')" />
              <div v-else class="detail-list">
                <div v-for="intent in intents" :key="intent.id" class="detail-list__item">
                  <NTag size="small">{{ intent.intentType }}</NTag>
                  <strong>{{ intent.status }}</strong>
                  <span>{{ intent.policy }} / {{ formatDate(intent.createdAt) }}</span>
                </div>
              </div>
            </NTabPane>
            <NTabPane name="orders" :tab="t('trading.orders')">
              <LoadingState v-if="ordersLoading" />
              <ErrorState v-else-if="ordersError" :title="ordersError" retryable @retry="loadOrders" />
              <EmptyState v-else-if="orders.length === 0" :title="t('trading.noOrders')" />
              <div v-else class="detail-list">
                <div v-for="order in orders" :key="order.id" class="detail-list__item">
                  <NTag :type="order.side === 'buy' ? 'success' : 'error'" size="small">{{ order.side }}</NTag>
                  <strong>{{ order.price }} / {{ order.quantity }}</strong>
                  <span>{{ order.status }} / {{ formatDate(order.createdAt) }}</span>
                </div>
              </div>
            </NTabPane>
            <NTabPane name="executions" :tab="t('trading.executions')">
              <LoadingState v-if="executionsLoading" />
              <ErrorState v-else-if="executionsError" :title="executionsError" retryable @retry="loadExecutions" />
              <EmptyState v-else-if="executions.length === 0" :title="t('trading.noExecutions')" />
              <div v-else class="detail-list">
                <div v-for="execution in executions" :key="execution.id" class="detail-list__item">
                  <NTag :type="execution.side === 'buy' ? 'success' : 'error'" size="small">{{ execution.side }}</NTag>
                  <strong>{{ execution.price }} / {{ execution.quantity }}</strong>
                  <span>{{ execution.status }} / {{ formatDate(execution.executedAt) }}</span>
                </div>
              </div>
            </NTabPane>
            <NTabPane name="notifications" :tab="t('trading.notifications')">
              <LoadingState v-if="notificationsLoading" />
              <ErrorState
                v-else-if="notificationsError"
                :title="notificationsError"
                retryable
                @retry="loadNotifications"
              />
              <EmptyState v-else-if="notifications.length === 0" :title="t('trading.noNotifications')" />
              <div v-else class="detail-list">
                <div v-for="notification in notifications" :key="notification.id" class="detail-list__item">
                  <NTag size="small">{{ notification.channel }}</NTag>
                  <strong>{{ notification.title }}</strong>
                  <span>
                    {{ notification.status }} / {{ notification.provider }} /
                    {{ notification.attemptCount }} / {{ notification.maxAttempts }}
                  </span>
                  <span>{{ notification.error || (notification.nextAttemptAt ? formatDate(notification.nextAttemptAt) : formatDate(notification.createdAt)) }}</span>
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
import { NButton, NTabPane, NTabs, NTag } from "naive-ui";
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";

import TradingViewChart from "@/components/chart/TradingViewChart.vue";
import EmptyState from "@/components/common/EmptyState.vue";
import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import StatusBadge from "@/components/common/StatusBadge.vue";
import { dataApi } from "@/services/api/data";
import { tradingApi } from "@/services/api/trading";
import { appColors } from "@/theme/tokens";
import type {
  ChartCandle,
  ChartMarker,
  Execution,
  Notification as TradingNotification,
  Order,
  Position,
  StrategyIntent,
  TradingTask,
} from "@/types/app";

import "./detailChartLayout.css";
import "./klineChartLayout.css";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const task = ref<TradingTask | null>(null);
const intents = ref<StrategyIntent[]>([]);
const orders = ref<Order[]>([]);
const executions = ref<Execution[]>([]);
const positions = ref<Position[]>([]);
const notifications = ref<TradingNotification[]>([]);
const candles = ref<ChartCandle[]>([]);
const taskLoading = ref(false);
const intentsLoading = ref(false);
const ordersLoading = ref(false);
const executionsLoading = ref(false);
const positionsLoading = ref(false);
const notificationsLoading = ref(false);
const candlesLoading = ref(false);
const taskError = ref("");
const intentsError = ref("");
const ordersError = ref("");
const executionsError = ref("");
const positionsError = ref("");
const notificationsError = ref("");
const candlesError = ref("");

const taskID = computed(() => String(route.params.id ?? ""));
const subtitle = computed(() => {
  if (!task.value) return "";
  return `${t(`strategy.${task.value.type}`)} / ${task.value.exchange} / ${task.value.symbol} / ${task.value.interval}`;
});
const summaryRows = computed(() => {
  if (!task.value) return [];
  return [
    { label: t("trading.type"), value: t(`strategy.${task.value.type}`) },
    { label: t("research.exchange"), value: task.value.exchange },
    { label: t("trading.accountId"), value: task.value.accountId },
    { label: t("research.symbol"), value: task.value.symbol },
    { label: t("research.interval"), value: task.value.interval },
    { label: t("strategy.strategy"), value: task.value.strategyId },
    { label: t("trading.orderIntent"), value: String(task.value.intentPolicy.orderIntent ?? "-") },
    { label: t("trading.notificationChannel"), value: String(task.value.intentPolicy.notificationChannel ?? "-") },
    { label: t("trading.worker"), value: task.value.lockedBy || "-" },
    { label: t("trading.heartbeatAt"), value: formatDate(task.value.heartbeatAt) },
    { label: t("common.error"), value: task.value.lastError || "-" },
  ];
});
const chartMarkers = computed<ChartMarker[]>(() =>
  executions.value.map((execution) => ({
    id: execution.id,
    time: Math.floor(new Date(execution.executedAt).getTime() / 1000),
    position: execution.side === "buy" ? "belowBar" : "aboveBar",
    shape: execution.side === "buy" ? "arrowUp" : "arrowDown",
    color: execution.side === "buy" ? appColors.success : appColors.danger,
    text: `${execution.side} ${execution.quantity}`,
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
    task.value = await tradingApi.getTask(taskID.value);
    await Promise.all([
      loadCandles(),
      loadIntents(),
      loadOrders(),
      loadExecutions(),
      loadPositions(),
      loadNotifications(),
    ]);
  } catch (loadError) {
    task.value = null;
    taskError.value = errorMessage(loadError, t("trading.detailLoadFailed"));
  } finally {
    taskLoading.value = false;
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
      limit: 1000,
    });
  } catch (loadError) {
    candles.value = [];
    candlesError.value = errorMessage(loadError, t("research.candlesLoadFailed"));
  } finally {
    candlesLoading.value = false;
  }
}

async function loadIntents() {
  if (!task.value) return;
  intentsLoading.value = true;
  intentsError.value = "";
  try {
    intents.value = await tradingApi.listIntents(task.value.id);
  } catch (loadError) {
    intents.value = [];
    intentsError.value = errorMessage(loadError, t("trading.intentsLoadFailed"));
  } finally {
    intentsLoading.value = false;
  }
}

async function loadOrders() {
  if (!task.value) return;
  ordersLoading.value = true;
  ordersError.value = "";
  try {
    orders.value = await tradingApi.listOrders(task.value.id);
  } catch (loadError) {
    orders.value = [];
    ordersError.value = errorMessage(loadError, t("trading.ordersLoadFailed"));
  } finally {
    ordersLoading.value = false;
  }
}

async function loadExecutions() {
  if (!task.value) return;
  executionsLoading.value = true;
  executionsError.value = "";
  try {
    executions.value = await tradingApi.listExecutions(task.value.id);
  } catch (loadError) {
    executions.value = [];
    executionsError.value = errorMessage(loadError, t("trading.executionsLoadFailed"));
  } finally {
    executionsLoading.value = false;
  }
}

async function loadPositions() {
  if (!task.value) return;
  positionsLoading.value = true;
  positionsError.value = "";
  try {
    positions.value = await tradingApi.listPositions(task.value.id);
  } catch (loadError) {
    positions.value = [];
    positionsError.value = errorMessage(loadError, t("trading.positionsLoadFailed"));
  } finally {
    positionsLoading.value = false;
  }
}

async function loadNotifications() {
  if (!task.value) return;
  notificationsLoading.value = true;
  notificationsError.value = "";
  try {
    notifications.value = await tradingApi.listNotifications(task.value.id);
  } catch (loadError) {
    notifications.value = [];
    notificationsError.value = errorMessage(loadError, t("trading.notificationsLoadFailed"));
  } finally {
    notificationsLoading.value = false;
  }
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
.trading-detail-workspace {
  display: grid;
  gap: 16px;
  width: 100%;
  min-width: 0;
}

.trading-detail-lower-grid {
  display: grid;
  grid-template-columns: minmax(240px, 300px) minmax(0, 1fr);
  gap: 16px;
  align-items: stretch;
  min-width: 0;
}

.trading-detail-section {
  min-width: 0;
  padding: 16px;
}

.trading-detail-summary {
  align-self: start;
}

.trading-detail-tabs {
  min-height: 320px;
}

.trading-detail-section h2 {
  margin: 0 0 12px;
  font-size: 15px;
  font-weight: 720;
  line-height: 1.35;
}

.trading-detail-section__heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.trading-summary-list {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.3fr);
  gap: 10px 12px;
  margin: 0;
}

.trading-summary-list dt,
.trading-summary-list dd {
  min-width: 0;
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
}

.trading-summary-list dt {
  color: var(--tt-muted);
}

.trading-summary-list dd {
  overflow-wrap: anywhere;
  font-weight: 650;
  text-align: right;
}

.detail-list {
  display: grid;
  gap: 10px;
  max-height: min(520px, 54vh);
  max-height: min(520px, 54dvh);
  overflow: auto;
  padding-right: 4px;
  padding-top: 12px;
}

.detail-list__item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 4px 10px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--tt-line);
}

.detail-list__item:last-child {
  padding-bottom: 0;
  border-bottom: 0;
}

.detail-list__item strong,
.detail-list__item span {
  display: block;
  min-width: 0;
  line-height: 1.45;
}

.detail-list__item span {
  grid-column: 2;
  color: var(--tt-muted);
  font-size: 12px;
}

@media (max-width: 980px) {
  .trading-detail-lower-grid {
    grid-template-columns: 1fr;
  }
}
</style>
