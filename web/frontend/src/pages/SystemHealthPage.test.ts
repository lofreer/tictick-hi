import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { i18n } from "@/i18n";
import SystemHealthPage from "@/pages/SystemHealthPage.vue";

const apiMocks = vi.hoisted(() => ({
  health: vi.fn(),
}));

vi.mock("@/services/api/system", () => ({
  systemApi: {
    health: apiMocks.health,
  },
}));

describe("SystemHealthPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
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
  });
});
