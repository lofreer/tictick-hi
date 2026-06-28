import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useResearchWorkspace } from "@/composables/useResearchWorkspace";
import { i18n } from "@/i18n";
import { dataApi } from "@/services/api/data";

const dataApiMocks = vi.hoisted(() => ({
  createTask: vi.fn(),
  deleteTask: vi.fn(),
  getCandles: vi.fn(),
  listTasks: vi.fn(),
  retryTask: vi.fn(),
  setRealtime: vi.fn(),
  setSync: vi.fn(),
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

vi.mock("naive-ui", () => ({
  useDialog: () => ({ warning: vi.fn() }),
  useMessage: () => messageMocks,
}));

vi.mock("vue-router", () => ({
  useRoute: () => ({ query: routerMocks.query }),
  useRouter: () => ({ replace: routerMocks.replace }),
}));

describe("useResearchWorkspace", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routerMocks.query = { exchange: "binance", symbol: "BTCUSDT", interval: "5m" };
    dataApiMocks.listTasks.mockResolvedValue([]);
    dataApiMocks.getCandles.mockResolvedValue(candleResult({ gaps: [] }));
    dataApiMocks.createTask.mockResolvedValue({ id: "dst_repair" });
    dataApiMocks.setSync.mockResolvedValue({ id: "dst_repair" });
  });

  it("queues a sync task for the first chart gap using the base interval", async () => {
    dataApiMocks.getCandles.mockResolvedValue(
      candleResult({
        baseInterval: "1m",
        gaps: [{ from: "2026-06-28T00:01:00Z", to: "2026-06-28T00:03:00Z", missingCandles: 2 }],
      }),
    );

    const workspace = mountWorkspace();
    await flushPromises();

    expect(workspace.canRepairGap.value).toBe(true);

    await workspace.repairFirstGap();
    await flushPromises();

    expect(dataApi.createTask).toHaveBeenCalledWith({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      startTime: "2026-06-28T00:01:00Z",
      endTime: "2026-06-28T00:03:00Z",
    });
    expect(dataApi.setSync).toHaveBeenCalledWith("dst_repair", true);
    expect(dataApi.listTasks).toHaveBeenCalledTimes(2);
    expect(messageMocks.success).toHaveBeenCalledWith("缺口修复任务已排队。");
  });

  it("does not create a repair task when the current chart has no gaps", async () => {
    const workspace = mountWorkspace();
    await flushPromises();

    expect(workspace.canRepairGap.value).toBe(false);

    await workspace.repairFirstGap();

    expect(dataApi.createTask).not.toHaveBeenCalled();
    expect(dataApi.setSync).not.toHaveBeenCalled();
    expect(messageMocks.error).toHaveBeenCalledWith("当前没有可修复缺口。");
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

function candleResult(overrides: Record<string, unknown>) {
  return {
    candles: [],
    source: "aggregated",
    requestedInterval: "5m",
    baseInterval: "1m",
    health: "ok",
    gaps: [],
    coverage: {
      requestedLimit: 1000,
      returnedCandles: 0,
      limitedByBaseWindow: false,
    },
    ...overrides,
  };
}
