import type { RouteLocationRaw } from "vue-router";
import type { TagProps } from "naive-ui";

import { backtestsApi } from "@/services/api/backtests";
import { tradingApi } from "@/services/api/trading";
import type { BacktestOrder, BacktestTask, Order, StrategyIntent, TradingTask } from "@/types/app";

const overviewFactTaskLimit = 4;

export type OverviewFactSource = "backtest" | "trading";

export type OverviewIntentFact = {
  intent: StrategyIntent;
  market: string;
  source: OverviewFactSource;
  taskName: string;
  to: RouteLocationRaw;
};

export type OverviewOrderFact = {
  at: string;
  id: string;
  market: string;
  price: string;
  quantity: string;
  side: string;
  source: OverviewFactSource;
  status: string;
  taskName: string;
  to: RouteLocationRaw;
};

export async function loadOverviewFacts(backtests: BacktestTask[], tradingTasks: TradingTask[]) {
  const [backtestIntents, backtestOrders, tradingIntents, tradingOrders] = await Promise.all([
    loadBacktestIntentFacts(backtests),
    loadBacktestOrderFacts(backtests),
    loadTradingIntentFacts(tradingTasks),
    loadTradingOrderFacts(tradingTasks),
  ]);

  return {
    orders: [...backtestOrders, ...tradingOrders],
    strategyIntents: [...backtestIntents, ...tradingIntents],
  };
}

export function overviewFactTagType(status: string): TagProps["type"] {
  if (status === "accepted" || status === "filled" || status === "sent" || status === "succeeded") return "success";
  if (status === "cancelled" || status === "failed" || status === "rejected") return "error";
  if (status === "pending" || status === "submitted" || status === "retry_scheduled") return "warning";
  return "default";
}

async function loadBacktestIntentFacts(tasks: BacktestTask[]) {
  const results = await Promise.all(
    recentFactTasks(tasks).map(async (task) =>
      (await backtestsApi.listIntents(task.id)).map((intent) => ({
        intent,
        market: marketLabel(task),
        source: "backtest" as const,
        taskName: task.name,
        to: { name: "backtests-detail", params: { id: task.id } },
      })),
    ),
  );
  return results.flat();
}

async function loadBacktestOrderFacts(tasks: BacktestTask[]) {
  const results = await Promise.all(
    recentFactTasks(tasks).map(async (task) =>
      (await backtestsApi.listOrders(task.id)).map((order) => backtestOrderFact(task, order)),
    ),
  );
  return results.flat();
}

async function loadTradingIntentFacts(tasks: TradingTask[]) {
  const results = await Promise.all(
    recentFactTasks(tasks).map(async (task) =>
      (await tradingApi.listIntents(task.id)).map((intent) => ({
        intent,
        market: marketLabel(task),
        source: "trading" as const,
        taskName: task.name,
        to: { name: "trading-detail", params: { id: task.id } },
      })),
    ),
  );
  return results.flat();
}

async function loadTradingOrderFacts(tasks: TradingTask[]) {
  const results = await Promise.all(
    recentFactTasks(tasks).map(async (task) =>
      (await tradingApi.listOrders(task.id)).map((order) => tradingOrderFact(task, order)),
    ),
  );
  return results.flat();
}

function recentFactTasks<T extends { createdAt?: string; updatedAt?: string }>(tasks: T[]) {
  return [...tasks].sort((left, right) => timestamp(right.updatedAt ?? right.createdAt) - timestamp(left.updatedAt ?? left.createdAt)).slice(0, overviewFactTaskLimit);
}

function backtestOrderFact(task: BacktestTask, order: BacktestOrder): OverviewOrderFact {
  return {
    at: order.occurredAt,
    id: order.id,
    market: marketLabel(task),
    price: order.price,
    quantity: order.quantity,
    side: order.side,
    source: "backtest",
    status: order.status,
    taskName: task.name,
    to: { name: "backtests-detail", params: { id: task.id } },
  };
}

function tradingOrderFact(task: TradingTask, order: Order): OverviewOrderFact {
  return {
    at: order.createdAt,
    id: order.id,
    market: marketLabel(task),
    price: order.price,
    quantity: order.quantity,
    side: order.side,
    source: "trading",
    status: order.status,
    taskName: task.name,
    to: { name: "trading-detail", params: { id: task.id } },
  };
}

function marketLabel(item: { exchange: string; symbol: string; interval: string }) {
  return `${item.exchange} / ${item.symbol} / ${item.interval}`;
}

function timestamp(value?: string) {
  const parsed = Date.parse(value ?? "");
  return Number.isFinite(parsed) ? parsed : 0;
}
