<template>
  <section class="surface overview-monitoring-panel">
    <div class="overview-monitoring-panel__header">
      <h2>{{ t("overview.monitoringContext") }}</h2>
      <NTag size="small">{{ t("overview.monitoringSnapshot") }}</NTag>
    </div>

    <div class="overview-monitoring-panel__grid">
      <div v-for="item in contextItems" :key="item.key" class="overview-monitoring-item">
        <div class="overview-monitoring-item__header">
          <span>{{ item.label }}</span>
          <NTag :type="item.statusType" size="small">{{ item.statusLabel }}</NTag>
        </div>
        <strong>{{ item.value }}</strong>
        <p>{{ item.detail }}</p>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { NTag, type TagProps } from "naive-ui";
import { useI18n } from "vue-i18n";

import type { SystemHealth } from "@/types/app";

type MonitoringContextItem = {
  key: string;
  label: string;
  value: string;
  detail: string;
  statusLabel: string;
  statusType: TagProps["type"];
};

const props = defineProps<{
  alertCount: number;
  factsError: string;
  formatDate: (value?: string) => string;
  hasTrendData: boolean;
  health: SystemHealth | null;
  serviceCount: number;
  trendPointCount: number;
  trendsError: string;
}>();

const { t } = useI18n();
const observedSourceCount = 2;

const contextItems = computed<MonitoringContextItem[]>(() => {
  const degradedSources = Number(Boolean(props.factsError)) + Number(Boolean(props.trendsError));
  const healthySources = observedSourceCount - degradedSources;
  return [
    {
      key: "snapshot",
      label: t("overview.monitoring.snapshot"),
      value: props.health?.checkedAt ? props.formatDate(props.health.checkedAt) : "-",
      detail: t("overview.monitoring.snapshotDetail", { services: props.serviceCount, status: props.health?.status ?? "-" }),
      statusLabel: statusLabel(snapshotStatusType.value),
      statusType: snapshotStatusType.value,
    },
    {
      key: "sources",
      label: t("overview.monitoring.sources"),
      value: `${healthySources}/${observedSourceCount}`,
      detail: t("overview.monitoring.sourcesDetail", { degraded: degradedSources, total: observedSourceCount }),
      statusLabel: statusLabel(degradedSources > 0 ? "warning" : "success"),
      statusType: degradedSources > 0 ? "warning" : "success",
    },
    {
      key: "trend-coverage",
      label: t("overview.monitoring.trendCoverage"),
      value: props.hasTrendData ? t("overview.trendWindow.7d") : "-",
      detail: t("overview.monitoring.trendCoverageDetail", { points: props.trendPointCount }),
      statusLabel: statusLabel(props.trendsError || !props.hasTrendData ? "warning" : "success"),
      statusType: props.trendsError || !props.hasTrendData ? "warning" : "success",
    },
    {
      key: "alert-load",
      label: t("overview.monitoring.alertLoad"),
      value: String(props.alertCount),
      detail: t("overview.monitoring.alertLoadDetail", { count: props.alertCount }),
      statusLabel: statusLabel(props.alertCount > 0 ? "warning" : "success"),
      statusType: props.alertCount > 0 ? "warning" : "success",
    },
  ];
});

const snapshotStatusType = computed<TagProps["type"]>(() => {
  if (!props.health) return "warning";
  return props.health.status === "ok" ? "success" : "warning";
});

function statusLabel(type: TagProps["type"]) {
  if (type === "warning") return t("overview.monitoring.status.watch");
  if (type === "success") return t("overview.monitoring.status.ok");
  return t("overview.monitoring.status.limited");
}
</script>

<style scoped>
.overview-monitoring-panel {
  display: grid;
  min-width: 0;
  padding: 16px;
  gap: 14px;
}

.overview-monitoring-panel__header,
.overview-monitoring-item__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.overview-monitoring-panel h2 {
  margin: 0;
  font-size: 16px;
  line-height: 1.35;
  font-weight: 760;
}

.overview-monitoring-panel__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0;
}

.overview-monitoring-item {
  min-width: 0;
  min-height: 112px;
  padding: 12px 14px 12px 0;
  border-top: 1px solid var(--tt-line);
}

.overview-monitoring-item + .overview-monitoring-item {
  padding-left: 14px;
  border-left: 1px solid var(--tt-line);
}

.overview-monitoring-item__header {
  min-width: 0;
}

.overview-monitoring-item__header > span {
  overflow: hidden;
  color: var(--tt-muted);
  font-size: 12px;
  line-height: 1.45;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.overview-monitoring-item strong {
  display: block;
  margin-top: 10px;
  overflow-wrap: anywhere;
  font-size: 18px;
  line-height: 1.25;
}

.overview-monitoring-item p {
  margin: 10px 0 0;
  color: var(--tt-muted);
  font-size: 12px;
  line-height: 1.5;
}

@media (max-width: 1100px) {
  .overview-monitoring-panel__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .overview-monitoring-item:nth-child(2n + 1) {
    padding-left: 0;
    border-left: 0;
  }
}

@media (max-width: 760px) {
  .overview-monitoring-panel__grid {
    grid-template-columns: 1fr;
  }

  .overview-monitoring-item,
  .overview-monitoring-item + .overview-monitoring-item {
    padding-left: 0;
    border-left: 0;
  }
}
</style>
