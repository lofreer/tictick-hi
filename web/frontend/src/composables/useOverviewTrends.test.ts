import { mount, flushPromises } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { i18n } from "@/i18n";
import { overviewApi } from "@/services/api/overview";
import { useOverviewTrends } from "@/composables/useOverviewTrends";

const apiMocks = vi.hoisted(() => ({
  overviewTrends: vi.fn(),
}));

vi.mock("@/services/api/overview", () => ({
  overviewApi: {
    trends: apiMocks.overviewTrends,
  },
}));

describe("useOverviewTrends", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    apiMocks.overviewTrends.mockResolvedValue({ days: 7, from: "2026-07-01T00:00:00Z", to: "2026-07-08T00:00:00Z", buckets: [] });
  });

  it("loads overview trend buckets and derives totals", async () => {
    apiMocks.overviewTrends.mockResolvedValue({
      days: 7,
      from: "2026-07-01T00:00:00Z",
      to: "2026-07-08T00:00:00Z",
      buckets: [
        { bucketStart: "2026-07-01T00:00:00Z", strategyIntents: 2, orders: 1, notifications: 0, failures: 0 },
        { bucketStart: "2026-07-02T00:00:00Z", strategyIntents: 0, orders: 1, notifications: 2, failures: 1 },
      ],
    });

    const workspace = mountTrends();
    await flushPromises();

    expect(overviewApi.trends).toHaveBeenCalledWith({ days: 7 });
    expect(workspace.hasTrendData.value).toBe(true);
    expect(workspace.trendTotals.value).toEqual({ strategyIntents: 2, orders: 2, notifications: 2, failures: 1 });
    expect(workspace.trendPoints.value.map((point) => ({ failures: point.failures, total: point.total, totalPct: point.totalPct }))).toEqual([
      { failures: 0, total: 3, totalPct: 100 },
      { failures: 1, total: 3, totalPct: 100 },
    ]);
  });

  it("surfaces trend loading failures as local degradation", async () => {
    apiMocks.overviewTrends.mockRejectedValue(new Error("trend unavailable"));

    const workspace = mountTrends();
    await flushPromises();

    expect(workspace.error.value).toBe("trend unavailable");
    expect(workspace.trendPoints.value).toEqual([]);
    expect(workspace.hasTrendData.value).toBe(false);
  });
});

function mountTrends() {
  const holder: { workspace?: ReturnType<typeof useOverviewTrends> } = {};
  mount(
    {
      template: "<div />",
      setup() {
        holder.workspace = useOverviewTrends();
        return {};
      },
    },
    {
      global: {
        plugins: [i18n],
      },
    },
  );
  if (!holder.workspace) {
    throw new Error("overview trends workspace was not mounted");
  }
  return holder.workspace;
}
