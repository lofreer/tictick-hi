<template>
  <section class="page">
    <header class="page-header">
      <div>
        <h1 class="page-title">{{ t("page.health.title") }}</h1>
        <p class="page-subtitle">{{ t("system.healthSubtitle") }}</p>
      </div>
      <NButton @click="loadHealth">{{ t("common.retry") }}</NButton>
    </header>

    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" :title="error" retryable @retry="loadHealth" />
    <div v-else-if="health" class="health-grid">
      <section class="surface health-summary">
        <h2>{{ t("system.overallStatus") }}</h2>
        <NTag :type="health.status === 'ok' ? 'success' : 'warning'">{{ health.status }}</NTag>
        <dl>
          <dt>{{ t("system.database") }}</dt>
          <dd>{{ health.database }}</dd>
          <dt>{{ t("system.checkedAt") }}</dt>
          <dd>{{ formatDate(health.checkedAt) }}</dd>
        </dl>
      </section>

      <section class="surface health-services">
        <h2>{{ t("system.services") }}</h2>
        <div class="health-service-list">
          <div v-for="service in health.services" :key="service.name" class="health-service">
            <div class="health-service__main">
              <strong>{{ service.name }}</strong>
              <span v-if="service.detail">{{ service.detail }}</span>
              <dl v-if="hasWorkerStats(service)" class="health-service__stats">
                <div>
                  <dt>{{ t("system.pending") }}</dt>
                  <dd>{{ service.pendingCount ?? 0 }}</dd>
                </div>
                <div>
                  <dt>{{ t("system.running") }}</dt>
                  <dd>{{ service.runningCount ?? 0 }}</dd>
                </div>
                <div>
                  <dt>{{ t("system.locked") }}</dt>
                  <dd>{{ service.lockedCount ?? 0 }}</dd>
                </div>
                <div>
                  <dt>{{ t("system.staleLease") }}</dt>
                  <dd>{{ service.staleLeaseCount ?? 0 }}</dd>
                </div>
                <div v-if="service.exchangeBackoffCount !== undefined">
                  <dt>{{ t("system.exchangeBackoff") }}</dt>
                  <dd>{{ service.exchangeBackoffCount }}</dd>
                </div>
                <div v-if="service.nextExchangeAttemptAt !== undefined">
                  <dt>{{ t("system.nextExchangeAttempt") }}</dt>
                  <dd>{{ formatDate(service.nextExchangeAttemptAt) }}</dd>
                </div>
                <div v-if="service.fetchLockSkipCount !== undefined">
                  <dt>{{ t("system.fetchLockSkips") }}</dt>
                  <dd>{{ service.fetchLockSkipCount }}</dd>
                </div>
                <div v-if="service.lastFetchLockSkippedAt !== undefined">
                  <dt>{{ t("system.lastFetchLockSkipped") }}</dt>
                  <dd>{{ formatDate(service.lastFetchLockSkippedAt) }}</dd>
                </div>
                <div>
                  <dt>{{ t("system.lastHeartbeat") }}</dt>
                  <dd>{{ formatDate(service.lastHeartbeatAt) }}</dd>
                </div>
                <div>
                  <dt>{{ t("system.lockedUntil") }}</dt>
                  <dd>{{ formatDate(service.lockedUntil) }}</dd>
                </div>
              </dl>
            </div>
            <NTag :type="serviceType(service.status)">
              {{ service.status }}
            </NTag>
          </div>
        </div>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { NButton, NTag, type TagProps } from "naive-ui";
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";

import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import { systemApi } from "@/services/api/system";
import type { ServiceHealth, SystemHealth } from "@/types/app";

const { t } = useI18n();
const health = ref<SystemHealth | null>(null);
const loading = ref(false);
const error = ref("");

onMounted(() => {
  void loadHealth();
});

async function loadHealth() {
  loading.value = true;
  error.value = "";
  try {
    health.value = await systemApi.health();
  } catch (loadError) {
    health.value = null;
    error.value = errorMessage(loadError, t("system.healthLoadFailed"));
  } finally {
    loading.value = false;
  }
}

function formatDate(value?: string) {
  return value ? new Date(value).toLocaleString() : "-";
}

function hasWorkerStats(service: ServiceHealth) {
  return (
    service.pendingCount !== undefined ||
    service.runningCount !== undefined ||
    service.lockedCount !== undefined ||
    service.staleLeaseCount !== undefined ||
    service.exchangeBackoffCount !== undefined ||
    service.nextExchangeAttemptAt !== undefined ||
    service.fetchLockSkipCount !== undefined ||
    service.lastFetchLockSkippedAt !== undefined ||
    service.lastHeartbeatAt !== undefined ||
    service.lockedUntil !== undefined
  );
}

function serviceType(status: string): TagProps["type"] {
  if (status === "ok") return "success";
  if (status === "failed") return "error";
  if (status === "warning") return "warning";
  return "default";
}

function errorMessage(loadError: unknown, fallback: string) {
  return loadError instanceof Error && loadError.message ? loadError.message : fallback;
}
</script>

<style scoped>
.health-grid {
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  gap: 16px;
  align-items: start;
}

.health-summary,
.health-services {
  padding: 16px;
}

.health-summary h2,
.health-services h2 {
  margin: 0 0 12px;
  font-size: 16px;
  line-height: 1.35;
}

.health-summary dl {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.2fr);
  gap: 10px 12px;
  margin: 16px 0 0;
}

.health-summary dt,
.health-summary dd {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
}

.health-summary dt {
  color: var(--tt-muted);
}

.health-summary dd {
  font-weight: 650;
  text-align: right;
}

.health-service-list {
  display: grid;
  gap: 10px;
}

.health-service {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--tt-line);
}

.health-service__main {
  min-width: 0;
}

.health-service:last-child {
  padding-bottom: 0;
  border-bottom: 0;
}

.health-service strong,
.health-service span {
  display: block;
  line-height: 1.45;
}

.health-service span {
  color: var(--tt-muted);
  font-size: 12px;
}

.health-service__stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px 12px;
  margin: 10px 0 0;
}

.health-service__stats div {
  min-width: 0;
}

.health-service__stats dt,
.health-service__stats dd {
  margin: 0;
  font-size: 12px;
  line-height: 1.45;
}

.health-service__stats dt {
  color: var(--tt-muted);
}

.health-service__stats dd {
  overflow-wrap: anywhere;
  font-weight: 650;
}

@media (max-width: 900px) {
  .health-grid {
    grid-template-columns: 1fr;
  }
}
</style>
