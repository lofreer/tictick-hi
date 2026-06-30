import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useResearchWorkspace } from "@/composables/useResearchWorkspace";
import { i18n } from "@/i18n";
import { dataApi } from "@/services/api/data";
import { marketApi } from "@/services/api/market";
import type { DataSyncTask } from "@/types/app";

const dataApiMocks = vi.hoisted(() => ({
  getCandles: vi.fn(),
  getTaskGaps: vi.fn(),
  listTasks: vi.fn(),
  repairTaskGaps: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
  success: vi.fn(),
}));

const marketApiMocks = vi.hoisted(() => ({
  listInstruments: vi.fn(),
}));

const routerMocks = vi.hoisted(() => ({
  replace: vi.fn(),
  query: {} as Record<string, string>,
}));

vi.mock("@/services/api/data", () => ({
  dataApi: dataApiMocks,
}));

vi.mock("@/services/api/market", () => ({
  marketApi: marketApiMocks,
}));

vi.mock("naive-ui", () => ({
  useDialog: () => ({ warning: vi.fn() }),
  useMessage: () => messageMocks,
}));

vi.mock("vue-router", () => ({
  useRoute: () => ({ query: routerMocks.query }),
  useRouter: () => ({ replace: routerMocks.replace }),
}));

describe("useResearchWorkspace task gap repair feedback", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routerMocks.query = { exchange: "binance", symbol: "BTCUSDT", interval: "1m" };
    dataApiMocks.listTasks.mockResolvedValue([]);
    dataApiMocks.getCandles.mockResolvedValue(candleResult());
    dataApiMocks.getTaskGaps.mockResolvedValue({
      taskId: "dst_1",
      gaps: [{ from: "2026-06-27T03:02:00Z", to: "2026-06-27T03:03:00Z", missingCandles: 1 }],
      limited: false,
      totalCount: 1,
      returnedCount: 1,
      repairLimit: 20,
    });
    dataApiMocks.repairTaskGaps.mockResolvedValue({
      sourceTaskId: "dst_1",
      createdTasks: [dataSyncTask({ id: "dst_repair_1" })],
      skippedExisting: 0,
      limited: false,
      totalCount: 1,
      repairLimit: 20,
    });
    marketApiMocks.listInstruments.mockResolvedValue([]);
  });

  it("opens the gap modal with created repair task metadata", async () => {
    const workspace = mountWorkspace();
    await flushPromises();

    await workspace.repairTaskGaps(dataSyncTask({ id: "dst_1" }));
    await flushPromises();

    expect(dataApi.repairTaskGaps).toHaveBeenCalledWith("dst_1");
    expect(dataApi.listTasks).toHaveBeenCalledTimes(2);
    expect(dataApi.getTaskGaps).toHaveBeenCalledWith("dst_1");
    expect(workspace.gapDetailsModalOpen.value).toBe(true);
    expect(workspace.taskGapRepairResult.value).toMatchObject({
      sourceTaskId: "dst_1",
      createdTasks: [{ id: "dst_repair_1" }],
      skippedExisting: 0,
      totalCount: 1,
      repairLimit: 20,
    });
    expect(workspace.taskGapRepairNotice.value).toBe("已排队 1 个缺口修复任务。");
    expect(workspace.taskGapRepairNoticeType.value).toBe("success");
    expect(messageMocks.success).toHaveBeenCalledWith("已排队 1 个缺口修复任务。");
  });

  it("surfaces skipped and limited repair metadata", async () => {
    dataApiMocks.repairTaskGaps.mockResolvedValueOnce({
      sourceTaskId: "dst_1",
      createdTasks: [],
      skippedExisting: 2,
      limited: true,
      totalCount: 27,
      repairLimit: 20,
    });
    const workspace = mountWorkspace();
    await flushPromises();

    await workspace.repairTaskGaps(dataSyncTask({ id: "dst_1" }));
    await flushPromises();

    expect(workspace.taskGapRepairResult.value).toMatchObject({
      skippedExisting: 2,
      limited: true,
      totalCount: 27,
      repairLimit: 20,
    });
    expect(workspace.taskGapRepairNotice.value).toBe("缺口修复任务已存在。");
    expect(workspace.taskGapRepairNoticeType.value).toBe("success");
  });

  it("clears stale repair metadata when another task gap modal opens", async () => {
    const workspace = mountWorkspace();
    await flushPromises();

    await workspace.repairTaskGaps(dataSyncTask({ id: "dst_1" }));
    await flushPromises();
    expect(workspace.taskGapRepairResult.value).not.toBeNull();

    await workspace.viewTaskGaps(dataSyncTask({ id: "dst_2" }));
    await flushPromises();

    expect(workspace.taskGapRepairResult.value).toBeNull();
    expect(workspace.taskGapRepairNotice.value).toBe("");
  });

  it("keeps raw exchange failures out of the visible notice", async () => {
    dataApiMocks.repairTaskGaps.mockRejectedValueOnce(new Error('Get "https://api.binance.com/api/v3/klines": EOF'));
    const workspace = mountWorkspace();
    await flushPromises();

    await workspace.repairTaskGaps(dataSyncTask({ id: "dst_1" }));
    await flushPromises();

    expect(workspace.gapDetailsModalOpen.value).toBe(true);
    expect(workspace.taskGapRepairNotice.value).toBe("创建任务缺口修复失败。");
    expect(workspace.taskGapRepairNotice.value).not.toContain("api.binance.com");
    expect(messageMocks.error).toHaveBeenCalledWith("创建任务缺口修复失败。");
  });
});

function mountWorkspace() {
  const holder: { workspace?: ReturnType<typeof useResearchWorkspace> } = {};
  mount(
    {
      template: "<div />",
      setup() {
        holder.workspace = useResearchWorkspace();
        return {};
      },
    },
    { global: { plugins: [i18n] } },
  );
  if (!holder.workspace) {
    throw new Error("research workspace was not mounted");
  }
  return holder.workspace;
}

function candleResult() {
  return {
    candles: [],
    source: "native",
    requestedInterval: "1m",
    baseInterval: "1m",
    health: "ok",
    gaps: [],
    issues: [],
    coverage: {
      requestedLimit: 1000,
      returnedCandles: 0,
      limitedByBaseWindow: false,
    },
    window: {
      count: 0,
    },
    pagination: {
      hasPrevious: false,
      hasNext: false,
    },
  };
}

function dataSyncTask(overrides: Partial<DataSyncTask>): DataSyncTask {
  return {
    id: "dst_1",
    exchange: "binance",
    symbol: "BTCUSDT",
    interval: "1m",
    realtimeEnabled: false,
    syncEnabled: false,
    status: "succeeded",
    marketStatus: "active",
    dataHealth: "gap",
    gapSummary: {
      count: 1,
      firstGap: {
        from: "2026-06-27T03:02:00Z",
        to: "2026-06-27T03:03:00Z",
        missingCandles: 1,
      },
    },
    attemptCount: 0,
    createdAt: "2026-06-28T00:00:00Z",
    updatedAt: "2026-06-28T00:00:00Z",
    ...overrides,
  };
}
