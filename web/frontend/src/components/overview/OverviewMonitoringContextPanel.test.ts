import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";

import OverviewMonitoringContextPanel from "@/components/overview/OverviewMonitoringContextPanel.vue";
import { i18n } from "@/i18n";
import type { SystemHealth } from "@/types/app";

describe("OverviewMonitoringContextPanel", () => {
  it("summarizes snapshot, source degradation, trend coverage, and alert load", () => {
    const wrapper = mount(OverviewMonitoringContextPanel, {
      global: { plugins: [i18n] },
      props: {
        alertCount: 2,
        factsError: "recent facts unavailable",
        formatDate: (value?: string) => `formatted:${value ?? "-"}`,
        hasTrendData: true,
        health: systemHealth({ checkedAt: "2026-07-07T08:00:00Z", services: [], status: "warning" }),
        serviceCount: 3,
        trendPointCount: 7,
        trendsError: "",
      },
    });

    expect(wrapper.text()).toContain("监控上下文");
    expect(wrapper.text()).toContain("formatted:2026-07-07T08:00:00Z");
    expect(wrapper.text()).toContain("健康 warning / 服务 3");
    expect(wrapper.text()).toContain("数据源降级");
    expect(wrapper.text()).toContain("1/2");
    expect(wrapper.text()).toContain("降级 1 / 来源 2");
    expect(wrapper.text()).toContain("趋势覆盖");
    expect(wrapper.text()).toContain("7D bucket 7 个");
    expect(wrapper.text()).toContain("告警负载");
    expect(wrapper.text()).toContain("当前异常提醒 2 项");
  });

  it("marks empty trend coverage as a watch state without failing the snapshot", () => {
    const wrapper = mount(OverviewMonitoringContextPanel, {
      global: { plugins: [i18n] },
      props: {
        alertCount: 0,
        factsError: "",
        formatDate: (value?: string) => value ?? "-",
        hasTrendData: false,
        health: systemHealth({ checkedAt: "2026-07-07T08:00:00Z", services: [], status: "ok" }),
        serviceCount: 0,
        trendPointCount: 0,
        trendsError: "",
      },
    });

    expect(wrapper.text()).toContain("趋势覆盖");
    expect(wrapper.text()).toContain("7D bucket 0 个");
    expect(wrapper.text()).toContain("关注");
    expect(wrapper.text()).toContain("当前异常提醒 0 项");
  });
});

function systemHealth(overrides: Partial<SystemHealth>): SystemHealth {
  return {
    checkedAt: "2026-07-07T08:00:00Z",
    database: "ok",
    services: [],
    status: "ok",
    ...overrides,
  };
}
