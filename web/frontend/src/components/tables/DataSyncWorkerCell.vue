<template>
  <span class="task-worker-cell" :title="detailTitle">
    <span class="task-worker-cell__primary">{{ primaryText }}</span>
    <span v-if="secondaryText" class="task-worker-cell__secondary">{{ secondaryText }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import type { DataSyncTask } from "@/types/app";
import { formatCompactDateTime } from "@/utils/displayText";

const props = defineProps<{ task: DataSyncTask }>();
const { t } = useI18n();

const primaryText = computed(() => {
  if (props.task.lockedBy) return props.task.lockedBy;
  if (props.task.finishedAt) return t("research.workerFinished");
  if (props.task.startedAt) return t("research.workerStarted");
  return "-";
});

const secondaryText = computed(() => {
  if (props.task.heartbeatAt) return t("research.workerHeartbeatAt", { time: formatCompactDateTime(props.task.heartbeatAt) });
  if (props.task.lockedUntil) return t("research.workerLockedUntil", { time: formatCompactDateTime(props.task.lockedUntil) });
  if (props.task.finishedAt) return formatCompactDateTime(props.task.finishedAt);
  if (props.task.startedAt) return formatCompactDateTime(props.task.startedAt);
  return "";
});

const detailTitle = computed(() => {
  const details = [
    props.task.lockedBy ? t("research.workerLockedBy", { worker: props.task.lockedBy }) : "",
    props.task.heartbeatAt ? t("research.workerHeartbeatAt", { time: props.task.heartbeatAt }) : "",
    props.task.lockedUntil ? t("research.workerLockedUntil", { time: props.task.lockedUntil }) : "",
    props.task.startedAt ? t("research.workerStartedAt", { time: props.task.startedAt }) : "",
    props.task.finishedAt ? t("research.workerFinishedAt", { time: props.task.finishedAt }) : "",
  ].filter(Boolean);
  return details.length > 0 ? details.join(" | ") : "-";
});
</script>

<style scoped>
.task-worker-cell {
  display: grid;
  gap: 2px;
  min-width: 0;
  max-width: 100%;
}

.task-worker-cell__primary,
.task-worker-cell__secondary {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-worker-cell__primary {
  color: var(--tt-text);
  font-weight: 650;
}

.task-worker-cell__secondary {
  color: var(--tt-muted);
  font-size: 12px;
  line-height: 1.35;
}
</style>
