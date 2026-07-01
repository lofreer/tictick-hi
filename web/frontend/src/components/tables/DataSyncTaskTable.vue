<template>
  <NDataTable
    class="data-sync-task-table"
    :columns="columns"
    :data="tasks"
    :row-key="rowKey"
    :bordered="false"
    :single-line="false"
    :max-height="260"
    :scroll-x="2282"
    size="small"
  />
</template>

<script setup lang="ts">
import { Eye, ListChecks, Play, RefreshCw, RotateCcw, Square, Trash2, TriangleAlert, Wrench } from "@lucide/vue";
import {
  NButton,
  NDataTable,
  NSpace,
  NTag,
  NText,
  NTooltip,
  type DataTableColumns,
  type DataTableRowKey,
  type TagProps,
} from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";

import StatusBadge from "@/components/common/StatusBadge.vue";
import DataSyncQualitySummary from "@/components/tables/DataSyncQualitySummary.vue";
import type { DataSyncTask } from "@/types/app";
import { formatCompactDateTime, summarizeText } from "@/utils/displayText";
import { sanitizeExternalError } from "@/utils/errorText";

const props = defineProps<{ tasks: DataSyncTask[]; repairingTaskId?: string }>();
const emit = defineEmits<{
  view: [task: DataSyncTask];
  delete: [task: DataSyncTask];
  "view-gaps": [task: DataSyncTask];
  "view-invalid": [task: DataSyncTask];
  "repair-gaps": [task: DataSyncTask];
  retry: [task: DataSyncTask];
  "toggle-realtime": [task: DataSyncTask];
  "toggle-sync": [task: DataSyncTask];
}>();

const { t } = useI18n();

const columns = computed<DataTableColumns<DataSyncTask>>(() => [
  { title: t("research.exchange"), key: "exchange", width: 92 },
  { title: t("research.symbol"), key: "symbol", width: 110 },
  {
    title: t("research.marketStatus"),
    key: "marketStatus",
    width: 148,
    render: (row) =>
      h(NTag, { bordered: false, size: "small", type: marketStatusTagType(row.marketStatus) }, () =>
        h("span", { class: "task-market-status", title: marketStatusLabel(row) }, marketStatusLabel(row)),
      ),
  },
  { title: t("research.interval"), key: "interval", width: 76 },
  {
    title: t("research.syncWindow"),
    key: "syncWindow",
    width: 220,
    render: syncWindowCell,
  },
  {
    title: t("research.latestSyncedAt"),
    key: "latestSyncedAt",
    minWidth: 150,
    render: (row) => timeCell(row.latestSyncedAt),
  },
  {
    title: t("research.dataHealth"),
    key: "dataHealth",
    width: 104,
    render: (row) =>
      h(NTag, { bordered: false, size: "small", type: dataHealthTagType(row.dataHealth) }, () =>
        t(`research.dataHealth.${row.dataHealth}`),
      ),
  },
  {
    title: t("research.qualitySummary"),
    key: "qualitySummary",
    width: 220,
    render: (row) => h(DataSyncQualitySummary, { task: row }),
  },
  {
    title: t("research.realtime"),
    key: "realtimeEnabled",
    width: 86,
    render: (row) => (row.realtimeEnabled ? t("status.running") : t("status.paused")),
  },
  {
    title: t("research.sync"),
    key: "status",
    width: 92,
    render: (row) => h(StatusBadge, { status: row.status }),
  },
  {
    title: t("research.lastError"),
    key: "lastError",
    width: 300,
    render: lastErrorCell,
  },
  {
    title: t("research.nextAttemptAt"),
    key: "nextAttemptAt",
    minWidth: 150,
    render: (row) => timeCell(row.nextAttemptAt),
  },
  {
    title: t("research.exchangeBackoffUntil"),
    key: "exchangeBackoffUntil",
    minWidth: 160,
    render: exchangeBackoffCell,
  },
  {
    title: t("research.actions"),
    key: "actions",
    width: 324,
    render: (row) =>
      h(NSpace, { size: 4, wrap: false }, () => [
        iconButton(Eye, t("research.viewChart"), () => emit("view", row)),
        ...(hasRepairableTaskGaps(row)
          ? [iconButton(ListChecks, t("research.viewTaskGaps"), () => emit("view-gaps", row))]
          : []),
        ...(hasInvalidTaskIssues(row)
          ? [iconButton(TriangleAlert, t("research.viewTaskInvalidIssues"), () => emit("view-invalid", row), "error")]
          : []),
        ...(hasRepairableTaskGaps(row)
          ? [
              iconButton(
                Wrench,
                marketActionLabel(row, t("research.repairTaskGaps")),
                () => emit("repair-gaps", row),
                "warning",
                props.repairingTaskId === row.id,
                !taskMarketActive(row),
              ),
            ]
          : []),
        iconButton(
          row.realtimeEnabled ? Square : Play,
          realtimeButtonLabel(row),
          () => emit("toggle-realtime", row),
          "default",
          false,
          !row.realtimeEnabled && !taskMarketActive(row),
        ),
        row.status === "failed"
          ? iconButton(
              RotateCcw,
              marketActionLabel(row, t("common.retry")),
              () => emit("retry", row),
              "default",
              false,
              !taskMarketActive(row),
            )
          : iconButton(
              row.syncEnabled ? Square : RefreshCw,
              syncButtonLabel(row),
              () => emit("toggle-sync", row),
              "default",
              false,
              !row.syncEnabled && !taskMarketActive(row),
            ),
        iconButton(Trash2, t("research.deleteTask"), () => emit("delete", row), "error"),
      ]),
  },
]);

function iconButton(
  icon: typeof Eye,
  label: string,
  onClick: () => void,
  type: "default" | "error" | "warning" = "default",
  loading = false,
  disabled = false,
) {
  return h(
    NButton,
    { disabled: loading || disabled, loading, size: "tiny", quaternary: true, type, title: label, onClick },
    { icon: () => h(icon, { size: 15 }) },
  );
}

function realtimeButtonLabel(row: DataSyncTask) {
  if (row.realtimeEnabled) return t("research.stopRealtime");
  if (!taskMarketActive(row)) return t("research.marketNotActiveAction");
  return t("research.startRealtime");
}

function syncButtonLabel(row: DataSyncTask) {
  if (row.syncEnabled) return t("research.stopSync");
  if (!taskMarketActive(row)) return t("research.marketNotActiveAction");
  return t("research.startSync");
}

function taskMarketActive(row: DataSyncTask) {
  return row.marketStatus === "active";
}

function marketActionLabel(row: DataSyncTask, activeLabel: string) {
  return taskMarketActive(row) ? activeLabel : t("research.marketNotActiveAction");
}

function marketStatusLabel(row: DataSyncTask) {
  const base = t(`research.marketStatus.${row.marketStatus}`);
  const detail = (row.marketStatusDetail ?? "").trim();
  if (!detail || detail === row.marketStatus || (row.marketStatus === "active" && detail.toLowerCase() === "active")) {
    return base;
  }
  return `${base} · ${detail}`;
}

function hasRepairableTaskGaps(row: DataSyncTask) {
  return (row.gapSummary?.count ?? 0) > 0;
}

function hasInvalidTaskIssues(row: DataSyncTask) {
  return (row.invalidSummary?.count ?? 0) > 0;
}

function syncWindowCell(row: DataSyncTask) {
  const text = syncWindowText(row);
  return h(
    NTooltip,
    { trigger: "hover", width: 420 },
    {
      trigger: () =>
        h(
          "span",
          {
            class: "task-sync-window",
            title: text,
          },
          text,
        ),
      default: () =>
        h(
          "span",
          {
            style: {
              display: "block",
              whiteSpace: "normal",
              overflowWrap: "anywhere",
            },
          },
          text,
        ),
    },
  );
}

function syncWindowText(row: DataSyncTask) {
  const window = syncWindowRangeText(row);
  if (row.repairSourceTaskId) {
    return t("research.syncWindowRepairSource", { source: row.repairSourceTaskId, window });
  }
  return window;
}

function syncWindowRangeText(row: DataSyncTask) {
  if (row.startTime && row.endTime) {
    return t("research.syncWindowRange", { from: row.startTime, to: row.endTime });
  }
  if (row.startTime) {
    return t("research.syncWindowFrom", { from: row.startTime });
  }
  if (row.endTime) {
    return t("research.syncWindowTo", { to: row.endTime });
  }
  return t("research.syncWindowContinuous");
}

function lastErrorCell(row: DataSyncTask) {
  const detail = sanitizeExternalError(row.lastError);
  if (!detail) {
    return h(NText, { depth: 3 }, () => "-");
  }
  const summary = summarizeText(detail);
  return h(
    NTooltip,
    { trigger: "hover", width: 420 },
    {
      trigger: () =>
        h(
          "span",
          {
            class: "task-error-text",
            title: detail,
          },
          summary,
        ),
      default: () =>
        h(
          "span",
          {
            style: {
              display: "block",
              whiteSpace: "normal",
              overflowWrap: "anywhere",
            },
          },
          detail,
        ),
    },
  );
}

function exchangeBackoffCell(row: DataSyncTask) {
  if (!row.exchangeBackoffUntil) {
    return h(NText, { depth: 3 }, () => "-");
  }
  const detail = sanitizeExternalError(row.exchangeBackoffLastError);
  if (!detail) {
    return timeCell(row.exchangeBackoffUntil);
  }
  const formatted = formatCompactDateTime(row.exchangeBackoffUntil);
  return h(
    NTooltip,
    { trigger: "hover", width: 420 },
    {
      trigger: () =>
        h(
          "span",
          {
            class: "task-exchange-backoff",
            title: `${row.exchangeBackoffUntil} · ${detail}`,
          },
          formatted,
        ),
      default: () =>
        h(
          "span",
          {
            style: {
              display: "block",
              whiteSpace: "normal",
              overflowWrap: "anywhere",
            },
          },
          detail,
        ),
    },
  );
}

function timeCell(value?: string) {
  if (!value) {
    return h(NText, { depth: 3 }, () => "-");
  }
  return h("span", { class: "task-time-text", title: value }, formatCompactDateTime(value));
}

function dataHealthTagType(health: DataSyncTask["dataHealth"]): TagProps["type"] {
  if (health === "ok") return "success";
  if (health === "gap" || health === "retrying") return "warning";
  if (health === "failed" || health === "invalid") return "error";
  if (health === "syncing") return "info";
  return "default";
}

function marketStatusTagType(status: DataSyncTask["marketStatus"]): TagProps["type"] {
  if (status === "active") return "success";
  if (status === "inactive") return "warning";
  return "error";
}

function rowKey(row: DataSyncTask): DataTableRowKey {
  return row.id;
}
</script>

<style scoped>
.task-error-text,
.task-sync-window,
.task-exchange-backoff,
.task-time-text {
  display: block;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-error-text {
  color: var(--tt-danger);
}

.task-exchange-backoff {
  color: var(--tt-warning);
}

.task-sync-window {
  color: var(--tt-text-secondary);
}

.data-sync-task-table {
  width: 100%;
  max-width: 100%;
  min-width: 0;
}

.data-sync-task-table :deep(.n-data-table-base-table),
.data-sync-task-table :deep(.n-data-table-base-table-header),
.data-sync-task-table :deep(.n-data-table-base-table-body),
.data-sync-task-table :deep(.n-data-table-scroll-container) {
  max-width: 100%;
  min-width: 0;
}

.data-sync-task-table :deep(.n-data-table-th),
.data-sync-task-table :deep(.n-data-table-td) {
  min-width: 0;
}
</style>
