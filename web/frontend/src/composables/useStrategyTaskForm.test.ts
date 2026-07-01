import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  defaultParamValues,
  isStrategyParamValueValid,
  useStrategyTaskForm,
  type StrategyTaskMode,
} from "@/composables/useStrategyTaskForm";
import { i18n } from "@/i18n";
import { backtestsApi } from "@/services/api/backtests";
import { marketApi } from "@/services/api/market";
import { strategiesApi } from "@/services/api/strategies";
import { tradingApi } from "@/services/api/trading";
import type { StrategyDefinition, StrategyParamSpec } from "@/types/app";

const apiMocks = vi.hoisted(() => ({
  createBacktest: vi.fn(),
  createTradingTask: vi.fn(),
  listInstruments: vi.fn(),
  listStrategies: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
  success: vi.fn(),
}));

const routerMocks = vi.hoisted(() => ({
  push: vi.fn(),
}));

vi.mock("@/services/api/backtests", () => ({
  backtestsApi: { createBacktest: apiMocks.createBacktest },
}));

vi.mock("@/services/api/market", () => ({
  marketApi: { listInstruments: apiMocks.listInstruments },
}));

vi.mock("@/services/api/strategies", () => ({
  strategiesApi: { listStrategies: apiMocks.listStrategies },
}));

vi.mock("@/services/api/trading", () => ({
  tradingApi: { createTask: apiMocks.createTradingTask },
}));

vi.mock("naive-ui", () => ({
  useMessage: () => messageMocks,
}));

vi.mock("vue-router", () => ({
  useRouter: () => routerMocks,
}));

describe("strategy task form", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    apiMocks.listStrategies.mockResolvedValue([strategyDefinition()]);
    apiMocks.createBacktest.mockResolvedValue({ id: "bt_1" });
    apiMocks.createTradingTask.mockResolvedValue({ id: "tt_1" });
    apiMocks.listInstruments.mockResolvedValue([
      {
        exchange: "binance",
        symbol: "BTCUSDT",
        baseAsset: "BTC",
        quoteAsset: "USDT",
        instrumentType: "spot",
        status: "active",
        searchPriority: 1,
      },
      {
        exchange: "binance",
        symbol: "SOLUSDT",
        baseAsset: "SOL",
        quoteAsset: "USDT",
        instrumentType: "spot",
        status: "active",
        searchPriority: 20,
      },
      {
        exchange: "okx",
        symbol: "SOL-USDT",
        baseAsset: "SOL",
        quoteAsset: "USDT",
        instrumentType: "spot",
        status: "active",
        searchPriority: 20,
      },
    ]);
  });

  it("creates defaults from strategy parameter specs", () => {
    const params: StrategyParamSpec[] = [
      {
        key: "fastPeriod",
        label: "Fast period",
        type: "number",
        required: true,
        default: 12,
        options: [],
      },
      {
        key: "side",
        label: "Side",
        type: "select",
        required: true,
        options: [{ label: "Both", value: "both" }],
      },
      {
        key: "enabled",
        label: "Enabled",
        type: "boolean",
        required: false,
        options: [],
      },
    ];

    expect(defaultParamValues(params)).toEqual({
      enabled: false,
      fastPeriod: 12,
      side: "both",
    });
  });

  it("validates values against strategy parameter specs", () => {
    const numberParam: StrategyParamSpec = {
      key: "fastPeriod",
      label: "Fast period",
      type: "number",
      required: true,
      default: 12,
      min: 2,
      max: 200,
      options: [],
    };
    const selectParam: StrategyParamSpec = {
      key: "signalMode",
      label: "Signal mode",
      type: "select",
      required: true,
      options: [{ label: "Order", value: "order" }],
    };

    expect(isStrategyParamValueValid(numberParam, 12)).toBe(true);
    expect(isStrategyParamValueValid(numberParam, 1)).toBe(false);
    expect(isStrategyParamValueValid(selectParam, "order")).toBe(true);
    expect(isStrategyParamValueValid(selectParam, "webhook")).toBe(false);
  });

  it("normalizes arbitrary valid backtest symbols before submit", async () => {
    const taskForm = mountTaskForm("backtest");
    await flushPromises();

    taskForm.form.symbol = " solusdt ";
    await flushPromises();

    expect(taskForm.form.symbol).toBe("SOLUSDT");

    await taskForm.submit();
    await flushPromises();

    expect(backtestsApi.createBacktest).toHaveBeenCalledWith(
      expect.objectContaining({
        exchange: "binance",
        symbol: "SOLUSDT",
      }),
    );
    expect(marketApi.listInstruments).toHaveBeenCalledWith({ exchange: "binance", limit: 1, q: "SOLUSDT", status: "all" });
  });

  it("blocks backtest submit when the symbol format does not match the exchange", async () => {
    const taskForm = mountTaskForm("backtest");
    await flushPromises();

    taskForm.form.symbol = "BTC-USDT";
    await flushPromises();

    expect(taskForm.canSubmit.value).toBe(false);

    await taskForm.submit();

    expect(backtestsApi.createBacktest).not.toHaveBeenCalled();
    expect(messageMocks.error).toHaveBeenCalledWith("交易对格式不符合当前交易所。");
  });

  it("coerces the symbol and suggestions when the exchange changes", async () => {
    const taskForm = mountTaskForm("backtest");
    await flushPromises();

    taskForm.form.exchange = "okx";
    await flushPromises();

    expect(taskForm.form.symbol).toBe("BTC-USDT");
    expect(taskForm.symbolOptions.value.map((option) => option.value)).toEqual(["BTC-USDT", "ETH-USDT"]);
  });

  it("normalizes valid trading task symbols before submit", async () => {
    const taskForm = mountTaskForm("trading");
    await flushPromises();

    taskForm.form.exchange = "okx";
    await flushPromises();
    taskForm.form.symbol = " sol-usdt ";
    await flushPromises();

    expect(taskForm.canSubmit.value).toBe(true);

    await taskForm.submit();
    await flushPromises();

    expect(tradingApi.createTask).toHaveBeenCalledWith(
      expect.objectContaining({
        exchange: "okx",
        symbol: "SOL-USDT",
      }),
    );
    expect(marketApi.listInstruments).toHaveBeenCalledWith({ exchange: "okx", limit: 1, q: "SOL-USDT", status: "all" });
  });

  it("blocks backtest submit when the catalog symbol is inactive", async () => {
    apiMocks.listInstruments.mockImplementation(async ({ q }: { q?: string }) =>
      q === "SOLUSDT"
        ? [marketInstrument("binance", "SOLUSDT", "inactive")]
        : [marketInstrument("binance", "BTCUSDT", "active")],
    );
    const taskForm = mountTaskForm("backtest");
    await flushPromises();

    taskForm.form.symbol = "SOLUSDT";
    await flushPromises();
    await taskForm.refreshMarketInstrumentCatalogStatus();
    await flushPromises();

    expect(taskForm.marketCatalogStatus.value).toBe("inactive");
    expect(taskForm.canSubmit.value).toBe(false);

    await taskForm.submit();
    await flushPromises();

    expect(backtestsApi.createBacktest).not.toHaveBeenCalled();
    expect(messageMocks.error).toHaveBeenCalledWith("交易对已不在当前交易所 active 目录中，请刷新交易对或选择仍可用的标的。");
  });

  it("blocks trading submit when the catalog symbol is missing", async () => {
    apiMocks.listInstruments.mockImplementation(async ({ q }: { q?: string }) =>
      q === "DOGEUSDT" ? [] : [marketInstrument("binance", "BTCUSDT", "active")],
    );
    const taskForm = mountTaskForm("trading");
    await flushPromises();

    taskForm.form.symbol = "DOGEUSDT";
    await flushPromises();
    await taskForm.refreshMarketInstrumentCatalogStatus();
    await flushPromises();

    expect(taskForm.marketCatalogStatus.value).toBe("missing");
    expect(taskForm.canSubmit.value).toBe(false);

    await taskForm.submit();
    await flushPromises();

    expect(tradingApi.createTask).not.toHaveBeenCalled();
    expect(messageMocks.error).toHaveBeenCalledWith("交易对不在当前交易所可用目录中，请先刷新交易对或更换标的。");
  });

  it("surfaces catalog validation failures before task creation", async () => {
    apiMocks.listInstruments.mockRejectedValue(new Error("network"));
    const taskForm = mountTaskForm("backtest");
    await flushPromises();
    await taskForm.refreshMarketInstrumentCatalogStatus();
    await flushPromises();

    expect(taskForm.marketCatalogStatus.value).toBe("unknown");
    expect(taskForm.marketCatalogError.value).toBe("校验交易对目录失败，请稍后重试。");
    expect(taskForm.canSubmit.value).toBe(false);

    await taskForm.submit();
    await flushPromises();

    expect(backtestsApi.createBacktest).not.toHaveBeenCalled();
    expect(messageMocks.error).toHaveBeenCalledWith("校验交易对目录失败，请稍后重试。");
  });
});

function mountTaskForm(mode: StrategyTaskMode) {
  const holder: { taskForm?: ReturnType<typeof useStrategyTaskForm> } = {};
  mount(
    {
      template: "<div />",
      setup() {
        holder.taskForm = useStrategyTaskForm(mode);
        return {};
      },
    },
    {
      global: {
        plugins: [i18n],
      },
    },
  );
  if (!holder.taskForm) {
    throw new Error("strategy task form was not mounted");
  }
  return holder.taskForm;
}

function strategyDefinition(): StrategyDefinition {
  return {
    id: "ema-cross",
    name: "EMA Cross",
    version: "v1",
    description: "EMA strategy",
    supportedIntervals: ["1m", "5m"],
    supportedIntents: ["order"],
    params: [],
  };
}

function marketInstrument(exchange: string, symbol: string, status: "active" | "inactive") {
  const normalized = symbol.replace("-", "");
  return {
    exchange,
    symbol,
    baseAsset: normalized.replace(/USDT$/, "") || normalized,
    quoteAsset: "USDT",
    instrumentType: "spot",
    status,
    exchangeStatus: status === "active" ? "TRADING" : "BREAK",
    searchPriority: status === "active" ? 1 : 20,
  };
}
