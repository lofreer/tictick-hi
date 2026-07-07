import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import type { RouteLocationRaw } from "vue-router";
import type { TagProps } from "naive-ui";

import { backtestsApi } from "@/services/api/backtests";
import { dataApi } from "@/services/api/data";
import { systemApi } from "@/services/api/system";
import { tradingApi } from "@/services/api/trading";
import {
  loadOverviewFacts,
  overviewFactTagType,
  type OverviewFactSource,
  type OverviewIntentFact,
  type OverviewOrderFact,
} from "@/composables/overviewFacts";
import { buildOverviewDepthMetrics } from "@/composables/overviewDepthMetrics";
import type { BacktestTask, DataSyncTask, Notification, ServiceHealth, SystemHealth, TaskStatus, TradingTask } from "@/types/app";

type SummaryCard = {
  key: string;
  label: string;
  value: number | string;
  detail: string;
  to: RouteLocationRaw;
};

type OverviewAlert = {
  key: string;
  label: string;
  title: string;
  detail: string;
  type: TagProps["type"];
  to: RouteLocationRaw;
};

type OverviewActivity = {
  key: string;
  title: string;
  detail: string;
  status: string;
  statusType: TagProps["type"];
  at?: string;
  to: RouteLocationRaw;
};

type RecentActivityWindow = "24h" | "7d" | "30d";

const recentFactsLimit = 8;
const recentActivityWindowMs: Record<RecentActivityWindow, number> = {
  "24h": 24 * 60 * 60 * 1000,
  "7d": 7 * 24 * 60 * 60 * 1000,
  "30d": 30 * 24 * 60 * 60 * 1000,
};
const recentActivityWindowValues = Object.keys(recentActivityWindowMs) as RecentActivityWindow[];

export function useOverviewWorkspace() {
  const { t } = useI18n();
  const health = ref<SystemHealth | null>(null);
  const dataSyncTasks = ref<DataSyncTask[]>([]);
  const backtests = ref<BacktestTask[]>([]);
  const tradingTasks = ref<TradingTask[]>([]);
  const strategyIntents = ref<OverviewIntentFact[]>([]);
  const orders = ref<OverviewOrderFact[]>([]);
  const notifications = ref<Notification[]>([]);
  const loading = ref(false);
  const hasLoaded = ref(false);
  const error = ref("");
  const factsError = ref("");
  const recentActivityWindow = ref<RecentActivityWindow>("24h");
  const recentActivitySince = ref(recentActivityWindowSince(recentActivityWindow.value));

  onMounted(() => {
    void loadOverview();
  });

  const services = computed(() => health.value?.services ?? []);
  const healthTagType = computed<TagProps["type"]>(() => (health.value?.status === "ok" ? "success" : "warning"));
  const recentActivityWindowOptions = computed(() => recentActivityWindowValues.map((value) => ({ label: t(`overview.recentWindow.${value}`), value })));
  const depthMetrics = computed(() =>
    buildOverviewDepthMetrics(
      {
        backtests: backtests.value,
        dataSyncTasks: dataSyncTasks.value,
        notifications: notifications.value,
        services: services.value,
        tradingTasks: tradingTasks.value,
      },
      t,
    ),
  );

  const summaryCards = computed<SummaryCard[]>(() => [
    {
      key: "sync",
      label: t("overview.dataSync"),
      value: dataSyncTasks.value.length,
      detail: t("overview.dataSyncDetail", {
        running: countStatus(dataSyncTasks.value, "running"),
        failed: countStatus(dataSyncTasks.value, "failed"),
        invalid: countDataHealth(dataSyncTasks.value, "invalid"),
        realtime: dataSyncTasks.value.filter((task) => task.realtimeEnabled).length,
      }),
      to: { name: "research" },
    },
    {
      key: "backtests",
      label: t("overview.backtests"),
      value: backtests.value.length,
      detail: t("overview.backtestsDetail", {
        running: countStatus(backtests.value, "running"),
        failed: countStatus(backtests.value, "failed"),
        succeeded: countStatus(backtests.value, "succeeded"),
      }),
      to: { name: "backtests" },
    },
    {
      key: "trading",
      label: t("overview.tradingTasks"),
      value: tradingTasks.value.length,
      detail: t("overview.tradingDetail", {
        running: countStatus(tradingTasks.value, "running"),
        paper: tradingTasks.value.filter((task) => task.type === "paper").length,
        live: tradingTasks.value.filter((task) => task.type === "live").length,
      }),
      to: { name: "trading" },
    },
    {
      key: "notifications",
      label: t("overview.notifications"),
      value: notifications.value.length,
      detail: t("overview.notificationsDetail", {
        failed: notifications.value.filter((item) => item.status === "failed").length,
        pending: notifications.value.filter((item) => item.status === "pending" || item.status === "retry_scheduled").length,
      }),
      to: { name: "system-notifications", query: { status: notifications.value.some((item) => item.status === "failed") ? "failed" : notifications.value.some((item) => item.status === "pending" || item.status === "retry_scheduled") ? "pending" : "all" } },
    },
    {
      key: "workers",
      label: t("overview.workers"),
      value: services.value.length,
      detail: t("overview.workersDetail", {
        stale: services.value.reduce((total, service) => total + (service.staleLeaseCount ?? 0), 0),
        locked: services.value.reduce((total, service) => total + (service.lockedCount ?? 0), 0),
      }),
      to: { name: "system-health" },
    },
  ]);

  const alerts = computed<OverviewAlert[]>(() => {
    const items: OverviewAlert[] = [];
    if (health.value && health.value.status !== "ok") {
      items.push(alert("health", t("overview.systemHealth"), health.value.status, t("overview.healthAlert"), "warning", { name: "system-health" }));
    }
    if (factsError.value) {
      items.push(alert("recent-facts-degraded", t("overview.recentActivity"), t("overview.degraded"), factsError.value, "warning", { name: "overview" }));
    }
    addCountAlert(items, "sync-failed", dataSyncTasks.value, "failed", t("overview.dataSync"), { name: "research" });
    addDataHealthAlert(items, "sync-gap", dataSyncTasks.value, "gap", t("overview.dataSync"), { name: "research" });
    addDataHealthAlert(items, "sync-invalid", dataSyncTasks.value, "invalid", t("overview.dataSync"), { name: "research" });
    addCountAlert(items, "backtests-failed", backtests.value, "failed", t("overview.backtests"), { name: "backtests" });
    addCountAlert(items, "trading-failed", tradingTasks.value, "failed", t("overview.tradingTasks"), { name: "trading" });

    const failedNotifications = notifications.value.filter((item) => item.status === "failed").length;
    if (failedNotifications > 0) {
      items.push(
        alert(
          "notifications-failed",
          t("overview.notifications"),
          t("overview.failedCount", { count: failedNotifications }),
          t("overview.notificationAlert"),
          "error",
          { name: "system-notifications", query: { status: "failed" } },
        ),
      );
    }
    return items;
  });

  const recentActivities = computed<OverviewActivity[]>(() =>
    [
      ...dataSyncTasks.value.map((task) => activityFromTask(task.id, t("overview.dataSync"), marketLabel(task), task.status, task.updatedAt ?? task.createdAt, { name: "research" })),
      ...backtests.value.map((task) =>
        activityFromTask(task.id, t("overview.backtests"), `${task.name} / ${marketLabel(task)}`, task.status, task.updatedAt ?? task.createdAt, {
          name: "backtests-detail",
          params: { id: task.id },
        }),
      ),
      ...tradingTasks.value.map((task) =>
        activityFromTask(task.id, t("overview.tradingTasks"), `${task.name} / ${marketLabel(task)}`, task.status, task.updatedAt ?? task.createdAt, {
          name: "trading-detail",
          params: { id: task.id },
        }),
      ),
      ...strategyIntents.value.map((item) => ({
        key: `intent-${item.id}`,
        title: t("overview.strategyIntents"),
        detail: `${sourceLabel(item.source)} / ${item.taskName} / ${item.market} / ${item.intentType} / ${item.policy}`,
        status: item.status,
        statusType: overviewFactTagType(item.status),
        at: item.createdAt,
        to: item.to,
      })),
      ...orders.value.map((item) => ({
        key: `order-${item.id}`,
        title: t("overview.orders"),
        detail: `${sourceLabel(item.source)} / ${item.taskName} / ${item.market} / ${item.side} ${item.quantity} @ ${item.price}`,
        status: item.status,
        statusType: overviewFactTagType(item.status),
        at: item.at,
        to: item.to,
      })),
      ...notifications.value.map((item) => ({
        key: `notification-${item.id}`,
        title: t("overview.notifications"),
        detail: `${item.channel} / ${item.title}`,
        status: item.status,
        statusType: notificationTagType(item.status),
        at: item.sentAt ?? item.lastAttemptAt ?? item.createdAt,
        to: { name: "system-notifications", query: { status: item.status === "failed" ? "failed" : item.status === "pending" || item.status === "retry_scheduled" ? "pending" : item.status === "sent" || item.status === "delivered" ? "sent" : "all" } },
      })),
    ]
      .filter((item) => item.at)
      .filter((item) => Date.parse(item.at ?? "") >= Date.parse(recentActivitySince.value))
      .sort((left, right) => Date.parse(right.at ?? "") - Date.parse(left.at ?? ""))
      .slice(0, 8),
  );

  async function loadOverview() {
    loading.value = true;
    error.value = "";
    factsError.value = "";
    try {
      const [nextHealth, nextSyncTasks, nextBacktests, nextTradingTasks, nextNotifications] = await Promise.all([
        systemApi.health(),
        dataApi.listTasks(),
        backtestsApi.listBacktests(),
        tradingApi.listTasks(),
        systemApi.listNotifications(),
      ]);
      health.value = nextHealth;
      dataSyncTasks.value = nextSyncTasks;
      backtests.value = nextBacktests;
      tradingTasks.value = nextTradingTasks;
      notifications.value = nextNotifications;
      await loadRecentFactsSafely();
      hasLoaded.value = true;
    } catch (loadError) {
      error.value = errorMessage(loadError, t("overview.loadFailed"));
    } finally {
      loading.value = false;
    }
  }

  async function loadRecentFactsSafely() {
    recentActivitySince.value = recentActivityWindowSince(recentActivityWindow.value);
    try {
      const nextFacts = await loadOverviewFacts({ limit: recentFactsLimit, since: recentActivitySince.value });
      strategyIntents.value = nextFacts.strategyIntents;
      orders.value = nextFacts.orders;
    } catch (loadError) {
      strategyIntents.value = [];
      orders.value = [];
      factsError.value = errorMessage(loadError, t("overview.recentFactsLoadFailed"));
    }
  }

  async function setRecentActivityWindow(value: string) {
    if (!isRecentActivityWindow(value) || value === recentActivityWindow.value) return;
    recentActivityWindow.value = value;
    factsError.value = "";
    await loadRecentFactsSafely();
  }

  function addCountAlert(
    items: OverviewAlert[],
    key: string,
    tasks: { status: TaskStatus }[],
    status: TaskStatus,
    title: string,
    to: RouteLocationRaw,
  ) {
    const count = countStatus(tasks, status);
    if (count === 0) return;
    const type = status === "failed" ? "error" : "warning";
    items.push(alert(key, title, t("overview.failedCount", { count }), t(`status.${status}`), type, to));
  }

  function addDataHealthAlert(
    items: OverviewAlert[],
    key: string,
    tasks: DataSyncTask[],
    health: DataSyncTask["dataHealth"],
    title: string,
    to: RouteLocationRaw,
  ) {
    const count = countDataHealth(tasks, health);
    if (count === 0) return;
    const type = health === "failed" || health === "invalid" ? "error" : "warning";
    items.push(alert(key, title, t("overview.failedCount", { count }), t(`research.dataHealth.${health}`), type, to));
  }

  function activityFromTask(
    id: string,
    title: string,
    detail: string,
    status: TaskStatus,
    at: string | undefined,
    to: RouteLocationRaw,
  ): OverviewActivity {
    return {
      key: `${title}-${id}`,
      title,
      detail,
      status: t(`status.${status}`),
      statusType: taskTagType(status),
      at,
      to,
    };
  }

  function serviceSummary(service: ServiceHealth) {
    return t("overview.serviceSummary", {
      pending: service.pendingCount ?? 0,
      running: service.runningCount ?? 0,
      locked: service.lockedCount ?? 0,
    });
  }

  function sourceLabel(source: OverviewFactSource) {
    return source === "backtest" ? t("overview.backtests") : t("overview.tradingTasks");
  }

  return {
    alerts,
    depthMetrics,
    error,
    factsError,
    formatDate,
    hasLoaded,
    health,
    healthTagType,
    loadOverview,
    loading,
    recentActivities,
    recentActivityWindow,
    recentActivityWindowOptions,
    serviceSummary,
    setRecentActivityWindow,
    services,
    summaryCards,
    t,
  };
}

function countStatus(tasks: { status: TaskStatus }[], status: TaskStatus) {
  return tasks.filter((task) => task.status === status).length;
}

function countDataHealth(tasks: DataSyncTask[], health: DataSyncTask["dataHealth"]) {
  return tasks.filter((task) => task.dataHealth === health).length;
}

function alert(key: string, title: string, label: string, detail: string, type: TagProps["type"], to: RouteLocationRaw): OverviewAlert {
  return { key, title, label, detail, type, to };
}

function taskTagType(status: TaskStatus): TagProps["type"] {
  if (status === "running" || status === "succeeded") return "success";
  if (status === "failed" || status === "cancelled") return "error";
  if (status === "gap" || status === "stopping") return "warning";
  return "default";
}

function notificationTagType(status: string): TagProps["type"] {
  if (status === "sent" || status === "delivered") return "success";
  if (status === "failed") return "error";
  if (status === "retry_scheduled") return "warning";
  return "default";
}

function marketLabel(item: { exchange: string; symbol: string; interval: string }) {
  return `${item.exchange} / ${item.symbol} / ${item.interval}`;
}

function formatDate(value?: string) {
  return value ? new Date(value).toLocaleString() : "-";
}

function isRecentActivityWindow(value: string): value is RecentActivityWindow {
  return Object.prototype.hasOwnProperty.call(recentActivityWindowMs, value);
}

function recentActivityWindowSince(value: RecentActivityWindow) {
  return new Date(Date.now() - recentActivityWindowMs[value]).toISOString();
}

function errorMessage(loadError: unknown, fallback: string) {
  return loadError instanceof Error && loadError.message ? loadError.message : fallback;
}
