import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";

import { overviewApi } from "@/services/api/overview";
import type { OverviewTrendBucket } from "@/types/app";

const overviewTrendDays = 7;

export type OverviewTrendPoint = OverviewTrendBucket & {
  failurePct: number;
  label: string;
  total: number;
  totalPct: number;
};

export type OverviewTrendTotals = {
  strategyIntents: number;
  orders: number;
  notifications: number;
  failures: number;
};

export function useOverviewTrends() {
  const { t } = useI18n();
  const trends = ref<OverviewTrendBucket[]>([]);
  const loading = ref(false);
  const error = ref("");

  onMounted(() => {
    void loadOverviewTrends();
  });

  const trendTotals = computed(() =>
    trends.value.reduce(
      (totals, bucket) => ({
        strategyIntents: totals.strategyIntents + bucket.strategyIntents,
        orders: totals.orders + bucket.orders,
        notifications: totals.notifications + bucket.notifications,
        failures: totals.failures + bucket.failures,
      }),
      { strategyIntents: 0, orders: 0, notifications: 0, failures: 0 } satisfies OverviewTrendTotals,
    ),
  );
  const maxTrendValue = computed(() =>
    Math.max(
      1,
      ...trends.value.map((bucket) => bucket.strategyIntents + bucket.orders + bucket.notifications),
      ...trends.value.map((bucket) => bucket.failures),
    ),
  );
  const trendPoints = computed<OverviewTrendPoint[]>(() =>
    trends.value.map((bucket) => {
      const total = bucket.strategyIntents + bucket.orders + bucket.notifications;
      return {
        ...bucket,
        failurePct: Math.max(4, Math.round((bucket.failures / maxTrendValue.value) * 100)),
        label: formatTrendDate(bucket.bucketStart),
        total,
        totalPct: Math.max(4, Math.round((total / maxTrendValue.value) * 100)),
      };
    }),
  );
  const hasTrendData = computed(() => trendPoints.value.some((point) => point.total > 0 || point.failures > 0));

  async function loadOverviewTrends() {
    loading.value = true;
    error.value = "";
    try {
      const nextTrends = await overviewApi.trends({ days: overviewTrendDays });
      trends.value = nextTrends.buckets;
    } catch (loadError) {
      trends.value = [];
      error.value = loadError instanceof Error && loadError.message ? loadError.message : t("overview.trendsLoadFailed");
    } finally {
      loading.value = false;
    }
  }

  return {
    error,
    hasTrendData,
    loadOverviewTrends,
    loading,
    trendPoints,
    trendTotals,
    t,
  };
}

function formatTrendDate(value: string) {
  return new Date(value).toLocaleDateString(undefined, { day: "2-digit", month: "2-digit" });
}
