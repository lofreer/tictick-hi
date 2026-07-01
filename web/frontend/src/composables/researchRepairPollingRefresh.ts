import type { Ref } from "vue";

import type { DataSyncTask } from "@/types/app";

type RefreshAfterRepairPollingOptions = {
  gapDetailsTask: Ref<DataSyncTask | null>;
  loadCandles: () => Promise<void>;
  task?: DataSyncTask;
  viewTaskGaps: (task: DataSyncTask, options?: { resetRepairResult?: boolean }) => Promise<void>;
};

export async function refreshAfterRepairPolling(options: RefreshAfterRepairPollingOptions) {
  const { gapDetailsTask, loadCandles, task, viewTaskGaps } = options;
  await loadCandles();
  if (task && gapDetailsTask.value?.id === task.id) {
    await viewTaskGaps(task, { resetRepairResult: false });
  }
}
