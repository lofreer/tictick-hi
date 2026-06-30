<template>
  <NModal v-model:show="modalOpen" preset="card" :title="t('research.invalidIssueDetailsTitle')" class="research-modal">
    <div v-if="task" class="research-gap-context">
      <NText depth="3">{{ task.exchange }} / {{ task.symbol }} / {{ task.interval }}</NText>
    </div>
    <NSpace align="center" class="research-invalid-issue-filters">
      <NSelect
        v-model:value="issueCode"
        clearable
        :options="issueCodeOptions"
        size="small"
        :placeholder="t('research.invalidIssueFilterType')"
        @update:value="applyFilters"
      />
      <NDatePicker
        v-model:value="timeRange"
        clearable
        type="datetimerange"
        size="small"
        :start-placeholder="t('research.invalidIssueFilterFrom')"
        :end-placeholder="t('research.invalidIssueFilterTo')"
        @update:value="applyFilters"
      />
      <NButton size="small" secondary @click="resetFilters">{{ t("common.reset") }}</NButton>
    </NSpace>
    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" :title="error" retryable @retry="task && load(task)" />
    <EmptyState v-else-if="!details || details.issues.length === 0" :title="t('research.noInvalidIssueDetails')" />
    <NDataTable v-else :columns="columns" :data="details.issues" :bordered="false" size="small" />
    <template #footer>
      <NSpace align="center" justify="space-between">
        <NSpace align="center">
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
          <NText v-if="repairNotice" :type="repairNoticeType">{{ repairNotice }}</NText>
          <NTag v-if="repairResult" :bordered="false" :type="repairResult.limited ? 'warning' : 'default'">
            {{
              t("research.invalidIssueRepairResultSummary", {
                created: repairResult.createdTasks.length,
                limit: repairResult.repairLimit,
                skipped: repairResult.skippedExisting,
                total: repairResult.totalCount,
              })
            }}
          </NTag>
          <NTag v-if="repairResult?.limited" :bordered="false" type="warning">
            {{ t("research.invalidIssueRepairResultLimited") }}
          </NTag>
          <NTag
            v-for="repairTask in repairTaskWindowTags"
            :key="repairTask.key"
            :bordered="false"
            :title="repairTask.title"
          >
            {{ repairTask.label }}
          </NTag>
          <NTag v-if="hiddenRepairTaskCount > 0" :bordered="false">
            {{ t("research.invalidIssueRepairTaskMore", { count: hiddenRepairTaskCount }) }}
          </NTag>
        </NSpace>
        <NSpace align="center" justify="end">
          <NPagination
            v-if="pageCount > 1"
            :disabled="loading"
            :page="page"
            :page-count="pageCount"
            size="small"
            @update:page="changePage"
          />
          <NButton
            v-if="details && details.totalCount > 0"
            :loading="repairLoading"
            secondary
            size="small"
            type="warning"
            @click="repairInvalidIssues"
          >
            {{ t("research.repairInvalidIssues") }}
          </NButton>
          <NButton @click="modalOpen = false">{{ t("common.close") }}</NButton>
        </NSpace>
      </NSpace>
    </template>
  </NModal>
</template>

<script setup lang="ts">
import {
  NButton,
  NDataTable,
  NDatePicker,
  NModal,
  NPagination,
  NSelect,
  NSpace,
  NTag,
  NText,
  type DataTableColumns,
  type SelectOption,
} from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";

import EmptyState from "@/components/common/EmptyState.vue";
import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import { dataApi, type DataSyncInvalidIssueQuery } from "@/services/api/data";
import type { CandleIssue, DataSyncGapRepairResult, DataSyncInvalidIssueList, DataSyncTask } from "@/types/app";
import type { RepairDataSyncInvalidIssuesRequest } from "@/types/app";
import { formatCompactDateTime } from "@/utils/displayText";

const { t } = useI18n();
const emit = defineEmits<{ repaired: [] }>();

const modalOpen = ref(false);
const loading = ref(false);
const repairLoading = ref(false);
const error = ref("");
const repairNotice = ref("");
const repairNoticeType = ref<"success" | "error" | "warning" | "default">("default");
const repairResult = ref<DataSyncGapRepairResult | null>(null);
const task = ref<DataSyncTask | null>(null);
const details = ref<DataSyncInvalidIssueList | null>(null);
const page = ref(1);
const issueCode = ref<string | null>(null);
const timeRange = ref<[number, number] | null>(null);
const pageSize = 50;
const issueCodes = [
  "invalid_open_price",
  "invalid_high_price",
  "invalid_low_price",
  "invalid_close_price",
  "invalid_volume",
  "invalid_high_bound",
  "invalid_low_bound",
];

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
const issueCodeOptions = computed<SelectOption[]>(() =>
  issueCodes.map((code) => ({
    label: invalidIssueLabel(code, code),
    value: code,
  })),
);
const pageCount = computed(() => (details.value ? Math.max(1, Math.ceil(details.value.totalCount / pageSize)) : 1));
const displayedIssueCount = computed(() => {
  if (!details.value) return 0;
  return Math.min(details.value.totalCount, details.value.offset + details.value.returnedCount);
});
const repairTaskWindowTags = computed(() => (repairResult.value?.createdTasks ?? []).slice(0, 3).map((repairTask) => ({
  key: repairTask.id,
  label: t("research.invalidIssueRepairTaskWindow", {
    id: repairTask.id,
    window: repairTaskWindow(repairTask),
  }),
  title: `${repairTask.exchange} / ${repairTask.symbol} / ${repairTask.interval}`,
})));
const hiddenRepairTaskCount = computed(() =>
  Math.max(0, (repairResult.value?.createdTasks.length ?? 0) - repairTaskWindowTags.value.length),
);

defineExpose({ open });

async function open(nextTask: DataSyncTask) {
  task.value = nextTask;
  details.value = null;
  error.value = "";
  repairNotice.value = "";
  repairNoticeType.value = "default";
  repairResult.value = null;
  page.value = 1;
  issueCode.value = null;
  timeRange.value = null;
  modalOpen.value = true;
  await load(nextTask);
}

async function load(currentTask: DataSyncTask) {
  loading.value = true;
  error.value = "";
  try {
    const query: DataSyncInvalidIssueQuery = {
      limit: pageSize,
      offset: (page.value - 1) * pageSize,
    };
    if (issueCode.value) {
      query.code = issueCode.value;
    }
    if (timeRange.value) {
      query.from = new Date(timeRange.value[0]).toISOString();
      query.to = new Date(timeRange.value[1]).toISOString();
    }
    details.value = await dataApi.getTaskInvalidIssues(currentTask.id, query);
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

async function applyFilters() {
  if (!task.value) return;
  repairNotice.value = "";
  repairResult.value = null;
  page.value = 1;
  await load(task.value);
}

async function resetFilters() {
  if (!task.value) return;
  repairNotice.value = "";
  repairResult.value = null;
  issueCode.value = null;
  timeRange.value = null;
  page.value = 1;
  await load(task.value);
}

async function repairInvalidIssues() {
  if (!task.value || repairLoading.value) return;
  repairLoading.value = true;
  repairNotice.value = "";
  repairNoticeType.value = "default";
  repairResult.value = null;
  try {
    const result = await dataApi.repairTaskInvalidIssues(task.value.id, currentRepairRequest());
    repairResult.value = result;
    if (result.createdTasks.length > 0) {
      repairNotice.value = t("research.invalidIssueRepairQueued", { count: result.createdTasks.length });
      repairNoticeType.value = "success";
    } else if (result.skippedExisting > 0) {
      repairNotice.value = t("research.invalidIssueRepairAlreadyQueued");
      repairNoticeType.value = "success";
    } else {
      repairNotice.value = t("research.noRepairableInvalidIssues");
      repairNoticeType.value = "warning";
    }
    emit("repaired");
    await load(task.value);
  } catch {
    repairNotice.value = t("research.invalidIssueRepairFailed");
    repairNoticeType.value = "error";
  } finally {
    repairLoading.value = false;
  }
}

function currentRepairRequest(): RepairDataSyncInvalidIssuesRequest {
  const request: RepairDataSyncInvalidIssuesRequest = {};
  if (issueCode.value) {
    request.code = issueCode.value;
  }
  if (timeRange.value) {
    request.from = new Date(timeRange.value[0]).toISOString();
    request.to = new Date(timeRange.value[1]).toISOString();
  }
  return request;
}

function repairTaskWindow(repairTask: DataSyncTask) {
  const from = repairTask.startTime ? formatCompactDateTime(repairTask.startTime) : "-";
  const to = repairTask.endTime ? formatCompactDateTime(repairTask.endTime) : "-";
  return `${from} - ${to}`;
}

function invalidIssueLabel(code?: string, fallback?: string) {
  if (!code) return fallback || t("research.invalidCandleIssue.unknown");
  const key = `research.invalidCandleIssue.${code}`;
  const translated = t(key);
  return translated === key ? fallback || code : translated;
}
</script>

<style scoped>
.research-invalid-issue-filters {
  margin-bottom: 12px;
}

.research-invalid-issue-filters :deep(.n-select) {
  width: 190px;
}

.research-invalid-issue-filters :deep(.n-date-picker) {
  width: 360px;
  max-width: 100%;
}

@media (max-width: 680px) {
  .research-invalid-issue-filters :deep(.n-select),
  .research-invalid-issue-filters :deep(.n-date-picker) {
    width: 100%;
  }
}
</style>
