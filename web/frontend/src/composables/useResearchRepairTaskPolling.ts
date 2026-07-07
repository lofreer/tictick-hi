import { onBeforeUnmount } from "vue";

import type { DataSyncTask, TaskStatus } from "@/types/app";

type PollingOptions = {
  intervalMs?: number;
  maxAttempts?: number;
};

type StartPollingOptions = {
  immediate?: boolean;
  onExhausted?: () => Promise<void> | void;
  onSettled?: () => Promise<void> | void;
  repairTaskIds?: string[];
  snapshotTaskIds?: string[];
};

type LoadRepairTasks = (ids: string[]) => Promise<DataSyncTask[] | void>;

const terminalStatuses = new Set<TaskStatus>(["succeeded", "failed", "cancelled", "paused"]);

export function useResearchRepairTaskPolling(loadRepairTasks: LoadRepairTasks, options: PollingOptions = {}) {
  const intervalMs = options.intervalMs ?? 4_000;
  const maxAttempts = options.maxAttempts ?? 6;
  let timer: number | null = null;
  let generation = 0;
  let attempts = 0;
  let onExhausted: (() => Promise<void> | void) | null = null;
  let onSettled: (() => Promise<void> | void) | null = null;
  let repairTaskIds = new Set<string>();
  let snapshotTaskIds = new Set<string>();

  function clearTimer() {
    if (timer === null) return;
    window.clearTimeout(timer);
    timer = null;
  }

  function stopRepairTaskPolling() {
    generation += 1;
    attempts = 0;
    onExhausted = null;
    onSettled = null;
    repairTaskIds = new Set();
    snapshotTaskIds = new Set();
    clearTimer();
  }

  function startRepairTaskPolling(startOptions: StartPollingOptions = {}) {
    generation += 1;
    attempts = 0;
    onExhausted = startOptions.onExhausted ?? null;
    onSettled = startOptions.onSettled ?? null;
    repairTaskIds = new Set(startOptions.repairTaskIds ?? []);
    snapshotTaskIds = new Set(startOptions.snapshotTaskIds ?? []);
    clearTimer();
    if (maxAttempts <= 0) return;

    const currentGeneration = generation;
    if (startOptions.immediate === false) {
      scheduleNext(currentGeneration);
      return;
    }
    void runAttempt(currentGeneration);
  }

  function scheduleNext(currentGeneration: number) {
    clearTimer();
    timer = window.setTimeout(() => {
      timer = null;
      void runAttempt(currentGeneration);
    }, intervalMs);
  }

  async function runAttempt(currentGeneration: number) {
    if (currentGeneration !== generation) return;
    attempts += 1;
    let latestTasks: DataSyncTask[] | void;
    try {
      latestTasks = await loadRepairTasks(pollingSnapshotTaskIds(repairTaskIds, snapshotTaskIds));
    } catch {
      latestTasks = undefined;
    }
    if (currentGeneration !== generation) return;
    if (repairTaskIds.size > 0 && latestTasks && watchedRepairTasksSettled(latestTasks, repairTaskIds)) {
      await runCallback(onSettled);
      if (currentGeneration === generation) stopRepairTaskPolling();
      return;
    }
    if (attempts >= maxAttempts) {
      await runCallback(onExhausted);
      if (currentGeneration === generation) stopRepairTaskPolling();
      return;
    }
    scheduleNext(currentGeneration);
  }

  onBeforeUnmount(stopRepairTaskPolling);

  return {
    startRepairTaskPolling,
    stopRepairTaskPolling,
  };
}

function watchedRepairTasksSettled(tasks: DataSyncTask[], ids: Set<string>) {
  const watchedTasks = tasks.filter((task) => ids.has(task.id));
  return watchedTasks.length === ids.size && watchedTasks.every((task) => terminalStatuses.has(task.status));
}

function pollingSnapshotTaskIds(repairTaskIds: Set<string>, snapshotTaskIds: Set<string>) {
  return [...new Set([...repairTaskIds, ...snapshotTaskIds])];
}

async function runCallback(callback: (() => Promise<void> | void) | null) {
  if (!callback) return;
  await callback();
}
