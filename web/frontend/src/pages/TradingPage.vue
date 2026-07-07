<template>
  <section class="page">
    <header class="page-header">
      <div>
        <h1 class="page-title">{{ t("page.trading.title") }}</h1>
        <p class="page-subtitle">{{ t("page.trading.subtitle") }}</p>
      </div>
      <NButton type="primary" @click="router.push({ name: 'trading-new' })">
        <template #icon>
          <Plus :size="17" />
        </template>
        {{ t("trading.create") }}
      </NButton>
    </header>

    <section class="surface trading-panel">
      <div class="trading-panel__header">
        <h2>{{ t("trading.tasks") }}</h2>
        <NRadioGroup :value="statusFilter" size="small" :aria-label="t('trading.statusFilter')" @update:value="setStatusFilter">
          <NRadioButton v-for="option in statusFilterOptions" :key="option.value" :value="option.value">
            {{ option.label }}
          </NRadioButton>
        </NRadioGroup>
      </div>
      <LoadingState v-if="loading" />
      <ErrorState v-else-if="error" :title="error" retryable @retry="loadTasks" />
      <EmptyState v-else-if="tasks.length === 0" :title="t('trading.noTasks')" />
      <EmptyState v-else-if="filteredTasks.length === 0" :title="t('trading.noTasksForFilter')" />
      <div v-else class="trading-table-wrap">
        <table class="trading-table">
          <thead>
            <tr>
              <th>{{ t("trading.name") }}</th>
              <th>{{ t("trading.type") }}</th>
              <th>{{ t("trading.market") }}</th>
              <th>{{ t("trading.accountId") }}</th>
              <th>{{ t("strategy.strategy") }}</th>
              <th>{{ t("backtests.params") }}</th>
              <th>{{ t("trading.recent") }}</th>
              <th>{{ t("backtests.createdAt") }}</th>
              <th>{{ t("research.actions") }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="task in filteredTasks" :key="task.id">
              <td>
                <RouterLink class="trading-table__name" :to="{ name: 'trading-detail', params: { id: task.id } }">
                  {{ task.name }}
                </RouterLink>
                <StatusBadge :status="task.status" />
              </td>
              <td>
                <NTag :type="task.type === 'live' ? 'warning' : 'info'" size="small">
                  {{ t(`strategy.${task.type}`) }}
                </NTag>
              </td>
              <td>{{ task.exchange }} / {{ task.symbol }} / {{ task.interval }}</td>
              <td>{{ task.accountId }}</td>
              <td>{{ task.strategyId }}</td>
              <td>{{ paramSummary(task.strategyParams) }}</td>
              <td>{{ recentSummary(task) }}</td>
              <td>{{ formatDate(task.createdAt) }}</td>
              <td>
                <NSpace :size="6" wrap>
                  <NButton size="small" @click="router.push({ name: 'trading-detail', params: { id: task.id } })">
                    {{ t("common.view") }}
                  </NButton>
                  <NButton
                    v-if="task.status !== 'running'"
                    size="small"
                    type="primary"
                    :loading="actionID === task.id"
                    @click="startTask(task)"
                  >
                    {{ t("trading.start") }}
                  </NButton>
                  <NButton
                    v-else
                    size="small"
                    :loading="actionID === task.id"
                    @click="pauseTask(task)"
                  >
                    {{ t("trading.pause") }}
                  </NButton>
                </NSpace>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </section>
</template>

<script setup lang="ts">
import { Plus } from "@lucide/vue";
import { NButton, NRadioButton, NRadioGroup, NSpace, NTag, useMessage } from "naive-ui";
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";

import EmptyState from "@/components/common/EmptyState.vue";
import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import StatusBadge from "@/components/common/StatusBadge.vue";
import { tradingApi } from "@/services/api/trading";
import type { TradingTask } from "@/types/app";
import { taskMatchesStatusFilter, taskStatusFilterFromQuery, taskStatusQueryValue, type TaskStatusFilter } from "./taskStatusFilters";

const router = useRouter();
const route = useRoute();
const message = useMessage();
const { t } = useI18n();
const tasks = ref<TradingTask[]>([]);
const loading = ref(false);
const error = ref("");
const actionID = ref("");
const statusFilter = ref<TaskStatusFilter>(taskStatusFilterFromQuery(route.query.status));
const statusFilterOptions = computed(() => [
  { label: t("trading.status.all"), value: "all" },
  { label: t("status.running"), value: "running" },
  { label: t("status.failed"), value: "failed" },
  { label: t("status.succeeded"), value: "succeeded" },
]);
const filteredTasks = computed(() => tasks.value.filter((task) => taskMatchesStatusFilter(task, statusFilter.value)));

onMounted(() => {
  void loadTasks();
});

watch(
  () => route.query.status,
  (value) => {
    statusFilter.value = taskStatusFilterFromQuery(value);
  },
);

async function loadTasks() {
  loading.value = true;
  error.value = "";
  try {
    tasks.value = await tradingApi.listTasks();
  } catch (loadError) {
    tasks.value = [];
    error.value = errorMessage(loadError, t("trading.loadFailed"));
  } finally {
    loading.value = false;
  }
}

async function startTask(task: TradingTask) {
  await runAction(task.id, async () => {
    await tradingApi.startTask(task.id);
    message.success(t("trading.updated"));
    await loadTasks();
  });
}

async function pauseTask(task: TradingTask) {
  await runAction(task.id, async () => {
    await tradingApi.pauseTask(task.id);
    message.success(t("trading.updated"));
    await loadTasks();
  });
}

async function runAction(id: string, action: () => Promise<void>) {
  actionID.value = id;
  try {
    await action();
  } catch (loadError) {
    message.error(errorMessage(loadError, t("trading.updateFailed")));
  } finally {
    actionID.value = "";
  }
}

function paramSummary(params: Record<string, unknown>) {
  const entries = Object.entries(params);
  if (entries.length === 0) return "-";
  return entries
    .slice(0, 4)
    .map(([key, value]) => `${key}: ${String(value)}`)
    .join(", ");
}

function recentSummary(_task: TradingTask) {
  return "-";
}

async function setStatusFilter(value: string) {
  statusFilter.value = taskStatusFilterFromQuery(value);
  const nextQuery = { ...route.query };
  const status = taskStatusQueryValue(statusFilter.value);
  if (status) nextQuery.status = status;
  else delete nextQuery.status;
  await router.replace({ query: nextQuery });
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
.trading-panel {
  overflow: hidden;
}

.trading-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 14px;
}

.trading-panel__header h2 {
  margin: 0;
  font-size: 16px;
  line-height: 1.35;
  font-weight: 760;
}

.trading-panel__header :deep(.n-radio-button) {
  min-width: 74px;
  text-align: center;
}

.trading-table-wrap {
  overflow-x: auto;
}

.trading-table {
  width: 100%;
  min-width: 1120px;
  border-collapse: collapse;
}

.trading-table th,
.trading-table td {
  padding: 12px 14px;
  border-bottom: 1px solid var(--tt-line);
  font-size: 13px;
  line-height: 1.5;
  text-align: left;
  vertical-align: top;
}

.trading-table th {
  color: var(--tt-muted);
  font-weight: 720;
}

.trading-table tbody tr:last-child td {
  border-bottom: 0;
}

.trading-table__name {
  display: inline-flex;
  margin-right: 8px;
  font-weight: 720;
}

.trading-table__name:hover {
  color: var(--tt-primary);
}
</style>
