<template>
  <NDataTable
    :columns="columns"
    :data="tasks"
    :row-key="rowKey"
    :bordered="false"
    :single-line="false"
    :scroll-x="1020"
    size="small"
  />
</template>

<script setup lang="ts">
import { Eye, Play, RefreshCw, Square, Trash2 } from "@lucide/vue";
import {
  NButton,
  NDataTable,
  NSpace,
  NText,
  NTooltip,
  type DataTableColumns,
  type DataTableRowKey,
} from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";

import StatusBadge from "@/components/common/StatusBadge.vue";
import type { DataSyncTask } from "@/types/app";

const props = defineProps<{ tasks: DataSyncTask[] }>();
const emit = defineEmits<{
  view: [task: DataSyncTask];
  delete: [task: DataSyncTask];
  "toggle-realtime": [task: DataSyncTask];
  "toggle-sync": [task: DataSyncTask];
}>();

const { t } = useI18n();

const columns = computed<DataTableColumns<DataSyncTask>>(() => [
  { title: t("research.exchange"), key: "exchange", width: 92 },
  { title: t("research.symbol"), key: "symbol", width: 110 },
  { title: t("research.interval"), key: "interval", width: 76 },
  {
    title: t("research.latestSyncedAt"),
    key: "latestSyncedAt",
    minWidth: 150,
    render: (row) => row.latestSyncedAt ?? "-",
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
    minWidth: 180,
    render: lastErrorCell,
  },
  {
    title: t("research.actions"),
    key: "actions",
    width: 232,
    render: (row) =>
      h(NSpace, { size: 4, wrap: false }, () => [
        iconButton(Eye, t("research.viewChart"), () => emit("view", row)),
        iconButton(
          row.realtimeEnabled ? Square : Play,
          row.realtimeEnabled ? t("research.stopRealtime") : t("research.startRealtime"),
          () => emit("toggle-realtime", row),
        ),
        iconButton(
          row.syncEnabled ? Square : RefreshCw,
          row.syncEnabled ? t("research.stopSync") : t("research.startSync"),
          () => emit("toggle-sync", row),
        ),
        iconButton(Trash2, t("research.deleteTask"), () => emit("delete", row), "error"),
      ]),
  },
]);

function iconButton(
  icon: typeof Eye,
  label: string,
  onClick: () => void,
  type: "default" | "error" = "default",
) {
  return h(
    NButton,
    { size: "tiny", quaternary: true, type, title: label, onClick },
    { icon: () => h(icon, { size: 15 }) },
  );
}

function lastErrorCell(row: DataSyncTask) {
  if (!row.lastError) {
    return h(NText, { depth: 3 }, () => "-");
  }
  const summary = summarizeError(row.lastError);
  return h(
    NTooltip,
    { trigger: "hover", width: 420 },
    {
      trigger: () =>
        h(
          "span",
          {
            class: "task-error-text",
            title: row.lastError,
            style: {
              display: "block",
              maxWidth: "260px",
              overflow: "hidden",
              textOverflow: "ellipsis",
              whiteSpace: "nowrap",
            },
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
          row.lastError,
        ),
    },
  );
}

function summarizeError(value: string) {
  const normalized = value.replace(/\s+/g, " ").trim();
  if (normalized.length <= 90) {
    return normalized;
  }
  return `${normalized.slice(0, 87)}...`;
}

function rowKey(row: DataSyncTask): DataTableRowKey {
  return row.id;
}
</script>

<style scoped>
.task-error-text {
  display: block;
  max-width: 260px;
  overflow: hidden;
  color: var(--tt-danger);
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
