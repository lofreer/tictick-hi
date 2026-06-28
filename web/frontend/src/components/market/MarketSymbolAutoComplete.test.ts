import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";

import MarketSymbolAutoComplete from "@/components/market/MarketSymbolAutoComplete.vue";
import { marketApi } from "@/services/api/market";

const marketApiMocks = vi.hoisted(() => ({
  listInstruments: vi.fn(),
}));

vi.mock("@/services/api/market", () => ({
  marketApi: marketApiMocks,
}));

vi.mock("naive-ui", () => ({
  NAutoComplete: {
    name: "NAutoComplete",
    props: ["loading", "options", "value"],
    emits: ["update:value"],
    template: "<input :value='value' />",
  },
}));

describe("MarketSymbolAutoComplete", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    marketApiMocks.listInstruments.mockResolvedValue([
      {
        exchange: "binance",
        symbol: "SOLUSDT",
        baseAsset: "SOL",
        quoteAsset: "USDT",
        instrumentType: "spot",
        status: "active",
        searchPriority: 3,
        createdAt: "2026-06-28T00:00:00Z",
        updatedAt: "2026-06-28T00:00:00Z",
      },
    ]);
  });

  it("loads symbol options from the market instrument API", async () => {
    const wrapper = mountAutoComplete();
    await flushPromises();

    expect(marketApi.listInstruments).toHaveBeenCalledWith({ exchange: "binance", limit: 20, q: "" });
    expect(wrapper.getComponent({ name: "NAutoComplete" }).props("options")).toEqual([
      { label: "SOLUSDT", value: "SOLUSDT" },
    ]);
  });

  it("falls back to local suggestions when the API fails", async () => {
    marketApiMocks.listInstruments.mockRejectedValueOnce(new Error("network"));
    const wrapper = mountAutoComplete();
    await flushPromises();

    expect(wrapper.getComponent({ name: "NAutoComplete" }).props("options")).toEqual([
      { label: "BTCUSDT", value: "BTCUSDT" },
      { label: "ETHUSDT", value: "ETHUSDT" },
    ]);
  });

  it("emits value updates from the underlying autocomplete", async () => {
    const wrapper = mountAutoComplete();
    await flushPromises();

    wrapper.getComponent({ name: "NAutoComplete" }).vm.$emit("update:value", "SOLUSDT");

    expect(wrapper.emitted("update:value")).toEqual([["SOLUSDT"]]);
  });
});

function mountAutoComplete() {
  return mount(MarketSymbolAutoComplete, {
    props: {
      exchange: "binance",
      value: "",
    },
  });
}
