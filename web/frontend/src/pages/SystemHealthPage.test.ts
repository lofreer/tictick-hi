import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { i18n } from "@/i18n";
import SystemHealthPage from "@/pages/SystemHealthPage.vue";

const apiMocks = vi.hoisted(() => ({
  health: vi.fn(),
}));

const routerMocks = vi.hoisted(() => ({
  replace: vi.fn(),
  query: {} as Record<string, string>,
}));

vi.mock("@/services/api/system", () => ({
  systemApi: {
    health: apiMocks.health,
  },
}));

vi.mock("vue-router", () => ({
  useRoute: () => ({ query: routerMocks.query }),
  useRouter: () => ({ replace: routerMocks.replace }),
}));

describe("SystemHealthPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routerMocks.query = {};
  });

  it("renders data sync fetch lock skip metrics from system health", async () => {
    apiMocks.health.mockResolvedValue({
      status: "ok",
      database: "ok",
      checkedAt: "2026-06-30T09:00:00Z",
      services: [
        {
          name: "sync-worker",
          status: "ok",
          pendingCount: 0,
          runningCount: 0,
          lockedCount: 0,
          staleLeaseCount: 0,
          fetchLockSkipCount: 3,
          lastFetchLockSkippedAt: "2026-06-30T08:55:00Z",
        },
      ],
    });

    const wrapper = mount(SystemHealthPage, {
      global: {
        plugins: [i18n],
      },
    });
    await flushPromises();

    expect(wrapper.text()).toContain("fetch 锁跳过");
    expect(wrapper.text()).toContain("3");
    expect(wrapper.text()).toContain("最近 fetch 锁跳过");
    expect(wrapper.text()).toContain("全部");
    expect(wrapper.text()).toContain("异常");
    expect(wrapper.text()).toContain("冷却");
  });

  it("filters services from focus query context", async () => {
    routerMocks.query = { focus: "stale" };
    apiMocks.health.mockResolvedValue({
      status: "warning",
      database: "ok",
      checkedAt: "2026-07-07T09:00:00Z",
      services: [
        {
          name: "sync-worker",
          status: "warning",
          staleLeaseCount: 2,
        },
        {
          name: "notify-worker",
          status: "ok",
          exchangeBackoffCount: 1,
        },
      ],
    });

    const wrapper = mount(SystemHealthPage, {
      global: {
        plugins: [i18n],
      },
    });
    await flushPromises();

    expect(wrapper.text()).toContain("sync-worker");
    expect(wrapper.text()).not.toContain("notify-worker");
    expect(wrapper.text()).toContain("过期锁");
  });

  it("shows an empty state when the focus has no matching services", async () => {
    routerMocks.query = { focus: "backoff" };
    apiMocks.health.mockResolvedValue({
      status: "ok",
      database: "ok",
      checkedAt: "2026-07-07T09:00:00Z",
      services: [{ name: "sync-worker", status: "ok" }],
    });

    const wrapper = mount(SystemHealthPage, {
      global: {
        plugins: [i18n],
      },
    });
    await flushPromises();

    expect(wrapper.text()).toContain("当前筛选下暂无服务");
    expect(wrapper.text()).not.toContain("sync-worker");
  });
});
