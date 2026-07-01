import { flushPromises, mount } from "@vue/test-utils";
import { NConfigProvider, NMessageProvider } from "naive-ui";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { defineComponent, type PropType } from "vue";

import MarketCandleGapTag from "@/components/research/MarketCandleGapTag.vue";
import { i18n } from "@/i18n";
import { dataApi } from "@/services/api/data";
import type { DataSyncTask } from "@/types/app";
import { formatCompactDateTime } from "@/utils/displayText";

const dataApiMocks = vi.hoisted(() => ({
  repairMarketCandleGap: vi.fn(),
  repairMarketCandleGaps: vi.fn(),
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
    dataApiMocks.repairMarketCandleGaps.mockResolvedValue({
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
      skippedExisting: 1,
      limited: false,
      totalCount: 2,
      repairLimit: 100,
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

  it("opens full-history gap details from the keyboard", async () => {
    const wrapper = mountTag();
    await flushPromises();

    await wrapper.find('[role="button"]').trigger("keydown.space");
    await flushPromises();

    expect(document.body.textContent).toContain("全历史缺口详情");
    expect(document.body.textContent).toContain("2026-06-27 03:02:00 UTC");
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
    expect(document.body.textContent).toContain("本次匹配 1 个，已创建 1 个，跳过 0 个，单次上限 1");
    expect(document.body.textContent).toContain("dst_market_repair_1");
    expect(document.body.textContent).toContain(
      `${formatCompactDateTime("2026-06-27T03:02:00Z")} - ${formatCompactDateTime("2026-06-27T03:03:00Z")}`,
    );
    expect(wrapper.findComponent(MarketCandleGapTag).emitted("repaired")).toHaveLength(1);
    expect(dataApi.scanMarketCandleGaps).toHaveBeenCalledTimes(2);
  });

  it("keeps the repair result visible while showing the refreshed healthy scan", async () => {
    dataApiMocks.scanMarketCandleGaps
      .mockResolvedValueOnce({
        exchange: "binance",
        symbol: "BTCUSDT",
        interval: "1m",
        window: { count: 4 },
        gaps: [{ from: "2026-06-27T03:02:00Z", to: "2026-06-27T03:03:00Z", missingCandles: 1 }],
        limited: false,
        totalCount: 1,
        returnedCount: 1,
      })
      .mockResolvedValueOnce({
        exchange: "binance",
        symbol: "BTCUSDT",
        interval: "1m",
        window: { count: 5 },
        gaps: [],
        limited: false,
        totalCount: 0,
        returnedCount: 0,
      });
    const wrapper = mountTag();
    await flushPromises();

    await wrapper.find('[role="button"]').trigger("click");
    await flushPromises();
    const repairButton = Array.from(document.body.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("修复首个缺口"),
    );
    repairButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    await flushPromises();

    expect(document.body.textContent).toContain("当前数据源全历史未检测到缺口。");
    expect(document.body.textContent).toContain("本次匹配 1 个，已创建 1 个，跳过 0 个，单次上限 1");
    expect(document.body.textContent).toContain("dst_market_repair_1");
    expect(document.body.textContent).not.toContain("2026-06-27T03:02:00Z 2026-06-27T03:03:00Z 1");
    expect(dataApi.scanMarketCandleGaps).toHaveBeenCalledTimes(2);
  });

  it("refreshes the full-history gap scan after queued repair tasks settle", async () => {
    dataApiMocks.scanMarketCandleGaps
      .mockResolvedValueOnce({
        exchange: "binance",
        symbol: "BTCUSDT",
        interval: "1m",
        window: { count: 4 },
        gaps: [{ from: "2026-06-27T03:02:00Z", to: "2026-06-27T03:03:00Z", missingCandles: 1 }],
        limited: false,
        totalCount: 1,
        returnedCount: 1,
      })
      .mockResolvedValueOnce({
        exchange: "binance",
        symbol: "BTCUSDT",
        interval: "1m",
        window: { count: 4 },
        gaps: [{ from: "2026-06-27T03:02:00Z", to: "2026-06-27T03:03:00Z", missingCandles: 1 }],
        limited: false,
        totalCount: 1,
        returnedCount: 1,
      })
      .mockResolvedValueOnce({
        exchange: "binance",
        symbol: "BTCUSDT",
        interval: "1m",
        window: { count: 5 },
        gaps: [],
        limited: false,
        totalCount: 0,
        returnedCount: 0,
      });
    const wrapper = mountTag();
    await flushPromises();

    await wrapper.find('[role="button"]').trigger("click");
    await flushPromises();
    const repairButton = Array.from(document.body.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("修复首个缺口"),
    );
    repairButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    await flushPromises();
    expect(dataApi.scanMarketCandleGaps).toHaveBeenCalledTimes(2);

    await wrapper.setProps({ tasks: [dataSyncTask({ id: "dst_market_repair_1", status: "running", dataHealth: "syncing" })] });
    await flushPromises();
    expect(dataApi.scanMarketCandleGaps).toHaveBeenCalledTimes(2);

    await wrapper.setProps({ tasks: [dataSyncTask({ id: "dst_market_repair_1", status: "succeeded", dataHealth: "ok" })] });
    await flushPromises();

    expect(dataApi.scanMarketCandleGaps).toHaveBeenCalledTimes(3);
    expect(document.body.textContent).toContain("当前数据源全历史未检测到缺口。");
    expect(document.body.textContent).toContain("本次匹配 1 个，已创建 1 个，跳过 0 个，单次上限 1");

    await wrapper.setProps({ tasks: [dataSyncTask({ id: "dst_market_repair_1", status: "succeeded", dataHealth: "ok" })] });
    await flushPromises();
    expect(dataApi.scanMarketCandleGaps).toHaveBeenCalledTimes(3);
  });

  it("repairs all returned full-history gaps", async () => {
    dataApiMocks.scanMarketCandleGaps.mockResolvedValue({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      window: { count: 8 },
      gaps: [
        { from: "2026-06-27T03:02:00Z", to: "2026-06-27T03:03:00Z", missingCandles: 1 },
        { from: "2026-06-27T03:05:00Z", to: "2026-06-27T03:07:00Z", missingCandles: 2 },
      ],
      limited: false,
      totalCount: 2,
      returnedCount: 2,
    });
    const wrapper = mountTag();
    await flushPromises();

    await wrapper.find('[role="button"]').trigger("click");
    await flushPromises();

    const repairReturnedButton = Array.from(document.body.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("修复当前 2 个"),
    );
    expect(repairReturnedButton).toBeDefined();
    repairReturnedButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    await flushPromises();

    expect(dataApi.repairMarketCandleGaps).toHaveBeenCalledWith({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      gaps: [
        { from: "2026-06-27T03:02:00Z", to: "2026-06-27T03:03:00Z" },
        { from: "2026-06-27T03:05:00Z", to: "2026-06-27T03:07:00Z" },
      ],
    });
    expect(document.body.textContent).toContain("本次匹配 2 个，已创建 1 个，跳过 1 个，单次上限 100");
    expect(document.body.textContent).toContain("dst_market_repair_1");
    expect(wrapper.findComponent(MarketCandleGapTag).emitted("repaired")).toHaveLength(1);
    expect(dataApi.scanMarketCandleGaps).toHaveBeenCalledTimes(2);
  });

  it("does not expose raw provider URLs when full-history gap repair fails", async () => {
    dataApiMocks.repairMarketCandleGaps.mockRejectedValueOnce(
      new Error('binance klines: Get "https://api.binance.com/api/v3/klines?symbol=BTCUSDT": EOF'),
    );
    const wrapper = mountTag();
    await flushPromises();

    await wrapper.find('[role="button"]').trigger("click");
    await flushPromises();

    const repairReturnedButton = Array.from(document.body.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("修复当前 1 个"),
    );
    expect(repairReturnedButton).toBeDefined();
    repairReturnedButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    await flushPromises();

    expect(document.body.textContent).toContain("创建全历史缺口修复失败");
    expect(document.body.textContent).not.toContain("api.binance.com");
    expect(document.body.textContent).not.toContain("symbol=BTCUSDT");
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
    props: { tasks: { type: Array as PropType<DataSyncTask[]>, default: () => [] } },
    template: `
      <NConfigProvider>
        <NMessageProvider>
          <MarketCandleGapTag exchange="binance" interval="1m" symbol="BTCUSDT" :tasks="tasks" />
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

function dataSyncTask(overrides: Partial<DataSyncTask>): DataSyncTask {
  return {
    id: "dst_market_repair_1",
    exchange: "binance",
    symbol: "BTCUSDT",
    interval: "1m",
    realtimeEnabled: false,
    syncEnabled: true,
    status: "pending",
    marketStatus: "active",
    dataHealth: "syncing",
    attemptCount: 0,
    createdAt: "2026-06-27T03:00:00Z",
    updatedAt: "2026-06-27T03:00:00Z",
    ...overrides,
  };
}
