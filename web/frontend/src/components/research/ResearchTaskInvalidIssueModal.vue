<template>
  <NModal v-model:show="modalOpen" preset="card" :title="t('research.invalidIssueDetailsTitle')" class="research-modal">
    <div v-if="task" class="research-gap-context">
      <NText depth="3">{{ task.exchange }} / {{ task.symbol }} / {{ task.interval }}</NText>
    </div>
    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" :title="error" retryable @retry="task && load(task)" />
    <EmptyState v-else-if="!details || details.issues.length === 0" :title="t('research.noInvalidIssueDetails')" />
    <NDataTable v-else :columns="columns" :data="details.issues" :bordered="false" size="small" />
    <template #footer>
      <NSpace justify="end">
        <NTag v-if="details?.limited" :bordered="false" type="warning">
          {{
            t("research.invalidIssueDetailsLimited", {
              returned: details.returnedCount,
              total: details.totalCount,
              limit: details.issueLimit,
            })
          }}
        </NTag>
        <NButton @click="modalOpen = false">{{ t("common.close") }}</NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<script setup lang="ts">
import { NButton, NDataTable, NModal, NSpace, NTag, NText, type DataTableColumns } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";

import EmptyState from "@/components/common/EmptyState.vue";
import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import { dataApi } from "@/services/api/data";
import type { CandleIssue, DataSyncInvalidIssueList, DataSyncTask } from "@/types/app";
import { formatCompactDateTime } from "@/utils/displayText";

const { t } = useI18n();

const modalOpen = ref(false);
const loading = ref(false);
const error = ref("");
const task = ref<DataSyncTask | null>(null);
const details = ref<DataSyncInvalidIssueList | null>(null);

const columns = computed<DataTableColumns<CandleIssue>>(() => [
  {
    title: t("research.invalidIssueOpenTime"),
    key: "openTime",
    minWidth: 180,
    render: (row) => (row.openTime ? formatCompactDateTime(row.openTime) : "-"),
  },
  {
    title: t("research.invalidIssueType"),
    key: "code",
    minWidth: 180,
    render: (row) => invalidIssueLabel(row.code, row.message),
  },
  {
    title: t("research.invalidIssueMessage"),
    key: "message",
    minWidth: 260,
    render: (row) => row.message || "-",
  },
]);

defineExpose({ open });

async function open(nextTask: DataSyncTask) {
  task.value = nextTask;
  details.value = null;
  error.value = "";
  modalOpen.value = true;
  await load(nextTask);
}

async function load(currentTask: DataSyncTask) {
  loading.value = true;
  error.value = "";
  try {
    details.value = await dataApi.getTaskInvalidIssues(currentTask.id);
  } catch {
    error.value = t("research.invalidIssueDetailsLoadFailed");
  } finally {
    loading.value = false;
  }
}

function invalidIssueLabel(code?: string, fallback?: string) {
  if (!code) return fallback || t("research.invalidCandleIssue.unknown");
  const key = `research.invalidCandleIssue.${code}`;
  const translated = t(key);
  return translated === key ? fallback || code : translated;
}
</script>
