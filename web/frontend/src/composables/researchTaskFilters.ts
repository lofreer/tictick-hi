import type { DataSyncTask } from "@/types/app";

export type DataHealthFilter = "all" | DataSyncTask["dataHealth"];

const dataHealthFilters: DataHealthFilter[] = ["all", "ok", "syncing", "gap", "failed", "paused", "retrying", "insufficient", "invalid"];

export function dataHealthFilterFromQuery(value: unknown): DataHealthFilter {
  return typeof value === "string" && dataHealthFilters.includes(value as DataHealthFilter) ? (value as DataHealthFilter) : "all";
}

export function dataHealthQueryValue(value: DataHealthFilter) {
  return value === "all" ? undefined : value;
}

export function taskMatchesDataHealthFilter(task: DataSyncTask, filter: DataHealthFilter) {
  return filter === "all" || task.dataHealth === filter;
}
