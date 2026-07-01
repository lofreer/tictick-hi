import { errorMessage } from "@/composables/researchWorkspaceHelpers";
import { dataApi } from "@/services/api/data";
import type { DataSyncTask } from "@/types/app";

type TaskCommandOptions = {
  loadTasks: () => Promise<unknown>;
  message: {
    error: (content: string) => void;
    success: (content: string) => void;
  };
  t: (key: string) => string;
  task: DataSyncTask;
};

export async function toggleResearchRealtimeTask(options: TaskCommandOptions) {
  if (!options.task.realtimeEnabled && !taskMarketActive(options.task)) {
    options.message.error(options.t("research.taskMarketNotActive"));
    return;
  }
  await runTaskCommand(options, async () => {
    await dataApi.setRealtime(options.task.id, !options.task.realtimeEnabled);
  });
}

export async function toggleResearchSyncTask(options: TaskCommandOptions) {
  if (!options.task.syncEnabled && !taskMarketActive(options.task)) {
    options.message.error(options.t("research.taskMarketNotActive"));
    return;
  }
  await runTaskCommand(options, async () => {
    await dataApi.setSync(options.task.id, !options.task.syncEnabled);
  });
}

export async function retryResearchSyncTask(options: TaskCommandOptions) {
  if (!taskMarketActive(options.task)) {
    options.message.error(options.t("research.taskMarketNotActive"));
    return;
  }
  await runTaskCommand(options, async () => {
    await dataApi.retryTask(options.task.id);
  }, "research.taskRetried");
}

async function runTaskCommand(
  options: TaskCommandOptions,
  command: () => Promise<void>,
  successKey = "research.taskUpdated",
) {
  try {
    await command();
    options.message.success(options.t(successKey));
    await options.loadTasks();
  } catch (error) {
    options.message.error(errorMessage(error, options.t("research.taskUpdateFailed")));
  }
}

function taskMarketActive(task: DataSyncTask) {
  return task.marketStatus === "active";
}
