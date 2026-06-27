<template>
  <section class="page">
    <header class="page-header">
      <div>
        <h1 class="page-title">{{ t("page.overview.title") }}</h1>
        <p class="page-subtitle">{{ t("page.overview.subtitle") }}</p>
      </div>
      <NButton @click="loadHealth">{{ t("common.retry") }}</NButton>
    </header>

    <div class="overview-grid">
      <section class="surface overview-card">
        <h2>{{ t("overview.projectLevel") }}</h2>
        <NTag type="warning">scaffold</NTag>
        <p>{{ t("overview.projectLevelDetail") }}</p>
      </section>

      <section class="surface overview-card">
        <h2>{{ t("overview.systemHealth") }}</h2>
        <LoadingState v-if="loading" />
        <ErrorState v-else-if="error" :title="error" retryable @retry="loadHealth" />
        <template v-else-if="health">
          <NTag :type="health.status === 'ok' ? 'success' : 'warning'">{{ health.status }}</NTag>
          <dl>
            <dt>{{ t("system.database") }}</dt>
            <dd>{{ health.database }}</dd>
            <dt>{{ t("system.checkedAt") }}</dt>
            <dd>{{ formatDate(health.checkedAt) }}</dd>
          </dl>
        </template>
      </section>

      <section class="surface overview-card overview-card--wide">
        <h2>{{ t("overview.moduleLevels") }}</h2>
        <div class="overview-module-grid">
          <div v-for="item in moduleLevels" :key="item.name" class="overview-module">
            <span>{{ item.name }}</span>
            <NTag size="small" :type="item.level === 'demo' ? 'info' : 'warning'">
              {{ item.level }}
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

const moduleLevels = [
  { name: "Docker Compose", level: "demo" },
  { name: "API server", level: "scaffold" },
  { name: "Research", level: "scaffold" },
  { name: "Backtest", level: "scaffold" },
  { name: "Trading", level: "scaffold" },
  { name: "Notification", level: "scaffold" },
] as const;

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
.overview-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.overview-card {
  padding: 16px;
}

.overview-card--wide {
  grid-column: 1 / -1;
}

.overview-card h2 {
  margin: 0 0 12px;
  font-size: 16px;
  line-height: 1.35;
}

.overview-card p {
  margin: 12px 0 0;
  color: var(--tt-muted);
  font-size: 13px;
  line-height: 1.6;
}

.overview-card dl {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.4fr);
  gap: 10px 12px;
  margin: 16px 0 0;
}

.overview-card dt,
.overview-card dd {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
}

.overview-card dt {
  color: var(--tt-muted);
}

.overview-card dd {
  font-weight: 650;
  text-align: right;
}

.overview-module-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.overview-module {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 0;
  border-bottom: 1px solid var(--tt-line);
}

.overview-module span {
  font-weight: 650;
}

@media (max-width: 900px) {
  .overview-grid,
  .overview-module-grid {
    grid-template-columns: 1fr;
  }
}
</style>
