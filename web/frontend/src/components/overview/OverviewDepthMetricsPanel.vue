<template>
  <section class="surface overview-depth-panel">
    <div class="overview-depth-panel__header">
      <h2>{{ t("overview.depthMetrics") }}</h2>
      <NTag size="small">{{ items.length }}</NTag>
    </div>

    <div class="overview-depth-panel__grid">
      <RouterLink
        v-for="item in items"
        :key="item.key"
        class="overview-depth-metric"
        :to="item.to"
        :aria-label="t('overview.openMetric', { target: item.label })"
      >
        <div class="overview-depth-metric__header">
          <span>{{ item.label }}</span>
          <NTag :type="item.statusType" size="small">{{ item.statusLabel }}</NTag>
        </div>
        <strong>{{ item.value }}</strong>
        <p>{{ item.detail }}</p>
        <ChevronRight class="overview-depth-metric__icon" :size="16" aria-hidden="true" />
      </RouterLink>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ChevronRight } from "@lucide/vue";
import { NTag } from "naive-ui";
import { useI18n } from "vue-i18n";

import type { OverviewDepthMetric } from "@/composables/overviewDepthMetrics";

defineProps<{
  items: OverviewDepthMetric[];
}>();

const { t } = useI18n();
</script>

<style scoped>
.overview-depth-panel {
  display: grid;
  min-width: 0;
  padding: 18px;
  gap: 16px;
  background:
    linear-gradient(180deg, var(--tt-surface) 0, color-mix(in srgb, var(--tt-surface-raised) 58%, var(--tt-surface)) 100%),
    var(--tt-surface);
}

.overview-depth-panel__header,
.overview-depth-metric__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.overview-depth-panel h2 {
  margin: 0;
  font-size: 16px;
  line-height: 1.35;
  font-weight: 760;
}

.overview-depth-panel__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0;
}

.overview-depth-metric {
  position: relative;
  min-width: 0;
  min-height: 128px;
  padding: 14px 36px 12px 0;
  border-top: 1px solid var(--tt-line);
  color: inherit;
  text-decoration: none;
}

.overview-depth-metric + .overview-depth-metric {
  padding-left: 14px;
  border-left: 1px solid var(--tt-line);
}

.overview-depth-metric:hover,
.overview-depth-metric:focus-visible {
  color: var(--tt-primary-strong);
}

.overview-depth-metric:focus-visible {
  outline: 2px solid var(--tt-primary);
  outline-offset: 2px;
}

.overview-depth-metric__header {
  min-width: 0;
}

.overview-depth-metric__header > span {
  overflow: hidden;
  color: var(--tt-muted);
  font-size: 12px;
  line-height: 1.45;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.overview-depth-metric strong {
  display: block;
  margin-top: 10px;
  font-size: 24px;
  line-height: 1.1;
}

.overview-depth-metric p {
  margin: 10px 0 0;
  color: var(--tt-muted);
  font-size: 12px;
  line-height: 1.5;
}

.overview-depth-metric__icon {
  position: absolute;
  top: 14px;
  right: 12px;
  color: var(--tt-muted);
}

.overview-depth-metric:hover .overview-depth-metric__icon,
.overview-depth-metric:focus-visible .overview-depth-metric__icon {
  color: var(--tt-primary);
}

@media (max-width: 1100px) {
  .overview-depth-panel__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .overview-depth-metric:nth-child(2n + 1) {
    padding-left: 0;
    border-left: 0;
  }
}

@media (max-width: 760px) {
  .overview-depth-panel__grid {
    grid-template-columns: 1fr;
  }

  .overview-depth-metric,
  .overview-depth-metric + .overview-depth-metric {
    padding-left: 0;
    border-left: 0;
  }
}
</style>
