import { flushPromises, mount } from "@vue/test-utils";
import { NConfigProvider } from "naive-ui";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { defineComponent } from "vue";

import MarketCandleInvalidIssueTag from "@/components/research/MarketCandleInvalidIssueTag.vue";
import { i18n } from "@/i18n";
import { dataApi } from "@/services/api/data";
import { formatCompactDateTime } from "@/utils/displayText";

const dataApiMocks = vi.hoisted(() => ({
  repairMarketCandleInvalidIssues: vi.fn(),
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
    dataApiMocks.repairMarketCandleInvalidIssues.mockResolvedValue({
      sourceTaskId: "",
      createdTasks: [
        {
          id: "dst_market_invalid_repair_1",
          exchange: "binance",
          symbol: "BTCUSDT",
          interval: "1m",
          startTime: "2026-06-27T03:01:00Z",
          endTime: "2026-06-27T03:02:00Z",
          realtimeEnabled: false,
          syncEnabled: true,
          status: "pending",
          dataHealth: "syncing",
        },
      ],
      skippedExisting: 1,
      limited: false,
      totalCount: 2,
      repairLimit: 100,
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

  it("queues repairs for returned full-history invalid candle details", async () => {
    const wrapper = mountTag();
    await flushPromises();

    await wrapper.find('[role="button"]').trigger("click");
    await flushPromises();
    expect(document.body.textContent).toContain("排队补同步当前异常");
    const repairButton = Array.from(document.querySelectorAll("button"))
      .find((button) => button.textContent?.includes("排队补同步当前异常"));
    expect(repairButton).toBeTruthy();
    repairButton?.click();
    await flushPromises();

    expect(dataApi.repairMarketCandleInvalidIssues).toHaveBeenCalledWith({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      openTimes: ["2026-06-27T03:01:00Z"],
    });
    expect(document.body.textContent).toContain("本次匹配 2 个，已创建 1 个，跳过 1 个，单次上限 100");
    expect(document.body.textContent).toContain("dst_market_invalid_repair_1");
    expect(document.body.textContent).toContain(
      `${formatCompactDateTime("2026-06-27T03:01:00Z")} - ${formatCompactDateTime("2026-06-27T03:02:00Z")}`,
    );
    expect(wrapper.findComponent(MarketCandleInvalidIssueTag).emitted("repaired")).toHaveLength(1);
    expect(dataApi.scanMarketCandleInvalidIssues).toHaveBeenCalledTimes(2);
  });

  it("does not expose normal repair action for invalid open-time full-history issues", async () => {
    dataApiMocks.scanMarketCandleInvalidIssues.mockResolvedValueOnce({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      window: { count: 4 },
      issues: [
        {
          code: "invalid_open_time",
          message: "open time is not aligned to interval",
          openTime: "2026-06-27T03:01:30Z",
        },
      ],
      limited: false,
      totalCount: 1,
      returnedCount: 1,
    });
    const wrapper = mountTag();
    await flushPromises();

    await wrapper.find('[role="button"]').trigger("click");
    await flushPromises();

    expect(document.body.textContent).toContain("K 线开盘时间未对齐周期");
    expect(document.body.textContent).not.toContain("排队补同步当前异常");
    expect(dataApi.repairMarketCandleInvalidIssues).not.toHaveBeenCalled();
  });

  it("keeps the repair result visible while showing the refreshed healthy scan", async () => {
    dataApiMocks.scanMarketCandleInvalidIssues
      .mockResolvedValueOnce({
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
      })
      .mockResolvedValueOnce({
        exchange: "binance",
        symbol: "BTCUSDT",
        interval: "1m",
        window: { count: 4 },
        issues: [],
        limited: false,
        totalCount: 0,
        returnedCount: 0,
      });
    const wrapper = mountTag();
    await flushPromises();

    await wrapper.find('[role="button"]').trigger("click");
    await flushPromises();
    const repairButton = Array.from(document.querySelectorAll("button"))
      .find((button) => button.textContent?.includes("排队补同步当前异常"));
    repairButton?.click();
    await flushPromises();

    expect(document.body.textContent).toContain("当前数据源全历史未检测到异常 K 线。");
    expect(document.body.textContent).toContain("本次匹配 2 个，已创建 1 个，跳过 1 个，单次上限 100");
    expect(document.body.textContent).toContain("dst_market_invalid_repair_1");
    expect(document.body.textContent).not.toContain("open price value must be positive");
    expect(dataApi.scanMarketCandleInvalidIssues).toHaveBeenCalledTimes(2);
  });

  it("does not expose raw provider URLs when full-history invalid repair fails", async () => {
    dataApiMocks.repairMarketCandleInvalidIssues.mockRejectedValueOnce(
      new Error('binance klines: Get "https://api.binance.com/api/v3/klines?symbol=BTCUSDT": EOF'),
    );
    const wrapper = mountTag();
    await flushPromises();

    await wrapper.find('[role="button"]').trigger("click");
    await flushPromises();
    const repairButton = Array.from(document.querySelectorAll("button"))
      .find((button) => button.textContent?.includes("排队补同步当前异常"));
    repairButton?.click();
    await flushPromises();

    expect(document.body.textContent).toContain("全历史异常补同步失败");
    expect(document.body.textContent).not.toContain("api.binance.com");
    expect(document.body.textContent).not.toContain("symbol=BTCUSDT");
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
        NButton: {
          emits: ["click"],
          props: ["loading"],
          template: '<button :disabled="loading" @click="$emit(\'click\')"><slot /></button>',
        },
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
