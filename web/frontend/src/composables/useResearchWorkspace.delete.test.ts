import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useResearchWorkspace } from "@/composables/useResearchWorkspace";
import { i18n } from "@/i18n";
import { dataApi } from "@/services/api/data";
import type { DataSyncTask } from "@/types/app";

const dataApiMocks = vi.hoisted(() => ({
  createTask: vi.fn(),
  deleteTask: vi.fn(),
  getCandles: vi.fn(),
  getTaskGaps: vi.fn(),
  listTasks: vi.fn(),
  repairTaskGap: vi.fn(),
  repairTaskGaps: vi.fn(),
  retryTask: vi.fn(),
  setRealtime: vi.fn(),
  setSync: vi.fn(),
}));

const dialogMocks = vi.hoisted(() => ({
  warning: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
  success: vi.fn(),
}));

const routerMocks = vi.hoisted(() => ({
  replace: vi.fn(),
  query: {} as Record<string, string>,
}));

vi.mock("@/services/api/data", () => ({
  dataApi: dataApiMocks,
}));

vi.mock("@/services/api/market", () => ({
  marketApi: {
    listInstrumentSyncStatuses: vi.fn().mockResolvedValue([]),
    listInstruments: vi.fn().mockResolvedValue([]),
  },
}));

vi.mock("naive-ui", () => ({
  useDialog: () => dialogMocks,
  useMessage: () => messageMocks,
}));

vi.mock("vue-router", () => ({
  useRoute: () => ({ query: routerMocks.query }),
  useRouter: () => ({ replace: routerMocks.replace }),
}));

describe("useResearchWorkspace delete task", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routerMocks.query = { exchange: "binance", symbol: "BTCUSDT", interval: "1m" };
    dataApiMocks.deleteTask.mockResolvedValue(undefined);
    dataApiMocks.getCandles.mockResolvedValue({
      candles: [],
      source: "native",
      requestedInterval: "1m",
      health: "insufficient",
      gaps: [],
      coverage: { requestedLimit: 1000, returnedCandles: 0, limitedByBaseWindow: false },
      window: { count: 0 },
      pagination: { hasPrevious: false, hasNext: false },
    });
    dataApiMocks.listTasks.mockResolvedValue([]);
  });

  it("deletes a sync task only after confirmation and refreshes the list", async () => {
    const workspace = mountWorkspace();
    await flushPromises();
    dataApiMocks.listTasks.mockClear();

    workspace.deleteTask(dataSyncTask({ id: "dst_delete" }));

    expect(dataApi.deleteTask).not.toHaveBeenCalled();
    expect(dialogMocks.warning).toHaveBeenCalledWith(expect.objectContaining({
      title: "确认删除同步任务",
      content: "删除 binance / BTCUSDT / 1m 的同步任务记录；已同步的 K 线数据不会被删除。",
    }));

    const options = dialogMocks.warning.mock.calls[0][0] as { onPositiveClick: () => Promise<void> };
    await options.onPositiveClick();
    await flushPromises();

    expect(dataApi.deleteTask).toHaveBeenCalledWith("dst_delete");
    expect(dataApi.listTasks).toHaveBeenCalledTimes(1);
    expect(messageMocks.success).toHaveBeenCalledWith("数据同步任务已删除。");
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
    {
      global: {
        plugins: [i18n],
      },
    },
  );
  if (!holder.workspace) {
    throw new Error("research workspace was not mounted");
  }
  return holder.workspace;
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
    dataHealth: "ok",
    attemptCount: 0,
    createdAt: "2026-06-28T00:00:00Z",
    updatedAt: "2026-06-28T00:00:00Z",
    ...overrides,
  };
}
