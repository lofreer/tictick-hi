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

import type { DataSyncGapRepairResult, DataSyncTask } from "@/types/app";
import { formatCompactDateTime } from "@/utils/displayText";

const props = defineProps<{
  result: DataSyncGapRepairResult | null;
  tasks?: DataSyncTask[];
}>();

const { t } = useI18n();

const latestTasks = computed(() => new Map((props.tasks ?? []).map((task) => [task.id, task])));
const repairTaskWindowTags = computed(() =>
  (props.result?.createdTasks ?? []).slice(0, 3).map((repairTask) => {
    const latestTask = latestTasks.value.get(repairTask.id) ?? repairTask;
    return {
      key: repairTask.id,
      label: t("research.marketRepairTaskWindowStatus", {
        health: t(`research.dataHealth.${latestTask.dataHealth}`),
        id: repairTask.id,
        status: t(`status.${latestTask.status}`),
        window: repairTaskWindow(repairTask),
      }),
      title: `${latestTask.exchange} / ${latestTask.symbol} / ${latestTask.interval} / ${t(`status.${latestTask.status}`)} / ${t(`research.dataHealth.${latestTask.dataHealth}`)}`,
      type: dataHealthTagType(latestTask.dataHealth),
    };
  }),
);
const hiddenRepairTaskCount = computed(() =>
  Math.max(0, (props.result?.createdTasks.length ?? 0) - repairTaskWindowTags.value.length),
);

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
