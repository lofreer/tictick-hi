<template>
  <section class="page">
    <header class="page-header">
      <div>
        <h1 class="page-title">{{ t("page.backtests.title") }}</h1>
        <p class="page-subtitle">{{ t("page.backtests.subtitle") }}</p>
      </div>
      <NButton type="primary" @click="router.push({ name: 'backtests-new' })">
        <template #icon>
          <Plus :size="17" />
        </template>
        {{ t("backtests.create") }}
      </NButton>
    </header>

    <section class="surface backtests-panel">
      <LoadingState v-if="loading" />
      <ErrorState v-else-if="error" :title="error" retryable @retry="loadBacktests" />
      <EmptyState v-else-if="tasks.length === 0" :title="t('backtests.noBacktests')" />
      <div v-else class="backtests-table-wrap">
        <table class="backtests-table">
          <thead>
            <tr>
              <th>{{ t("backtests.name") }}</th>
              <th>{{ t("backtests.market") }}</th>
              <th>{{ t("backtests.timeRange") }}</th>
              <th>{{ t("strategy.strategy") }}</th>
              <th>{{ t("backtests.params") }}</th>
              <th>{{ t("backtests.result") }}</th>
              <th>{{ t("backtests.createdAt") }}</th>
              <th>{{ t("research.actions") }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="task in tasks" :key="task.id">
              <td>
                <RouterLink class="backtests-table__name" :to="{ name: 'backtests-detail', params: { id: task.id } }">
                  {{ task.name }}
                </RouterLink>
                <StatusBadge :status="task.status" />
              </td>
              <td>{{ task.exchange }} / {{ task.symbol }} / {{ task.interval }}</td>
              <td>{{ timeRange(task) }}</td>
              <td>{{ task.strategyId }}</td>
              <td>{{ paramSummary(task.strategyParams) }}</td>
              <td>{{ resultSummary(task.resultSummary) }}</td>
              <td>{{ formatDate(task.createdAt) }}</td>
              <td>
                <NButton size="small" @click="router.push({ name: 'backtests-detail', params: { id: task.id } })">
                  {{ t("common.view") }}
                </NButton>
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
import { NButton } from "naive-ui";
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";

import EmptyState from "@/components/common/EmptyState.vue";
import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import StatusBadge from "@/components/common/StatusBadge.vue";
import { backtestsApi } from "@/services/api/backtests";
import type { BacktestTask } from "@/types/app";

const router = useRouter();
const { t } = useI18n();
const tasks = ref<BacktestTask[]>([]);
const loading = ref(false);
const error = ref("");

onMounted(() => {
  void loadBacktests();
});

async function loadBacktests() {
  loading.value = true;
  error.value = "";
  try {
    tasks.value = await backtestsApi.listBacktests();
  } catch (loadError) {
    tasks.value = [];
    error.value = errorMessage(loadError, t("backtests.loadFailed"));
  } finally {
    loading.value = false;
  }
}

function timeRange(task: BacktestTask) {
  return `${formatDate(task.startTime)} - ${formatDate(task.endTime)}`;
}

function paramSummary(params: Record<string, unknown>) {
  const entries = Object.entries(params);
  if (entries.length === 0) {
    return "-";
  }
  return entries
    .slice(0, 4)
    .map(([key, value]) => `${key}: ${String(value)}`)
    .join(", ");
}

function resultSummary(summary: Record<string, unknown>) {
  const totalOrders = summary.totalOrders ?? summary.orderCount;
  if (totalOrders !== undefined) {
    return `${t("backtests.orders")}: ${String(totalOrders)}`;
  }
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
.backtests-panel {
  overflow: hidden;
}

.backtests-table-wrap {
  overflow-x: auto;
}

.backtests-table {
  width: 100%;
  min-width: 1040px;
  border-collapse: collapse;
}

.backtests-table th,
.backtests-table td {
  padding: 12px 14px;
  border-bottom: 1px solid var(--tt-line);
  font-size: 13px;
  line-height: 1.5;
  text-align: left;
  vertical-align: top;
}

.backtests-table th {
  color: var(--tt-muted);
  font-weight: 720;
}

.backtests-table tbody tr:last-child td {
  border-bottom: 0;
}

.backtests-table__name {
  display: inline-flex;
  margin-right: 8px;
  font-weight: 720;
}

.backtests-table__name:hover {
  color: var(--tt-primary);
}
</style>
