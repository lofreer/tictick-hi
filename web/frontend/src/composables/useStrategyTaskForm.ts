import type { SelectOption } from "naive-ui";
import { useMessage } from "naive-ui";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";

import { backtestsApi } from "@/services/api/backtests";
import { strategiesApi } from "@/services/api/strategies";
import { tradingApi } from "@/services/api/trading";
import type {
  BacktestTriggerMode,
  StrategyDefinition,
  StrategyParamSpec,
  StrategyParamValue,
  StrategyParamValues,
} from "@/types/app";
import {
  coerceSymbolForExchange,
  isSymbolFormatForExchange,
  normalizeSymbolInput,
  symbolOptionsForExchange,
} from "@/utils/marketSymbols";

export type StrategyTaskMode = "backtest" | "trading";

type StrategyTaskForm = {
  name: string;
  exchange: string;
  symbol: string;
  interval: string;
  startTime: number | null;
  endTime: number | null;
  initialBalance: number;
  feeBps: number;
  slippageBps: number;
  triggerMode: BacktestTriggerMode;
  executionMode: "paper" | "live";
  accountId: string;
  orderIntent: "execute" | "notify";
  notificationChannel: string;
  liveExecutionConfirmed: boolean;
  riskLimitPct: number;
};

const defaultIntervals = ["1m", "5m", "15m", "1h", "4h", "1d"];

export function useStrategyTaskForm(mode: StrategyTaskMode) {
  const message = useMessage();
  const router = useRouter();
  const { t } = useI18n();

  const now = Date.now();
  const loading = ref(false);
  const submitLoading = ref(false);
  const error = ref("");
  const strategies = ref<StrategyDefinition[]>([]);
  const selectedStrategyId = ref("");
  const paramValues = ref<StrategyParamValues>({});
  const form = reactive<StrategyTaskForm>({
    name: "BTCUSDT backtest",
    exchange: "binance",
    symbol: "BTCUSDT",
    interval: "5m",
    startTime: now - 30 * 24 * 60 * 60 * 1000,
    endTime: now,
    initialBalance: 10000,
    feeBps: 1,
    slippageBps: 0,
    triggerMode: "closed_candle",
    executionMode: "paper",
    accountId: "paper",
    orderIntent: "execute",
    notificationChannel: "default",
    liveExecutionConfirmed: false,
    riskLimitPct: 10,
  });

  const selectedStrategy = computed(() =>
    strategies.value.find((strategy) => strategy.id === selectedStrategyId.value),
  );
  const strategyOptions = computed<SelectOption[]>(() =>
    strategies.value.map((strategy) => ({
      label: `${strategy.name} ${strategy.version}`,
      value: strategy.id,
    })),
  );
  const intervalOptions = computed<SelectOption[]>(() =>
    supportedIntervals.value.map((interval) => ({ label: interval, value: interval })),
  );
  const canSubmit = computed(
    () =>
      form.exchange !== "" &&
      form.symbol !== "" &&
      form.interval !== "" &&
      isSymbolFormatForExchange(form.exchange, form.symbol) &&
      selectedStrategy.value !== undefined &&
      taskFieldsValid() &&
      selectedStrategy.value.params.every((param) => isStrategyParamValueValid(param, paramValues.value[param.key])),
  );
  const symbolOptions = computed(() => symbolOptionsForExchange(form.exchange));

  const supportedIntervals = computed(() => {
    const intervals = selectedStrategy.value?.supportedIntervals ?? [];
    return intervals.length > 0 ? intervals : defaultIntervals;
  });

  watch(selectedStrategy, (strategy) => {
    if (strategy === undefined) {
      paramValues.value = {};
      return;
    }
    if (!supportedIntervals.value.includes(form.interval)) {
      form.interval = supportedIntervals.value[0] ?? "5m";
    }
    paramValues.value = defaultParamValues(strategy.params);
  });

  watch(
    () => form.exchange,
    (exchange) => {
      form.symbol = coerceSymbolForExchange(exchange, form.symbol);
    },
  );

  watch(
    () => form.symbol,
    (symbol) => {
      const normalized = normalizeSymbolInput(symbol);
      if (normalized !== symbol) {
        form.symbol = normalized;
      }
    },
  );

  watch(
    () => form.executionMode,
    (mode) => {
      if (mode === "live" && form.orderIntent === "execute") {
        form.orderIntent = "notify";
        form.liveExecutionConfirmed = false;
      }
      if (mode === "paper" && form.accountId === "") {
        form.accountId = "paper";
      }
    },
  );

  onMounted(() => {
    void loadStrategies();
  });

  async function loadStrategies() {
    loading.value = true;
    error.value = "";
    try {
      strategies.value = await strategiesApi.listStrategies();
      if (strategies.value.length > 0 && !strategies.value.some((item) => item.id === selectedStrategyId.value)) {
        selectedStrategyId.value = strategies.value[0].id;
      }
    } catch (loadError) {
      strategies.value = [];
      selectedStrategyId.value = "";
      error.value = errorMessage(loadError, t("strategy.loadFailed"));
    } finally {
      loading.value = false;
    }
  }

  async function submit() {
    if (!canSubmit.value || selectedStrategy.value === undefined) {
      message.error(
        form.exchange && form.symbol && form.interval && !isSymbolFormatForExchange(form.exchange, form.symbol)
          ? t("research.invalidSymbolFormat")
          : t("strategy.requiredFields"),
      );
      return;
    }

    submitLoading.value = true;
    try {
      if (mode === "backtest") {
        const created = await backtestsApi.createBacktest({
          name: form.name,
          exchange: form.exchange,
          symbol: normalizeSymbolInput(form.symbol),
          interval: form.interval,
          startTime: toISOString(form.startTime),
          endTime: toISOString(form.endTime),
          strategyId: selectedStrategy.value.id,
          strategyParams: compactParamValues(paramValues.value),
          initialBalance: String(form.initialBalance),
          feeBps: String(form.feeBps),
          slippageBps: String(form.slippageBps),
          triggerMode: form.triggerMode,
        });
        message.success(t("strategy.backtestCreated"));
        await router.push({ name: "backtests-detail", params: { id: created.id } });
      } else {
        const created = await tradingApi.createTask({
          name: form.name,
          type: form.executionMode,
          exchange: form.exchange,
          accountId: form.accountId,
          symbol: normalizeSymbolInput(form.symbol),
          interval: form.interval,
          strategyId: selectedStrategy.value.id,
          strategyParams: compactParamValues(paramValues.value),
          intentPolicy: {
            orderIntent: form.orderIntent,
            notificationChannel: form.notificationChannel,
            liveExecutionConfirmed: form.liveExecutionConfirmed,
            riskLimitPct: form.riskLimitPct,
          },
        });
        message.success(t("strategy.tradingCreated"));
        await router.push({ name: "trading-detail", params: { id: created.id } });
      }
    } catch (submitError) {
      const fallback = mode === "backtest" ? t("strategy.backtestCreateFailed") : t("strategy.taskSubmitFailed");
      message.error(errorMessage(submitError, fallback));
    } finally {
      submitLoading.value = false;
    }
  }

  function taskFieldsValid() {
    if (mode !== "backtest") {
      return (
        form.name !== "" &&
        form.accountId !== "" &&
        form.riskLimitPct >= 0 &&
        form.riskLimitPct <= 100 &&
        (form.executionMode !== "live" || form.orderIntent !== "execute")
      );
    }
    return (
      form.name !== "" &&
      form.startTime !== null &&
      form.endTime !== null &&
      form.startTime < form.endTime &&
      form.initialBalance > 0 &&
      form.feeBps >= 0 &&
      form.slippageBps >= 0
    );
  }

  return {
    canSubmit,
    error,
    form,
    intervalOptions,
    loadStrategies,
    loading,
    paramValues,
    selectedStrategy,
    selectedStrategyId,
    strategies,
    strategyOptions,
    submit,
    submitLoading,
    symbolOptions,
    supportedIntervals,
  };
}

export function defaultParamValues(params: StrategyParamSpec[]) {
  return params.reduce<StrategyParamValues>((values, param) => {
    values[param.key] = normalizeDefaultValue(param);
    return values;
  }, {});
}

function normalizeDefaultValue(param: StrategyParamSpec): StrategyParamValue {
  if (param.default !== undefined) {
    return param.default;
  }
  if (param.type === "number") {
    return param.min ?? 0;
  }
  if (param.type === "select") {
    return param.options[0]?.value ?? "";
  }
  if (param.type === "boolean") {
    return false;
  }
  return "";
}

export function isStrategyParamValueValid(param: StrategyParamSpec, value: StrategyParamValue | undefined) {
  const empty = value === undefined || value === null || value === "";
  if (empty && !param.required) {
    return true;
  }
  if (empty) {
    return false;
  }
  if (param.type === "number") {
    if (typeof value !== "number" || !Number.isFinite(value)) {
      return false;
    }
    if (param.min !== undefined && value < param.min) {
      return false;
    }
    if (param.max !== undefined && value > param.max) {
      return false;
    }
    return true;
  }
  if (param.type === "boolean") {
    return typeof value === "boolean";
  }
  if (param.type === "select") {
    if (typeof value !== "string" || value.length === 0) {
      return false;
    }
    return param.options.length === 0 || param.options.some((option) => option.value === value);
  }
  return typeof value === "string" && value.length > 0;
}

function toISOString(value: number | null) {
  return value === null ? undefined : new Date(value).toISOString();
}

function compactParamValues(values: StrategyParamValues) {
  return Object.fromEntries(Object.entries(values).filter(([, value]) => value !== null)) as StrategyParamValues;
}

function errorMessage(error: unknown, fallback: string) {
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return fallback;
}
