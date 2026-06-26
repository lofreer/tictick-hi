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
      <LoadingState v-if="loading" />
      <ErrorState v-else-if="error" :title="error" retryable @retry="loadTasks" />
      <EmptyState v-else-if="tasks.length === 0" :title="t('trading.noTasks')" />
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
            <tr v-for="task in tasks" :key="task.id">
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
import { NButton, NSpace, NTag, useMessage } from "naive-ui";
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";

import EmptyState from "@/components/common/EmptyState.vue";
import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import StatusBadge from "@/components/common/StatusBadge.vue";
import { tradingApi } from "@/services/api/trading";
import type { TradingTask } from "@/types/app";

const router = useRouter();
const message = useMessage();
const { t } = useI18n();
const tasks = ref<TradingTask[]>([]);
const loading = ref(false);
const error = ref("");
const actionID = ref("");

onMounted(() => {
  void loadTasks();
});

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
