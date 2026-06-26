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
            <div>
              <strong>{{ service.name }}</strong>
              <span v-if="service.detail">{{ service.detail }}</span>
            </div>
            <NTag :type="service.status === 'ok' ? 'success' : service.status === 'failed' ? 'error' : 'default'">
              {{ service.status }}
            </NTag>
          </div>
        </div>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { NButton, NTag } from "naive-ui";
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";

import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import { systemApi } from "@/services/api/system";
import type { SystemHealth } from "@/types/app";

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
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--tt-line);
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

@media (max-width: 900px) {
  .health-grid {
    grid-template-columns: 1fr;
  }
}
</style>
