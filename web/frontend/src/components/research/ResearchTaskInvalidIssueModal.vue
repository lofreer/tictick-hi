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
      <NSpace align="center" justify="space-between">
        <NTag
          v-if="details && details.totalCount > 0"
          :bordered="false"
          :type="details.limited ? 'warning' : 'default'"
        >
          {{
            t("research.invalidIssueDetailsLimited", {
              returned: displayedIssueCount,
              total: details.totalCount,
              limit: details.issueLimit,
            })
          }}
        </NTag>
        <span v-else />
        <NSpace align="center" justify="end">
          <NPagination
            v-if="pageCount > 1"
            :disabled="loading"
            :page="page"
            :page-count="pageCount"
            size="small"
            @update:page="changePage"
          />
          <NButton @click="modalOpen = false">{{ t("common.close") }}</NButton>
        </NSpace>
      </NSpace>
    </template>
  </NModal>
</template>

<script setup lang="ts">
import { NButton, NDataTable, NModal, NPagination, NSpace, NTag, NText, type DataTableColumns } from "naive-ui";
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
const page = ref(1);
const pageSize = 50;

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
const pageCount = computed(() => (details.value ? Math.max(1, Math.ceil(details.value.totalCount / pageSize)) : 1));
const displayedIssueCount = computed(() => {
  if (!details.value) return 0;
  return Math.min(details.value.totalCount, details.value.offset + details.value.returnedCount);
});

defineExpose({ open });

async function open(nextTask: DataSyncTask) {
  task.value = nextTask;
  details.value = null;
  error.value = "";
  page.value = 1;
  modalOpen.value = true;
  await load(nextTask);
}

async function load(currentTask: DataSyncTask) {
  loading.value = true;
  error.value = "";
  try {
    details.value = await dataApi.getTaskInvalidIssues(currentTask.id, {
      limit: pageSize,
      offset: (page.value - 1) * pageSize,
    });
  } catch {
    error.value = t("research.invalidIssueDetailsLoadFailed");
  } finally {
    loading.value = false;
  }
}

async function changePage(nextPage: number) {
  if (!task.value || nextPage === page.value) return;
  page.value = nextPage;
  await load(task.value);
}

function invalidIssueLabel(code?: string, fallback?: string) {
  if (!code) return fallback || t("research.invalidCandleIssue.unknown");
  const key = `research.invalidCandleIssue.${code}`;
  const translated = t(key);
  return translated === key ? fallback || code : translated;
}
</script>
