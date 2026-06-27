import { useDialog, useMessage } from "naive-ui";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";

import { dataApi } from "@/services/api/data";
import type { CandleResult, ChartCandle, CreateDataSyncTask, DataSyncTask } from "@/types/app";

type ResearchForm = {
  exchange: string;
  symbol: string;
  interval: string;
  startTime: number | null;
  endTime: number | null;
};

export function useResearchWorkspace() {
  const route = useRoute();
  const router = useRouter();
  const dialog = useDialog();
  const message = useMessage();
  const { t } = useI18n();

  const exchange = ref(readQuery(route.query.exchange, "binance"));
  const symbol = ref(readQuery(route.query.symbol, "BTCUSDT"));
  const interval = ref(readQuery(route.query.interval, "1m"));
  const tasks = ref<DataSyncTask[]>([]);
  const candles = ref<ChartCandle[]>([]);
  const candleResult = ref<CandleResult | null>(null);
  const tasksLoading = ref(false);
  const candlesLoading = ref(false);
  const createLoading = ref(false);
  const tasksError = ref("");
  const candlesError = ref("");
  const createModalOpen = ref(false);
  const createForm = reactive<ResearchForm>({
    exchange: exchange.value,
    symbol: symbol.value,
    interval: interval.value,
    startTime: null,
    endTime: null,
  });
  const canCreateTask = computed(
    () => createForm.exchange !== "" && createForm.symbol !== "" && createForm.interval !== "",
  );

  watch([exchange, symbol, interval], () => {
    router.replace({
      name: "research",
      query: { exchange: exchange.value, symbol: symbol.value, interval: interval.value },
    });
    void loadCandles();
  });

  onMounted(() => {
    void refreshAll();
  });

  async function refreshAll() {
    await Promise.all([loadTasks(), loadCandles()]);
  }

  async function loadTasks() {
    tasksLoading.value = true;
    tasksError.value = "";
    try {
      tasks.value = await dataApi.listTasks();
    } catch (error) {
      tasksError.value = errorMessage(error, t("research.tasksLoadFailed"));
    } finally {
      tasksLoading.value = false;
    }
  }

  async function loadCandles() {
    candlesLoading.value = true;
    candlesError.value = "";
    try {
      const result = await dataApi.getCandles({
        exchange: exchange.value,
        symbol: symbol.value,
        interval: interval.value,
      });
      candleResult.value = result;
      candles.value = result.candles;
    } catch (error) {
      candles.value = [];
      candleResult.value = null;
      candlesError.value = errorMessage(error, t("research.candlesLoadFailed"));
    } finally {
      candlesLoading.value = false;
    }
  }

  function openCreateTask() {
    createForm.exchange = exchange.value;
    createForm.symbol = symbol.value;
    createForm.interval = interval.value;
    createForm.startTime = null;
    createForm.endTime = null;
    createModalOpen.value = true;
  }

  async function createTask() {
    if (!canCreateTask.value) {
      message.error(t("research.requiredFields"));
      return;
    }

    const request: CreateDataSyncTask = {
      exchange: createForm.exchange,
      symbol: createForm.symbol,
      interval: createForm.interval,
      startTime: toISOString(createForm.startTime),
      endTime: toISOString(createForm.endTime),
    };

    createLoading.value = true;
    try {
      await dataApi.createTask(request);
      createModalOpen.value = false;
      message.success(t("research.taskCreated"));
      await loadTasks();
    } catch (error) {
      message.error(errorMessage(error, t("research.taskCreateFailed")));
    } finally {
      createLoading.value = false;
    }
  }

  function selectTask(task: DataSyncTask) {
    exchange.value = task.exchange;
    symbol.value = task.symbol;
    interval.value = task.interval;
  }

  function deleteTask(task: DataSyncTask) {
    dialog.warning({
      title: t("research.deleteConfirmTitle"),
      content: `${task.exchange} / ${task.symbol} / ${task.interval}`,
      positiveText: t("common.delete"),
      negativeText: t("common.cancel"),
      onPositiveClick: () => runAction(async () => {
        await dataApi.deleteTask(task.id);
        message.success(t("research.taskDeleted"));
        await loadTasks();
      }),
    });
  }

  async function toggleRealtime(task: DataSyncTask) {
    await runAction(async () => {
      await dataApi.setRealtime(task.id, !task.realtimeEnabled);
      message.success(t("research.taskUpdated"));
      await loadTasks();
    });
  }

  async function toggleSync(task: DataSyncTask) {
    await runAction(async () => {
      await dataApi.setSync(task.id, !task.syncEnabled);
      message.success(t("research.taskUpdated"));
      await loadTasks();
    });
  }

  async function retryTask(task: DataSyncTask) {
    await runAction(async () => {
      await dataApi.retryTask(task.id);
      message.success(t("research.taskRetried"));
      await loadTasks();
    });
  }

  async function runAction(action: () => Promise<void>) {
    try {
      await action();
    } catch (error) {
      message.error(errorMessage(error, t("research.taskUpdateFailed")));
    }
  }

  return {
    canCreateTask,
    candleResult,
    candles,
    candlesError,
    candlesLoading,
    createForm,
    createLoading,
    createModalOpen,
    createTask,
    deleteTask,
    exchange,
    interval,
    loadCandles,
    loadTasks,
    openCreateTask,
    refreshAll,
    retryTask,
    selectTask,
    symbol,
    tasks,
    tasksError,
    tasksLoading,
    toggleRealtime,
    toggleSync,
  };
}

function readQuery(value: unknown, fallback: string) {
  return typeof value === "string" && value.length > 0 ? value : fallback;
}

function toISOString(value: number | null) {
  return value === null ? undefined : new Date(value).toISOString();
}

function errorMessage(error: unknown, fallback: string) {
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return fallback;
}
