import type { DataSyncTask } from "@/types/app";

const terminalStatuses = new Set<DataSyncTask["status"]>(["succeeded", "failed", "cancelled", "paused"]);

export function repairTaskSettleKey(taskIds: string[]) {
  return taskIds.join("|");
}

export function repairTasksSettled(tasks: DataSyncTask[] | undefined, taskIds: string[]) {
  if (taskIds.length === 0 || !tasks) return false;
  const taskById = new Map(tasks.map((task) => [task.id, task]));
  return taskIds.every((id) => {
    const task = taskById.get(id);
    return Boolean(task && terminalStatuses.has(task.status));
  });
}
