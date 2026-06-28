import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import type { RouteLocationRaw } from "vue-router";
import type { TagProps } from "naive-ui";

import { backtestsApi } from "@/services/api/backtests";
import { dataApi } from "@/services/api/data";
import { systemApi } from "@/services/api/system";
import { tradingApi } from "@/services/api/trading";
import type { BacktestTask, DataSyncTask, Notification, ServiceHealth, SystemHealth, TaskStatus, TradingTask } from "@/types/app";

type SummaryCard = {
  key: string;
  label: string;
  value: number | string;
  detail: string;
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

export function useOverviewWorkspace() {
  const { t } = useI18n();
  const health = ref<SystemHealth | null>(null);
  const dataSyncTasks = ref<DataSyncTask[]>([]);
  const backtests = ref<BacktestTask[]>([]);
  const tradingTasks = ref<TradingTask[]>([]);
  const notifications = ref<Notification[]>([]);
  const loading = ref(false);
  const hasLoaded = ref(false);
  const error = ref("");

  onMounted(() => {
    void loadOverview();
  });

  const services = computed(() => health.value?.services ?? []);
  const healthTagType = computed<TagProps["type"]>(() => (health.value?.status === "ok" ? "success" : "warning"));

  const summaryCards = computed<SummaryCard[]>(() => [
    {
      key: "sync",
      label: t("overview.dataSync"),
      value: dataSyncTasks.value.length,
      detail: t("overview.dataSyncDetail", {
        running: countStatus(dataSyncTasks.value, "running"),
        failed: countStatus(dataSyncTasks.value, "failed"),
        realtime: dataSyncTasks.value.filter((task) => task.realtimeEnabled).length,
      }),
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
    },
    {
      key: "notifications",
      label: t("overview.notifications"),
      value: notifications.value.length,
      detail: t("overview.notificationsDetail", {
        failed: notifications.value.filter((item) => item.status === "failed").length,
        pending: notifications.value.filter((item) => item.status === "pending" || item.status === "retry_scheduled").length,
      }),
    },
    {
      key: "workers",
      label: t("overview.workers"),
      value: services.value.length,
      detail: t("overview.workersDetail", {
        stale: services.value.reduce((total, service) => total + (service.staleLeaseCount ?? 0), 0),
        locked: services.value.reduce((total, service) => total + (service.lockedCount ?? 0), 0),
      }),
    },
  ]);

  const alerts = computed<OverviewAlert[]>(() => {
    const items: OverviewAlert[] = [];
    if (health.value && health.value.status !== "ok") {
      items.push(alert("health", t("overview.systemHealth"), health.value.status, t("overview.healthAlert"), "warning", { name: "system-health" }));
    }
    addCountAlert(items, "sync-failed", dataSyncTasks.value, "failed", t("overview.dataSync"), { name: "research" });
    addDataHealthAlert(items, "sync-gap", dataSyncTasks.value, "gap", t("overview.dataSync"), { name: "research" });
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
          { name: "system-notifications" },
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
      ...notifications.value.map((item) => ({
        key: `notification-${item.id}`,
        title: t("overview.notifications"),
        detail: `${item.channel} / ${item.title}`,
        status: item.status,
        statusType: notificationTagType(item.status),
        at: item.sentAt ?? item.lastAttemptAt ?? item.createdAt,
        to: { name: "system-notifications" },
      })),
    ]
      .filter((item) => item.at)
      .sort((left, right) => Date.parse(right.at ?? "") - Date.parse(left.at ?? ""))
      .slice(0, 8),
  );

  async function loadOverview() {
    loading.value = true;
    error.value = "";
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
      hasLoaded.value = true;
    } catch (loadError) {
      error.value = errorMessage(loadError, t("overview.loadFailed"));
    } finally {
      loading.value = false;
    }
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
    const count = tasks.filter((task) => task.dataHealth === health).length;
    if (count === 0) return;
    const type = health === "failed" ? "error" : "warning";
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

  return {
    alerts,
    error,
    formatDate,
    hasLoaded,
    health,
    healthTagType,
    loadOverview,
    loading,
    recentActivities,
    serviceSummary,
    services,
    summaryCards,
    t,
  };
}

function countStatus(tasks: { status: TaskStatus }[], status: TaskStatus) {
  return tasks.filter((task) => task.status === status).length;
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

function errorMessage(loadError: unknown, fallback: string) {
  return loadError instanceof Error && loadError.message ? loadError.message : fallback;
}
