import { flushPromises, mount } from "@vue/test-utils";
import { NConfigProvider } from "naive-ui";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { defineComponent } from "vue";

import MarketCandleInvalidIssueTag from "@/components/research/MarketCandleInvalidIssueTag.vue";
import { i18n } from "@/i18n";
import { dataApi } from "@/services/api/data";

const dataApiMocks = vi.hoisted(() => ({
  scanMarketCandleInvalidIssues: vi.fn(),
}));

vi.mock("@/services/api/data", () => ({
  dataApi: dataApiMocks,
}));

describe("MarketCandleInvalidIssueTag", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
    vi.clearAllMocks();
    dataApiMocks.scanMarketCandleInvalidIssues.mockResolvedValue({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      window: { count: 4 },
      issues: [
        {
          code: "invalid_open_price",
          message: "open price value must be positive",
          openTime: "2026-06-27T03:01:00Z",
        },
      ],
      limited: false,
      totalCount: 1,
      returnedCount: 1,
    });
  });

  afterEach(() => {
    document.body.innerHTML = "";
  });

  it("loads and displays full-history invalid candle metadata for the selected market", async () => {
    const wrapper = mountTag();
    await flushPromises();

    expect(dataApi.scanMarketCandleInvalidIssues).toHaveBeenCalledWith({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      limit: 20,
    });
    expect(wrapper.text()).toContain("全历史异常 1 根");
    expect(wrapper.find('[role="button"]').attributes("title")).toContain("2026-06-27 03:01:00 UTC");
    expect(wrapper.find('[role="button"]').attributes("title")).toContain("开盘价必须为正");
  });

  it("opens full-history invalid candle details", async () => {
    const wrapper = mountTag();
    await flushPromises();

    await wrapper.find('[role="button"]').trigger("click");
    await flushPromises();

    expect(document.body.textContent).toContain("全历史异常详情");
    expect(document.body.textContent).toContain("2026-06-27 03:01:00 UTC");
    expect(document.body.textContent).toContain("开盘价必须为正");
    expect(document.body.textContent).toContain("open price value must be positive");
  });

  it("shows a healthy full-history invalid scan", async () => {
    dataApiMocks.scanMarketCandleInvalidIssues.mockResolvedValueOnce({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      window: { count: 12 },
      issues: [],
      limited: false,
      totalCount: 0,
      returnedCount: 0,
    });
    const wrapper = mountTag();
    await flushPromises();

    expect(wrapper.text()).toContain("全历史无异常 / 12 根");
  });

  it("shows a failed scan state without throwing", async () => {
    dataApiMocks.scanMarketCandleInvalidIssues.mockRejectedValueOnce(new Error("network"));
    const wrapper = mountTag();
    await flushPromises();

    expect(wrapper.text()).toContain("全历史异常扫描失败");
  });
});

function mountTag() {
  const wrapper = defineComponent({
    components: { MarketCandleInvalidIssueTag, NConfigProvider },
    template: `
      <NConfigProvider>
        <MarketCandleInvalidIssueTag exchange="binance" interval="1m" symbol="BTCUSDT" />
      </NConfigProvider>
    `,
  });
  return mount(wrapper, {
    global: {
      plugins: [i18n],
      stubs: {
        NDataTable: {
          props: ["data"],
          template: '<div><div v-for="row in data" :key="row.openTime">{{ row.openTime }} {{ row.code }} {{ row.message }}</div></div>',
        },
        NModal: {
          props: ["show"],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>',
        },
      },
    },
  });
}
