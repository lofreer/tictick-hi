<template>
  <NTag v-if="result" :bordered="false" :type="result.limited ? 'warning' : 'default'">
    {{
      t("research.marketRepairResultSummary", {
        created: result.createdTasks.length,
        limit: result.repairLimit,
        skipped: result.skippedExisting,
        total: result.totalCount,
      })
    }}
  </NTag>
  <NTag v-if="result?.limited" :bordered="false" type="warning">
    {{ t("research.marketRepairResultLimited") }}
  </NTag>
  <NTag v-if="settlementTag" :bordered="false" :type="settlementTag.type">
    {{ t(settlementTag.key, settlementTag.values) }}
  </NTag>
  <NTag v-if="chartWindowTag" :bordered="false" :type="chartWindowTag.type">
    {{ t(chartWindowTag.key, chartWindowTag.values) }}
  </NTag>
  <NTag v-for="repairTask in repairTaskWindowTags" :key="repairTask.key" :bordered="false" :title="repairTask.title" :type="repairTask.type">
    {{ repairTask.label }}
  </NTag>
  <NTag v-if="hiddenRepairTaskCount > 0" :bordered="false">
    {{ t("research.marketRepairTaskMore", { count: hiddenRepairTaskCount }) }}
  </NTag>
</template>

<script setup lang="ts">
import { NTag, type TagProps } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import type { CandleResult, DataSyncGapRepairResult, DataSyncTask } from "@/types/app";
import { formatCompactDateTime } from "@/utils/displayText";

const props = defineProps<{
  candleResult?: CandleResult | null;
  result: DataSyncGapRepairResult | null;
  tasks?: DataSyncTask[];
}>();

const { t } = useI18n();
const terminalStatuses = new Set<DataSyncTask["status"]>(["succeeded", "failed", "cancelled", "paused"]);

const latestTasks = computed(() => new Map((props.tasks ?? []).map((task) => [task.id, task])));
const repairTasks = computed(() =>
  (props.result?.createdTasks ?? []).map((repairTask) => latestTasks.value.get(repairTask.id) ?? repairTask),
);
const repairTaskWindowTags = computed(() =>
  repairTasks.value.slice(0, 3).map((latestTask) => {
    return {
      key: latestTask.id,
      label: t("research.marketRepairTaskWindowStatus", {
        health: t(`research.dataHealth.${latestTask.dataHealth}`),
        id: latestTask.id,
        status: t(`status.${latestTask.status}`),
        window: repairTaskWindow(latestTask),
      }),
      title: `${latestTask.exchange} / ${latestTask.symbol} / ${latestTask.interval} / ${t(`status.${latestTask.status}`)} / ${t(`research.dataHealth.${latestTask.dataHealth}`)}`,
      type: dataHealthTagType(latestTask.dataHealth),
    };
  }),
);
const hiddenRepairTaskCount = computed(() =>
  Math.max(0, (props.result?.createdTasks.length ?? 0) - repairTaskWindowTags.value.length),
);
const settlementTag = computed(() => {
  if (repairTasks.value.length === 0) return null;
  const running = repairTasks.value.filter((task) => !terminalStatuses.has(task.status)).length;
  if (running > 0) return { key: "research.marketRepairSettlementRunning", type: "info" as const, values: { count: running } };
  const hasFailed = repairTasks.value.some((task) => task.status === "failed" || task.dataHealth === "failed" || task.dataHealth === "invalid");
  if (hasFailed) return { key: "research.marketRepairSettlementFailed", type: "error" as const, values: {} };
  const allOK = repairTasks.value.every((task) => task.status === "succeeded" && task.dataHealth === "ok");
  if (allOK) return { key: "research.marketRepairSettlementOK", type: "success" as const, values: {} };
  return { key: "research.marketRepairSettlementReview", type: "warning" as const, values: {} };
});
const chartWindowTag = computed(() => {
  if (!props.result || !props.candleResult) return null;
  if (props.candleResult.health === "ok") return { key: "research.marketRepairChartWindowOK", type: "success" as const, values: {} };
  if (props.candleResult.health === "gap") {
    return { key: "research.marketRepairChartWindowGap", type: "warning" as const, values: { count: props.candleResult.gaps.length } };
  }
  if (props.candleResult.health === "invalid") {
    return { key: "research.marketRepairChartWindowInvalid", type: "error" as const, values: { count: props.candleResult.issues.length } };
  }
  return {
    key: "research.marketRepairChartWindowReview",
    type: dataHealthTagType(props.candleResult.health),
    values: { health: t(`research.dataHealth.${props.candleResult.health}`) },
  };
});

function repairTaskWindow(repairTask: DataSyncTask) {
  const from = repairTask.startTime ? formatCompactDateTime(repairTask.startTime) : "-";
  const to = repairTask.endTime ? formatCompactDateTime(repairTask.endTime) : "-";
  return `${from} - ${to}`;
}

function dataHealthTagType(health: DataSyncTask["dataHealth"]): TagProps["type"] {
  if (health === "ok") return "success";
  if (health === "invalid" || health === "failed") return "error";
  if (health === "gap" || health === "insufficient" || health === "retrying") return "warning";
  return "default";
}
</script>
