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
  <NTag v-for="repairTask in repairTaskWindowTags" :key="repairTask.key" :bordered="false" :title="repairTask.title">
    {{ repairTask.label }}
  </NTag>
  <NTag v-if="hiddenRepairTaskCount > 0" :bordered="false">
    {{ t("research.marketRepairTaskMore", { count: hiddenRepairTaskCount }) }}
  </NTag>
</template>

<script setup lang="ts">
import { NTag } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import type { DataSyncGapRepairResult, DataSyncTask } from "@/types/app";
import { formatCompactDateTime } from "@/utils/displayText";

const props = defineProps<{
  result: DataSyncGapRepairResult | null;
}>();

const { t } = useI18n();

const repairTaskWindowTags = computed(() =>
  (props.result?.createdTasks ?? []).slice(0, 3).map((repairTask) => ({
    key: repairTask.id,
    label: t("research.marketRepairTaskWindow", {
      id: repairTask.id,
      window: repairTaskWindow(repairTask),
    }),
    title: `${repairTask.exchange} / ${repairTask.symbol} / ${repairTask.interval}`,
  })),
);
const hiddenRepairTaskCount = computed(() =>
  Math.max(0, (props.result?.createdTasks.length ?? 0) - repairTaskWindowTags.value.length),
);

function repairTaskWindow(repairTask: DataSyncTask) {
  const from = repairTask.startTime ? formatCompactDateTime(repairTask.startTime) : "-";
  const to = repairTask.endTime ? formatCompactDateTime(repairTask.endTime) : "-";
  return `${from} - ${to}`;
}
</script>
