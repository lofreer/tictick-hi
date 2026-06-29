<template>
  <NTooltip v-if="invalidText" trigger="hover" :width="420">
    <template #trigger>
      <span class="task-quality-summary task-invalid-summary" :title="invalidText">
        {{ invalidText }}
      </span>
    </template>
    <span class="task-quality-summary__detail">{{ invalidText }}</span>
  </NTooltip>
  <NTooltip v-else-if="gapText" trigger="hover" :width="420">
    <template #trigger>
      <span class="task-quality-summary task-gap-summary" :title="gapText">
        {{ gapText }}
      </span>
    </template>
    <span class="task-quality-summary__detail">{{ gapText }}</span>
  </NTooltip>
  <NText v-else depth="3">-</NText>
</template>

<script setup lang="ts">
import { NText, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import type { DataSyncTask } from "@/types/app";

const props = defineProps<{ task: DataSyncTask }>();

const { t } = useI18n();

const invalidText = computed(() => {
  const summary = props.task.invalidSummary;
  if (!summary || summary.count <= 0) return "";
  const issue = summary.firstIssue;
  if (!issue?.openTime) {
    return t("research.invalidSummaryCountOnly", { count: summary.count });
  }
  return t("research.invalidSummaryFirst", {
    count: summary.count,
    time: issue.openTime,
    reason: invalidIssueLabel(issue.code, issue.message),
  });
});

const gapText = computed(() => {
  const summary = props.task.gapSummary;
  if (!summary || summary.count <= 0) return "";
  if (!summary.firstGap) {
    return t("research.gapSummaryCountOnly", { count: summary.count });
  }
  return t("research.gapSummaryFirst", {
    count: summary.count,
    from: summary.firstGap.from,
    missing: summary.firstGap.missingCandles,
    to: summary.firstGap.to,
  });
});

function invalidIssueLabel(code?: string, fallback?: string) {
  if (!code) return fallback || t("research.invalidCandleIssue.unknown");
  const key = `research.invalidCandleIssue.${code}`;
  const translated = t(key);
  return translated === key ? fallback || code : translated;
}
</script>

<style scoped>
.task-quality-summary {
  display: block;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-quality-summary__detail {
  display: block;
  overflow-wrap: anywhere;
  white-space: normal;
}

.task-gap-summary {
  color: var(--tt-warning);
}

.task-invalid-summary {
  color: var(--tt-danger);
}
</style>
