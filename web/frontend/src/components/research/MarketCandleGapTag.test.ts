import { flushPromises, mount } from "@vue/test-utils";
import { NConfigProvider, NMessageProvider } from "naive-ui";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { defineComponent } from "vue";

import MarketCandleGapTag from "@/components/research/MarketCandleGapTag.vue";
import { i18n } from "@/i18n";
import { dataApi } from "@/services/api/data";

const dataApiMocks = vi.hoisted(() => ({
  repairMarketCandleGap: vi.fn(),
  scanMarketCandleGaps: vi.fn(),
}));

vi.mock("@/services/api/data", () => ({
  dataApi: dataApiMocks,
}));

describe("MarketCandleGapTag", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
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
    dataApiMocks.repairMarketCandleGap.mockResolvedValue({
      sourceTaskId: "",
      createdTasks: [
        {
          id: "dst_market_repair_1",
          exchange: "binance",
          symbol: "BTCUSDT",
          interval: "1m",
          startTime: "2026-06-27T03:02:00Z",
          endTime: "2026-06-27T03:03:00Z",
          syncEnabled: true,
          realtimeEnabled: false,
          status: "pending",
          dataHealth: "syncing",
        },
      ],
      skippedExisting: 0,
      limited: false,
      totalCount: 1,
      repairLimit: 1,
    });
  });

  afterEach(() => {
    document.body.innerHTML = "";
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
    expect(wrapper.find('[role="button"]').attributes("title")).toContain("2026-06-27 03:02:00 UTC");
  });

  it("opens full-history gap details and repairs the first gap", async () => {
    const wrapper = mountTag();
    await flushPromises();

    await wrapper.find('[role="button"]').trigger("click");
    await flushPromises();

    expect(document.body.textContent).toContain("全历史缺口详情");
    expect(document.body.textContent).toContain("2026-06-27 03:02:00 UTC");

    const repairButton = Array.from(document.body.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("修复首个缺口"),
    );
    expect(repairButton).toBeDefined();
    repairButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    await flushPromises();

    expect(dataApi.repairMarketCandleGap).toHaveBeenCalledWith({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      from: "2026-06-27T03:02:00Z",
      to: "2026-06-27T03:03:00Z",
    });
    expect(wrapper.findComponent(MarketCandleGapTag).emitted("repaired")).toHaveLength(1);
    expect(dataApi.scanMarketCandleGaps).toHaveBeenCalledTimes(2);
  });

  it("shows a failed scan state without throwing", async () => {
    dataApiMocks.scanMarketCandleGaps.mockRejectedValueOnce(new Error("network"));
    const wrapper = mountTag();
    await flushPromises();

    expect(wrapper.text()).toContain("全历史缺口扫描失败");
  });
});

function mountTag() {
  const wrapper = defineComponent({
    components: { MarketCandleGapTag, NConfigProvider, NMessageProvider },
    template: `
      <NConfigProvider>
        <NMessageProvider>
          <MarketCandleGapTag exchange="binance" interval="1m" symbol="BTCUSDT" />
        </NMessageProvider>
      </NConfigProvider>
    `,
  });
  return mount(wrapper, {
    global: {
      plugins: [i18n],
      stubs: {
        NDataTable: {
          props: ["data"],
          template: '<div><div v-for="row in data" :key="row.from">{{ row.from }} {{ row.to }} {{ row.missingCandles }}</div></div>',
        },
        NModal: {
          props: ["show"],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>',
        },
      },
    },
  });
}
