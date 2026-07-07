<template>
  <section class="page">
    <header class="page-header">
      <div>
        <h1 class="page-title">{{ t("page.overview.title") }}</h1>
        <p class="page-subtitle">{{ t("page.overview.subtitle") }}</p>
      </div>
      <NButton :loading="loading" @click="loadOverview">
        <template #icon>
          <RefreshCw :size="16" />
        </template>
        {{ t("common.retry") }}
      </NButton>
    </header>

    <ErrorState v-if="error" :title="error" retryable @retry="loadOverview" />
    <LoadingState v-else-if="loading && !hasLoaded" />

    <template v-else>
      <section class="surface overview-banner">
        <div>
          <h2>{{ t("overview.projectLevel") }}</h2>
          <p>{{ t("overview.projectLevelDetail") }}</p>
        </div>
        <NTag type="warning">scaffold</NTag>
      </section>

      <div class="overview-metrics">
        <section v-for="card in summaryCards" :key="card.key" class="surface overview-metric">
          <span>{{ card.label }}</span>
          <strong>{{ card.value }}</strong>
          <p>{{ card.detail }}</p>
        </section>
      </div>

      <div class="overview-grid">
        <section class="surface overview-panel">
          <div class="overview-panel__header">
            <h2>{{ t("overview.systemHealth") }}</h2>
            <NTag :type="healthTagType">{{ health?.status ?? "-" }}</NTag>
          </div>
          <dl class="overview-health-summary">
            <dt>{{ t("system.database") }}</dt>
            <dd>{{ health?.database ?? "-" }}</dd>
            <dt>{{ t("system.checkedAt") }}</dt>
            <dd>{{ formatDate(health?.checkedAt) }}</dd>
          </dl>
          <div class="overview-service-list">
            <div v-for="service in services" :key="service.name" class="overview-service">
              <div>
                <strong>{{ service.name }}</strong>
                <span>{{ service.detail || serviceSummary(service) }}</span>
              </div>
              <NTag :type="service.status === 'ok' ? 'success' : 'warning'" size="small">
                {{ service.status }}
              </NTag>
            </div>
          </div>
        </section>

        <section class="surface overview-panel">
          <div class="overview-panel__header">
            <h2>{{ t("overview.alerts") }}</h2>
            <NTag :type="alerts.length > 0 ? 'warning' : 'success'">
              {{ alerts.length }}
            </NTag>
          </div>
          <EmptyState v-if="alerts.length === 0" :title="t('overview.noAlerts')" />
          <div v-else class="overview-alert-list">
            <RouterLink v-for="alert in alerts" :key="alert.key" class="overview-alert" :to="alert.to">
              <NTag :type="alert.type" size="small">{{ alert.label }}</NTag>
              <div>
                <strong>{{ alert.title }}</strong>
                <span>{{ alert.detail }}</span>
              </div>
            </RouterLink>
          </div>
        </section>

        <section class="surface overview-panel overview-panel--wide">
          <div class="overview-panel__header">
            <h2>{{ t("overview.recentActivity") }}</h2>
            <div class="overview-panel__actions">
              <NTag v-if="factsError" type="warning" size="small" :title="factsError">{{ t("overview.degraded") }}</NTag>
              <NTag size="small">{{ recentActivities.length }}</NTag>
            </div>
          </div>
          <EmptyState v-if="recentActivities.length === 0" :title="t('common.noData')" />
          <div v-else class="overview-activity-list">
            <RouterLink
              v-for="activity in recentActivities"
              :key="activity.key"
              class="overview-activity"
              :to="activity.to"
            >
              <div>
                <strong>{{ activity.title }}</strong>
                <span>{{ activity.detail }}</span>
              </div>
              <div class="overview-activity__meta">
                <NTag :type="activity.statusType" size="small">{{ activity.status }}</NTag>
                <time>{{ formatDate(activity.at) }}</time>
              </div>
            </RouterLink>
          </div>
        </section>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { RefreshCw } from "@lucide/vue";
import { NButton, NTag } from "naive-ui";

import EmptyState from "@/components/common/EmptyState.vue";
import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import { useOverviewWorkspace } from "@/composables/useOverviewWorkspace";

const {
  alerts,
  error,
  factsError,
  formatDate,
  hasLoaded,
  health,
  healthTagType,
  loadOverview,
  loading,
  recentActivities,
  serviceSummary,
  services,
  summaryCards,
  t,
} = useOverviewWorkspace();
</script>

<style scoped>
.overview-banner,
.overview-panel__header,
.overview-service,
.overview-alert,
.overview-activity {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
}

.overview-banner {
  padding: 16px;
}

.overview-banner h2,
.overview-panel h2 {
  margin: 0;
  font-size: 16px;
  line-height: 1.35;
  font-weight: 760;
}

.overview-banner p {
  margin: 6px 0 0;
  color: var(--tt-muted);
  font-size: 13px;
  line-height: 1.6;
}

.overview-metrics {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
}

.overview-metric,
.overview-panel {
  padding: 16px;
}

.overview-metric {
  min-width: 0;
}

.overview-metric span,
.overview-service span,
.overview-alert span,
.overview-activity span,
.overview-activity time {
  display: block;
  color: var(--tt-muted);
  font-size: 12px;
  line-height: 1.5;
}

.overview-metric strong {
  display: block;
  margin-top: 8px;
  font-size: 26px;
  line-height: 1.1;
}

.overview-metric p {
  margin: 8px 0 0;
  color: var(--tt-muted);
  font-size: 12px;
  line-height: 1.5;
}

.overview-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.overview-panel {
  min-width: 0;
}

.overview-panel--wide {
  grid-column: 1 / -1;
}

.overview-panel__actions {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 8px;
}

.overview-health-summary {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.4fr);
  gap: 10px 12px;
  margin: 16px 0;
}

.overview-health-summary dt,
.overview-health-summary dd {
  min-width: 0;
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
}

.overview-health-summary dt {
  color: var(--tt-muted);
}

.overview-health-summary dd {
  overflow-wrap: anywhere;
  font-weight: 650;
  text-align: right;
}

.overview-service-list,
.overview-alert-list,
.overview-activity-list {
  display: grid;
  gap: 10px;
}

.overview-service,
.overview-alert,
.overview-activity {
  min-width: 0;
  padding: 10px 0;
  border-bottom: 1px solid var(--tt-line);
}

.overview-service:last-child,
.overview-alert:last-child,
.overview-activity:last-child {
  border-bottom: 0;
}

.overview-service div,
.overview-alert div,
.overview-activity div {
  min-width: 0;
}

.overview-service strong,
.overview-alert strong,
.overview-activity strong {
  display: block;
  overflow: hidden;
  font-size: 13px;
  line-height: 1.45;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.overview-alert:hover,
.overview-activity:hover {
  color: var(--tt-primary);
}

.overview-activity__meta {
  display: grid;
  justify-items: end;
  gap: 6px;
  flex: 0 0 auto;
}

@media (max-width: 1100px) {
  .overview-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 760px) {
  .overview-banner,
  .overview-grid,
  .overview-metrics {
    grid-template-columns: 1fr;
  }

  .overview-banner,
  .overview-service,
  .overview-alert,
  .overview-activity {
    align-items: flex-start;
    flex-direction: column;
  }

  .overview-activity__meta {
    justify-items: start;
  }
}
</style>
