import type { SelectOption } from "naive-ui";
import { useMessage } from "naive-ui";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

import { strategiesApi } from "@/services/api/strategies";
import type {
  StrategyDefinition,
  StrategyParamSpec,
  StrategyParamValue,
  StrategyParamValues,
} from "@/types/app";

export type StrategyTaskMode = "backtest" | "trading";

type StrategyTaskForm = {
  exchange: string;
  symbol: string;
  interval: string;
  startTime: number | null;
  endTime: number | null;
  initialBalance: number;
  executionMode: "paper" | "live";
  riskLimitPct: number;
};

const defaultIntervals = ["1m", "5m", "15m", "1h", "4h", "1d"];

export function useStrategyTaskForm(mode: StrategyTaskMode) {
  const message = useMessage();
  const { t } = useI18n();

  const loading = ref(false);
  const submitLoading = ref(false);
  const error = ref("");
  const strategies = ref<StrategyDefinition[]>([]);
  const selectedStrategyId = ref("");
  const paramValues = ref<StrategyParamValues>({});
  const form = reactive<StrategyTaskForm>({
    exchange: "binance",
    symbol: "BTCUSDT",
    interval: "5m",
    startTime: null,
    endTime: null,
    initialBalance: 10000,
    executionMode: "paper",
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
      selectedStrategy.value !== undefined &&
      selectedStrategy.value.params.every((param) => isFilled(param, paramValues.value[param.key])),
  );

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
      message.error(t("strategy.requiredFields"));
      return;
    }

    submitLoading.value = true;
    try {
      await Promise.resolve();
      message.success(mode === "backtest" ? t("strategy.backtestReady") : t("strategy.tradingReady"));
    } finally {
      submitLoading.value = false;
    }
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

function isFilled(param: StrategyParamSpec, value: StrategyParamValue | undefined) {
  if (!param.required) {
    return true;
  }
  if (param.type === "number") {
    return typeof value === "number" && Number.isFinite(value);
  }
  if (param.type === "boolean") {
    return typeof value === "boolean";
  }
  return typeof value === "string" && value.length > 0;
}

function errorMessage(error: unknown, fallback: string) {
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return fallback;
}
