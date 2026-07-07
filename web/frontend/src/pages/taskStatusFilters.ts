export type TaskStatusFilter = "all" | "pending" | "running" | "succeeded" | "failed" | "cancelled";

const taskStatusFilters: TaskStatusFilter[] = ["all", "pending", "running", "succeeded", "failed", "cancelled"];

export function taskStatusFilterFromQuery(value: unknown): TaskStatusFilter {
  return typeof value === "string" && taskStatusFilters.includes(value as TaskStatusFilter) ? (value as TaskStatusFilter) : "all";
}

export function taskStatusQueryValue(value: TaskStatusFilter) {
  return value === "all" ? undefined : value;
}

export function taskMatchesStatusFilter(task: { status: string }, filter: TaskStatusFilter) {
  return filter === "all" || task.status === filter;
}
