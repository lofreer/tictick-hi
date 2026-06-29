import { useDialog, useMessage } from "naive-ui";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";

import {
  candleQuery,
  candleWindowForTimeRange,
  canLoadNextCandleWindow,
  canLoadPreviousCandleWindow,
  errorMessage,
  initialResearchForm,
  nextCandleWindow,
  previousCandleWindow,
  readOptionalQuery,
  readQuery,
  repairSourceTask,
  researchQuery,
  selectedTaskMatchesMarket,
  type ResearchTimeRangePreset,
} from "@/composables/researchWorkspaceHelpers";
import { researchChartGapMarkers } from "@/composables/researchChartGapMarkers";
import { repairChartGap } from "@/composables/researchGapRepairActions";
import {
  retryResearchSyncTask,
  toggleResearchRealtimeTask,
  toggleResearchSyncTask,
} from "@/composables/researchTaskCommandActions";
import { createResearchDataSyncTask } from "@/composables/researchTaskCreateActions";
import { useMarketInstrumentSyncStatuses } from "@/composables/useMarketInstrumentSyncStatuses";
import { useMarketSymbolNormalization } from "@/composables/useMarketSymbolNormalization";
import { dataApi } from "@/services/api/data";
import type { CandleResult, ChartCandle, DataSyncGapList, DataSyncTask } from "@/types/app";
import { coerceSymbolForExchange, isSymbolFormatForExchange, normalizeSymbolInput } from "@/utils/marketSymbols";

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
  const initialCandleWindowCursor = readOptionalQuery(route.query.cursor);
  const candleWindowCursor = ref(initialCandleWindowCursor);
  const candleWindowFrom = ref(initialCandleWindowCursor ? "" : readOptionalQuery(route.query.from));
  const candleWindowTo = ref(initialCandleWindowCursor ? "" : readOptionalQuery(route.query.to));
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
  const selectedChartTask = ref<DataSyncTask | null>(null);
  const createForm = reactive(initialResearchForm(exchange.value, symbol.value, interval.value));
  const {
    create: createMarketInstrumentSyncStatus,
    current: currentMarketInstrumentSyncStatus,
    error: marketInstrumentSyncStatusError,
    load: loadMarketInstrumentSyncStatuses,
  } = useMarketInstrumentSyncStatuses(exchange, computed(() => createForm.exchange), (error) =>
    errorMessage(error, t("research.instrumentCatalogStatusLoadFailed")),
  );
  const canCreateTask = computed(
    () =>
      createForm.exchange !== "" &&
      createForm.symbol !== "" &&
      createForm.interval !== "" &&
      isSymbolFormatForExchange(createForm.exchange, createForm.symbol),
  );
  const firstRepairableGap = computed(() => candleResult.value?.gaps[0] ?? null);
  const canRepairGap = computed(() => firstRepairableGap.value !== null);
  const chartMarkers = computed(() => researchChartGapMarkers(candles.value, candleResult.value, t));
  const selectedRepairSourceTask = computed(() => {
    return repairSourceTask(
      selectedChartTask.value,
      exchange.value,
      normalizeSymbolInput(symbol.value),
      candleResult.value?.baseInterval || interval.value,
    );
  });
  const canLoadPreviousCandles = computed(() => canLoadPreviousCandleWindow(candleResult.value));
  const canLoadNextCandles = computed(() => canLoadNextCandleWindow(candleResult.value));
  useMarketSymbolNormalization(exchange, symbol, createForm);

  watch([exchange, symbol, interval, candleWindowFrom, candleWindowTo, candleWindowCursor], (nextValues, previousValues) => {
    const contextChanged =
      nextValues[0] !== previousValues[0] || nextValues[1] !== previousValues[1] || nextValues[2] !== previousValues[2];
    if (contextChanged && (candleWindowFrom.value || candleWindowTo.value || candleWindowCursor.value)) {
      candleWindowFrom.value = "";
      candleWindowTo.value = "";
      candleWindowCursor.value = "";
      return;
    }
    if (
      contextChanged &&
      selectedChartTask.value &&
      !selectedTaskMatchesMarket(selectedChartTask.value, exchange.value, normalizeSymbolInput(symbol.value))
    ) {
      selectedChartTask.value = null;
    }
    replaceResearchQuery();
    void loadCandles();
  });

  onMounted(() => void refreshAll());

  async function refreshAll() {
    await Promise.all([loadTasks(), loadCandles(), loadMarketInstrumentSyncStatuses()]);
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

      const result = await dataApi.getCandles(candleQuery(
        exchange.value,
        normalizeSymbolInput(symbol.value),
        interval.value,
        candleWindowFrom.value,
        candleWindowTo.value,
        candleWindowCursor.value,
      ));
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

    createLoading.value = true;
    try {
      await createResearchDataSyncTask({
        closeCreateModal: () => {
          createModalOpen.value = false;
        },
        form: createForm,
        loadTasks,
        message,
        t,
      });
    } finally {
      createLoading.value = false;
    }
  }

  function selectTask(task: DataSyncTask) {
    exchange.value = task.exchange;
    symbol.value = task.symbol;
    interval.value = task.interval;
    candleWindowFrom.value = "";
    candleWindowTo.value = "";
    candleWindowCursor.value = "";
    selectedChartTask.value = task;
  }

  function loadPreviousCandles() { applyCandleWindow(previousCandleWindow(candleResult.value)); }

  function loadNextCandles() { applyCandleWindow(nextCandleWindow(candleResult.value)); }

  function applyTimeRange(preset: ResearchTimeRangePreset, now = new Date()) {
    const previousWindow = `${candleWindowCursor.value}|${candleWindowFrom.value}|${candleWindowTo.value}`;
    applyCandleWindow(candleWindowForTimeRange(preset, now));
    if (previousWindow === `${candleWindowCursor.value}|${candleWindowFrom.value}|${candleWindowTo.value}`) void loadCandles();
  }

  function applyCandleWindow(window: { cursor?: string; from?: string; to?: string } | null) {
    if (!window) return;
    candleWindowCursor.value = window.cursor ?? "";
    candleWindowFrom.value = window.from ?? "";
    candleWindowTo.value = window.to ?? "";
  }

  function replaceResearchQuery() {
    const query = researchQuery(exchange.value, symbol.value, interval.value, candleWindowFrom.value, candleWindowTo.value, candleWindowCursor.value);
    router.replace({
      name: "research",
      query,
    });
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
    await toggleResearchRealtimeTask({ loadTasks, message, t, task });
  }

  async function toggleSync(task: DataSyncTask) {
    await toggleResearchSyncTask({ loadTasks, message, t, task });
  }

  async function retryTask(task: DataSyncTask) {
    await retryResearchSyncTask({ loadTasks, message, t, task });
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

    repairGapLoading.value = true;
    try {
      await repairChartGap({
        exchange: exchange.value,
        gap,
        loadTasks,
        onSuccess: (messageKey) => message.success(t(messageKey)),
        repairInterval,
        sourceTask: selectedRepairSourceTask.value,
        symbol: normalizeSymbolInput(symbol.value),
      });
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
    chartMarkers,
    canLoadNextCandles,
    canLoadPreviousCandles,
    createForm,
    createLoading,
    createModalOpen,
    createMarketInstrumentSyncStatus,
    createTask,
    currentMarketInstrumentSyncStatus,
    deleteTask,
    exchange,
    applyTimeRange,
    gapDetails,
    gapDetailsError,
    gapDetailsLoading,
    gapDetailsModalOpen,
    gapDetailsTask,
    interval,
    loadCandles,
    loadMarketInstrumentSyncStatuses,
    loadNextCandles,
    loadPreviousCandles,
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
    marketInstrumentSyncStatusError,
    tasks,
    tasksError,
    tasksLoading,
    toggleRealtime,
    toggleSync,
    viewTaskGaps,
  };
}
