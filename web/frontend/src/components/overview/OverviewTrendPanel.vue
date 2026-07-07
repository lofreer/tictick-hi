<template>
  <section class="surface overview-trend-panel">
    <div class="overview-trend-panel__header">
      <h2>{{ t("overview.trends") }}</h2>
      <div class="overview-trend-panel__actions">
        <NButton v-if="error" quaternary size="small" :loading="loading" @click="$emit('retry')">
          {{ t("common.retry") }}
        </NButton>
        <NTag v-if="error" type="warning" size="small" :title="error">{{ t("overview.degraded") }}</NTag>
        <NTag v-else size="small">{{ t("overview.trendWindow.7d") }}</NTag>
      </div>
    </div>

    <EmptyState v-if="!hasTrendData" :title="error ? t('overview.trendsLoadFailed') : t('common.noData')" />
    <div v-else class="overview-trend">
      <div class="overview-trend__summary">
        <span>{{ t("overview.strategyIntents") }} {{ totals.strategyIntents }}</span>
        <span>{{ t("overview.orders") }} {{ totals.orders }}</span>
        <span>{{ t("overview.notifications") }} {{ totals.notifications }}</span>
        <span>{{ t("overview.failures") }} {{ totals.failures }}</span>
      </div>
      <div class="overview-trend__bars" :aria-label="t('overview.trends')">
        <div
          v-for="point in points"
          :key="point.bucketStart"
          class="overview-trend__bucket"
          :style="{ '--total-pct': `${point.totalPct}%`, '--failure-pct': `${point.failurePct}%` }"
          :title="`${point.label} / ${point.total} / ${point.failures}`"
        >
          <div class="overview-trend__bar">
            <span class="overview-trend__bar-total" />
            <span v-if="point.failures > 0" class="overview-trend__bar-failure" />
          </div>
          <span>{{ point.label }}</span>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { NButton, NTag } from "naive-ui";
import { useI18n } from "vue-i18n";

import EmptyState from "@/components/common/EmptyState.vue";
import type { OverviewTrendPoint, OverviewTrendTotals } from "@/composables/useOverviewTrends";

defineProps<{
  error: string;
  hasTrendData: boolean;
  loading: boolean;
  points: OverviewTrendPoint[];
  totals: OverviewTrendTotals;
}>();

defineEmits<{
  (event: "retry"): void;
}>();

const { t } = useI18n();
</script>

<style scoped>
.overview-trend-panel {
  display: grid;
  min-width: 0;
  padding: 16px;
  gap: 14px;
}

.overview-trend-panel__header,
.overview-trend-panel__actions,
.overview-trend__summary {
  display: flex;
  align-items: center;
  gap: 8px;
}

.overview-trend-panel__header {
  justify-content: space-between;
}

.overview-trend-panel h2 {
  margin: 0;
  font-size: 16px;
  line-height: 1.35;
  font-weight: 760;
}

.overview-trend-panel__actions,
.overview-trend__summary {
  flex-wrap: wrap;
  justify-content: flex-end;
}

.overview-trend {
  display: grid;
  gap: 14px;
}

.overview-trend__summary span,
.overview-trend__bucket > span {
  color: var(--tt-muted);
  font-size: 12px;
  line-height: 1.45;
}

.overview-trend__bars {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  align-items: end;
  gap: 10px;
  min-height: 132px;
}

.overview-trend__bucket {
  display: grid;
  min-width: 0;
  gap: 8px;
  text-align: center;
}

.overview-trend__bar {
  position: relative;
  height: 104px;
  overflow: hidden;
  border: 1px solid var(--tt-line);
  border-radius: 6px;
  background: color-mix(in srgb, var(--tt-surface-raised) 82%, transparent);
}

.overview-trend__bar-total,
.overview-trend__bar-failure {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  min-height: 4px;
}

.overview-trend__bar-total {
  height: var(--total-pct);
  background: color-mix(in srgb, var(--tt-primary) 62%, transparent);
}

.overview-trend__bar-failure {
  height: var(--failure-pct);
  background: color-mix(in srgb, #d03050 78%, transparent);
}

@media (max-width: 760px) {
  .overview-trend-panel__header {
    align-items: flex-start;
    flex-direction: column;
  }

  .overview-trend-panel__actions,
  .overview-trend__summary {
    justify-content: flex-start;
  }

  .overview-trend__bars {
    gap: 6px;
  }
}
</style>
