import { useDialog, useMessage } from "naive-ui";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";

import { dataApi } from "@/services/api/data";
import type { CandleResult, ChartCandle, CreateDataSyncTask, DataSyncGapList, DataSyncTask } from "@/types/app";
import {
  coerceSymbolForExchange,
  isSymbolFormatForExchange,
  normalizeSymbolInput,
} from "@/utils/marketSymbols";

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

  const initialExchange = readQuery(route.query.exchange, "binance");
  const exchange = ref(initialExchange);
  const symbol = ref(coerceSymbolForExchange(initialExchange, readQuery(route.query.symbol, "BTCUSDT")));
  const interval = ref(readQuery(route.query.interval, "1m"));
  const tasks = ref<DataSyncTask[]>([]);
  const candles = ref<ChartCandle[]>([]);
  const candleResult = ref<CandleResult | null>(null);
  const tasksLoading = ref(false);
  const candlesLoading = ref(false);
  const createLoading = ref(false);
  const repairGapLoading = ref(false);
  const repairTaskGapsLoadingId = ref("");
  const gapDetailsModalOpen = ref(false);
  const gapDetailsLoading = ref(false);
  const gapDetailsError = ref("");
  const gapDetailsTask = ref<DataSyncTask | null>(null);
  const gapDetails = ref<DataSyncGapList | null>(null);
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
    () =>
      createForm.exchange !== "" &&
      createForm.symbol !== "" &&
      createForm.interval !== "" &&
      isSymbolFormatForExchange(createForm.exchange, createForm.symbol),
  );
  const firstRepairableGap = computed(() => candleResult.value?.gaps[0] ?? null);
  const canRepairGap = computed(() => firstRepairableGap.value !== null);

  watch(exchange, (nextExchange) => {
    symbol.value = coerceSymbolForExchange(nextExchange, symbol.value);
  });

  watch(symbol, (nextSymbol) => {
    const normalized = normalizeSymbolInput(nextSymbol);
    if (normalized !== nextSymbol) {
      symbol.value = normalized;
    }
  });

  watch(
    () => createForm.exchange,
    (nextExchange) => {
      createForm.symbol = coerceSymbolForExchange(nextExchange, createForm.symbol);
    },
  );

  watch(
    () => createForm.symbol,
    (nextSymbol) => {
      const normalized = normalizeSymbolInput(nextSymbol);
      if (normalized !== nextSymbol) {
        createForm.symbol = normalized;
      }
    },
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
      if (!isSymbolFormatForExchange(exchange.value, symbol.value)) {
        candles.value = [];
        candleResult.value = null;
        candlesError.value = t("research.invalidSymbolFormat");
        return;
      }

      const result = await dataApi.getCandles({
        exchange: exchange.value,
        symbol: normalizeSymbolInput(symbol.value),
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
    createForm.symbol = coerceSymbolForExchange(createForm.exchange, symbol.value);
    createForm.interval = interval.value;
    createForm.startTime = null;
    createForm.endTime = null;
    createModalOpen.value = true;
  }

  async function createTask() {
    if (!canCreateTask.value) {
      message.error(
        createForm.exchange && createForm.symbol && createForm.interval
          ? t("research.invalidSymbolFormat")
          : t("research.requiredFields"),
      );
      return;
    }

    const request: CreateDataSyncTask = {
      exchange: createForm.exchange,
      symbol: normalizeSymbolInput(createForm.symbol),
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

  async function viewTaskGaps(task: DataSyncTask) {
    gapDetailsTask.value = task;
    gapDetails.value = null;
    gapDetailsError.value = "";
    gapDetailsModalOpen.value = true;
    gapDetailsLoading.value = true;
    try {
      gapDetails.value = await dataApi.getTaskGaps(task.id);
    } catch (error) {
      gapDetailsError.value = errorMessage(error, t("research.gapDetailsLoadFailed"));
    } finally {
      gapDetailsLoading.value = false;
    }
  }

  async function repairTaskGaps(task: DataSyncTask) {
    if (repairTaskGapsLoadingId.value) {
      return;
    }
    repairTaskGapsLoadingId.value = task.id;
    try {
      const result = await dataApi.repairTaskGaps(task.id);
      if (result.createdTasks.length > 0) {
        message.success(t("research.taskGapRepairQueued", { count: result.createdTasks.length }));
      } else if (result.skippedExisting > 0) {
        message.success(t("research.taskGapRepairAlreadyQueued"));
      } else {
        message.success(t("research.noRepairableTaskGaps"));
      }
      await loadTasks();
    } catch (error) {
      message.error(errorMessage(error, t("research.taskGapRepairFailed")));
    } finally {
      repairTaskGapsLoadingId.value = "";
    }
  }

  async function repairFirstGap() {
    const gap = firstRepairableGap.value;
    if (!gap) {
      message.error(t("research.noRepairableGap"));
      return;
    }
    const repairInterval = candleResult.value?.baseInterval || interval.value;
    const request: CreateDataSyncTask = {
      exchange: exchange.value,
      symbol: normalizeSymbolInput(symbol.value),
      interval: repairInterval,
      startTime: gap.from,
      endTime: gap.to,
    };

    repairGapLoading.value = true;
    try {
      const task = await dataApi.createTask(request);
      await dataApi.setSync(task.id, true);
      message.success(t("research.gapRepairQueued"));
      await loadTasks();
    } catch (error) {
      message.error(errorMessage(error, t("research.gapRepairFailed")));
    } finally {
      repairGapLoading.value = false;
    }
  }

  async function runAction(action: () => Promise<void>, fallback = t("research.taskUpdateFailed")) {
    try {
      await action();
    } catch (error) {
      message.error(errorMessage(error, fallback));
    }
  }

  return {
    canCreateTask,
    canRepairGap,
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
    gapDetails,
    gapDetailsError,
    gapDetailsLoading,
    gapDetailsModalOpen,
    gapDetailsTask,
    interval,
    loadCandles,
    loadTasks,
    openCreateTask,
    repairFirstGap,
    repairGapLoading,
    repairTaskGaps,
    repairTaskGapsLoadingId,
    refreshAll,
    retryTask,
    selectTask,
    symbol,
    tasks,
    tasksError,
    tasksLoading,
    toggleRealtime,
    toggleSync,
    viewTaskGaps,
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
