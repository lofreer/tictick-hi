import type { RouteLocationRaw } from "vue-router";
import type { TagProps } from "naive-ui";

import type { BacktestTask, DataSyncTask, Notification, ServiceHealth, TaskStatus, TradingTask } from "@/types/app";

type Translate = (key: string, named?: Record<string, string | number>) => string;

export type OverviewDepthMetric = {
  key: string;
  label: string;
  value: string;
  detail: string;
  statusLabel: string;
  statusType: TagProps["type"];
  to: RouteLocationRaw;
};

export type OverviewDepthMetricSources = {
  backtests: BacktestTask[];
  dataSyncTasks: DataSyncTask[];
  notifications: Notification[];
  services: ServiceHealth[];
  tradingTasks: TradingTask[];
};

export function buildOverviewDepthMetrics(sources: OverviewDepthMetricSources, t: Translate): OverviewDepthMetric[] {
  return [
    dataQualityMetric(sources.dataSyncTasks, t),
    automationMetric(sources.dataSyncTasks, sources.services, t),
    executionMetric(sources.backtests, sources.tradingTasks, t),
    deliveryMetric(sources.notifications, t),
  ];
}

function dataQualityMetric(tasks: DataSyncTask[], t: Translate): OverviewDepthMetric {
  const failed = countStatus(tasks, "failed");
  const gap = countDataHealth(tasks, "gap");
  const invalid = countDataHealth(tasks, "invalid");
  const healthy = tasks.filter((task) => task.status !== "failed" && task.status !== "cancelled" && (task.dataHealth === "ok" || task.dataHealth === "syncing")).length;
  const statusType = failed + invalid > 0 ? "error" : gap > 0 ? "warning" : statusTypeForActivity(tasks.length);
  return {
    key: "data-quality",
    label: t("overview.depth.dataQuality"),
    value: ratio(healthy, tasks.length),
    detail: t("overview.depth.dataQualityDetail", { failed, gap, healthy, invalid }),
    statusLabel: statusLabel(statusType, t),
    statusType,
    to: { name: "research", query: { dataHealth: failed > 0 ? "failed" : invalid > 0 ? "invalid" : gap > 0 ? "gap" : "all" } },
  };
}

function automationMetric(tasks: DataSyncTask[], services: ServiceHealth[], t: Translate): OverviewDepthMetric {
  const realtime = tasks.filter((task) => task.realtimeEnabled).length;
  const running = countStatus(tasks, "running");
  const stale = services.reduce((total, service) => total + (service.staleLeaseCount ?? 0), 0);
  const backoff = services.reduce((total, service) => total + (service.exchangeBackoffCount ?? 0), 0);
  const unhealthyServices = services.filter((service) => service.status !== "ok").length;
  const statusType = stale > 0 ? "error" : backoff + unhealthyServices > 0 ? "warning" : statusTypeForActivity(realtime + running + services.length);
  return {
    key: "automation",
    label: t("overview.depth.automation"),
    value: ratio(running, Math.max(realtime, tasks.length)),
    detail: t("overview.depth.automationDetail", { backoff, realtime, running, stale }),
    statusLabel: statusLabel(statusType, t),
    statusType,
    to: { name: "system-health" },
  };
}

function executionMetric(backtests: BacktestTask[], tradingTasks: TradingTask[], t: Translate): OverviewDepthMetric {
  const running = countStatus(backtests, "running") + countStatus(tradingTasks, "running");
  const backtestFailed = countStatus(backtests, "failed");
  const tradingFailed = countStatus(tradingTasks, "failed");
  const live = tradingTasks.filter((task) => task.type === "live").length;
  const total = backtests.length + tradingTasks.length;
  const statusType = backtestFailed + tradingFailed > 0 ? "error" : running === 0 && total > 0 ? "warning" : statusTypeForActivity(total);
  return {
    key: "execution",
    label: t("overview.depth.execution"),
    value: ratio(running, total),
    detail: t("overview.depth.executionDetail", { backtestFailed, live, running, tradingFailed }),
    statusLabel: statusLabel(statusType, t),
    statusType,
    to: tradingFailed > 0
      ? { name: "trading", query: { status: "failed" } }
      : backtestFailed > 0
        ? { name: "backtests", query: { status: "failed" } }
        : { name: "trading", query: { status: running > 0 ? "running" : "all" } },
  };
}

function deliveryMetric(notifications: Notification[], t: Translate): OverviewDepthMetric {
  const sent = notifications.filter((item) => item.status === "sent" || item.status === "delivered").length;
  const failed = notifications.filter((item) => item.status === "failed").length;
  const pending = notifications.filter((item) => item.status === "pending" || item.status === "retry_scheduled").length;
  const statusType = failed > 0 ? "error" : pending > 0 ? "warning" : statusTypeForActivity(notifications.length);
  return {
    key: "delivery",
    label: t("overview.depth.delivery"),
    value: ratio(sent, notifications.length),
    detail: t("overview.depth.deliveryDetail", { failed, pending, sent }),
    statusLabel: statusLabel(statusType, t),
    statusType,
    to: { name: "system-notifications", query: { status: failed > 0 ? "failed" : pending > 0 ? "pending" : "all" } },
  };
}

function countStatus(tasks: { status: TaskStatus }[], status: TaskStatus) {
  return tasks.filter((task) => task.status === status).length;
}

function countDataHealth(tasks: DataSyncTask[], health: DataSyncTask["dataHealth"]) {
  return tasks.filter((task) => task.dataHealth === health).length;
}

function ratio(value: number, total: number) {
  return `${value}/${total}`;
}

function statusTypeForActivity(count: number): TagProps["type"] {
  return count > 0 ? "success" : "default";
}

function statusLabel(type: TagProps["type"], t: Translate) {
  if (type === "error") return t("overview.depth.status.error");
  if (type === "warning") return t("overview.depth.status.warning");
  if (type === "success") return t("overview.depth.status.ok");
  return t("overview.depth.status.idle");
}
