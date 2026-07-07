import type { RouteLocationRaw } from "vue-router";
import type { TagProps } from "naive-ui";

import { overviewApi } from "@/services/api/overview";
import type { OverviewOrderFact as APIOverviewOrderFact, OverviewStrategyIntentFact as APIOverviewStrategyIntentFact } from "@/types/app";

export type OverviewFactSource = "backtest" | "trading";

export type OverviewIntentFact = APIOverviewStrategyIntentFact & {
  market: string;
  source: OverviewFactSource;
  to: RouteLocationRaw;
};

export type OverviewOrderFact = APIOverviewOrderFact & {
  at: string;
  market: string;
  source: OverviewFactSource;
  to: RouteLocationRaw;
};

export async function loadOverviewFacts() {
  const facts = await overviewApi.recentFacts();
  return {
    orders: facts.orders.map(orderFact),
    strategyIntents: facts.strategyIntents.map(intentFact),
  };
}

export function overviewFactTagType(status: string): TagProps["type"] {
  if (status === "accepted" || status === "filled" || status === "sent" || status === "succeeded") return "success";
  if (status === "cancelled" || status === "failed" || status === "rejected") return "error";
  if (status === "pending" || status === "submitted" || status === "retry_scheduled") return "warning";
  return "default";
}

function intentFact(intent: APIOverviewStrategyIntentFact): OverviewIntentFact {
  return {
    ...intent,
    market: marketLabel(intent),
    source: sourceFromTaskType(intent.taskType),
    to: detailRoute(intent.taskType, intent.taskId),
  };
}

function orderFact(order: APIOverviewOrderFact): OverviewOrderFact {
  return {
    ...order,
    at: order.occurredAt,
    market: marketLabel(order),
    source: sourceFromTaskType(order.taskType),
    to: detailRoute(order.taskType, order.taskId),
  };
}

function sourceFromTaskType(taskType: string): OverviewFactSource {
  return taskType === "backtest" ? "backtest" : "trading";
}

function detailRoute(taskType: string, taskID: string): RouteLocationRaw {
  if (taskType === "backtest") {
    return { name: "backtests-detail", params: { id: taskID } };
  }
  return { name: "trading-detail", params: { id: taskID } };
}

function marketLabel(item: { exchange: string; symbol: string; interval: string }) {
  return `${item.exchange} / ${item.symbol} / ${item.interval}`;
}
