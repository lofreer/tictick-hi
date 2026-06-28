import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";

import MarketCandleGapTag from "@/components/research/MarketCandleGapTag.vue";
import { i18n } from "@/i18n";
import { dataApi } from "@/services/api/data";

const dataApiMocks = vi.hoisted(() => ({
  scanMarketCandleGaps: vi.fn(),
}));

vi.mock("@/services/api/data", () => ({
  dataApi: dataApiMocks,
}));

describe("MarketCandleGapTag", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    dataApiMocks.scanMarketCandleGaps.mockResolvedValue({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      window: { count: 4 },
      gaps: [{ from: "2026-06-27T03:02:00Z", to: "2026-06-27T03:03:00Z", missingCandles: 1 }],
      limited: false,
      totalCount: 1,
      returnedCount: 1,
    });
  });

  it("loads and displays full-history gap metadata for the selected market", async () => {
    const wrapper = mountTag();
    await flushPromises();

    expect(dataApi.scanMarketCandleGaps).toHaveBeenCalledWith({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      limit: 20,
    });
    expect(wrapper.text()).toContain("全历史缺口 1 处");
    expect(wrapper.attributes("title")).toContain("2026-06-27 03:02:00 UTC");
  });

  it("shows a failed scan state without throwing", async () => {
    dataApiMocks.scanMarketCandleGaps.mockRejectedValueOnce(new Error("network"));
    const wrapper = mountTag();
    await flushPromises();

    expect(wrapper.text()).toContain("全历史缺口扫描失败");
  });
});

function mountTag() {
  return mount(MarketCandleGapTag, {
    global: { plugins: [i18n] },
    props: {
      exchange: "binance",
      interval: "1m",
      symbol: "BTCUSDT",
    },
  });
}
