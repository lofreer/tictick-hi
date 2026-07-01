import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";

import StrategyMarketCatalogStatus from "@/components/strategy/StrategyMarketCatalogStatus.vue";
import { i18n } from "@/i18n";

describe("StrategyMarketCatalogStatus", () => {
  it("shows active catalog status with localized exchange detail", () => {
    const wrapper = mountStatus({ status: "active", detail: "TRADING" });

    expect(wrapper.text()).toContain("可用");
    expect(wrapper.text()).toContain("交易所状态：交易中");
    expect(wrapper.text()).not.toContain("TRADING");
  });

  it("shows inactive catalog status without leaking raw exchange detail", () => {
    const wrapper = mountStatus({ status: "inactive", detail: "BREAK" });

    expect(wrapper.text()).toContain("不可用");
    expect(wrapper.text()).toContain("暂停交易");
    expect(wrapper.text()).not.toContain("BREAK");
  });
});

function mountStatus(props: {
  detail: string;
  status: "active" | "inactive" | "missing" | "unknown";
}) {
  return mount(StrategyMarketCatalogStatus, {
    global: { plugins: [i18n] },
    props: {
      error: "",
      loading: false,
      ...props,
    },
  });
}
