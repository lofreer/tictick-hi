import { flushPromises, mount } from "@vue/test-utils";
import { defineComponent } from "vue";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useResearchRepairTaskPolling } from "@/composables/useResearchRepairTaskPolling";
import type { DataSyncTask } from "@/types/app";

describe("useResearchRepairTaskPolling", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("refreshes immediately and then stops after the bounded attempt count", async () => {
    const loadTasks = vi.fn().mockResolvedValue(undefined);
    const wrapper = mountHost(loadTasks, { intervalMs: 50, maxAttempts: 3 });

    wrapper.vm.startRepairTaskPolling();
    await flushPromises();
    expect(loadTasks).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(50);
    expect(loadTasks).toHaveBeenCalledTimes(2);

    await vi.advanceTimersByTimeAsync(50);
    expect(loadTasks).toHaveBeenCalledTimes(3);

    await vi.advanceTimersByTimeAsync(200);
    expect(loadTasks).toHaveBeenCalledTimes(3);
  });

  it("can start with the first refresh delayed when the caller already refreshed", async () => {
    const loadTasks = vi.fn().mockResolvedValue(undefined);
    const wrapper = mountHost(loadTasks, { intervalMs: 50, maxAttempts: 2 });

    wrapper.vm.startRepairTaskPolling({ immediate: false });
    await flushPromises();
    expect(loadTasks).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(50);
    expect(loadTasks).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(50);
    expect(loadTasks).toHaveBeenCalledTimes(2);
  });

  it("restarts the bounded polling sequence instead of stacking timers", async () => {
    const loadTasks = vi.fn().mockResolvedValue(undefined);
    const wrapper = mountHost(loadTasks, { intervalMs: 50, maxAttempts: 2 });

    wrapper.vm.startRepairTaskPolling({ immediate: false });
    await vi.advanceTimersByTimeAsync(30);
    wrapper.vm.startRepairTaskPolling({ immediate: false });
    await vi.advanceTimersByTimeAsync(49);
    expect(loadTasks).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(1);
    expect(loadTasks).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(50);
    expect(loadTasks).toHaveBeenCalledTimes(2);
  });

  it("clears pending polling when the host unmounts", async () => {
    const loadTasks = vi.fn().mockResolvedValue(undefined);
    const wrapper = mountHost(loadTasks, { intervalMs: 50, maxAttempts: 2 });

    wrapper.vm.startRepairTaskPolling({ immediate: false });
    wrapper.unmount();
    await vi.advanceTimersByTimeAsync(100);

    expect(loadTasks).not.toHaveBeenCalled();
  });

  it("refreshes the chart and stops when watched repair tasks settle", async () => {
    const refreshChart = vi.fn().mockResolvedValue(undefined);
    const loadTasks = vi.fn()
      .mockResolvedValueOnce([dataSyncTask("dst_repair_1", "running")])
      .mockResolvedValueOnce([dataSyncTask("dst_repair_1", "succeeded")]);
    const wrapper = mountHost(loadTasks, { intervalMs: 50, maxAttempts: 4 });

    wrapper.vm.startRepairTaskPolling({
      onSettled: refreshChart,
      repairTaskIds: ["dst_repair_1"],
    });
    await flushPromises();
    expect(refreshChart).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(50);
    await flushPromises();
    expect(loadTasks).toHaveBeenCalledTimes(2);
    expect(refreshChart).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(200);
    expect(loadTasks).toHaveBeenCalledTimes(2);
  });

  it("runs the exhausted callback when watched repair tasks never settle", async () => {
    const refreshChart = vi.fn().mockResolvedValue(undefined);
    const loadTasks = vi.fn().mockResolvedValue([dataSyncTask("dst_repair_1", "running")]);
    const wrapper = mountHost(loadTasks, { intervalMs: 50, maxAttempts: 2 });

    wrapper.vm.startRepairTaskPolling({
      onExhausted: refreshChart,
      repairTaskIds: ["dst_repair_1"],
    });
    await flushPromises();
    expect(refreshChart).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(50);
    await flushPromises();
    expect(loadTasks).toHaveBeenCalledTimes(2);
    expect(refreshChart).toHaveBeenCalledTimes(1);
  });
});

function mountHost(loadTasks: () => Promise<DataSyncTask[] | void>, options: { intervalMs: number; maxAttempts: number }) {
  return mount(defineComponent({
    setup(_, { expose }) {
      const polling = useResearchRepairTaskPolling(loadTasks, options);
      expose(polling);
      return () => null;
    },
  })) as unknown as {
    unmount: () => void;
    vm: ReturnType<typeof useResearchRepairTaskPolling>;
  };
}

function dataSyncTask(id: string, status: DataSyncTask["status"]): DataSyncTask {
  return {
    id,
    exchange: "binance",
    symbol: "BTCUSDT",
    interval: "1m",
    realtimeEnabled: false,
    syncEnabled: true,
    status,
    marketStatus: "active",
    dataHealth: status === "succeeded" ? "ok" : "syncing",
    attemptCount: 0,
    createdAt: "2026-06-27T03:00:00Z",
    updatedAt: "2026-06-27T03:00:00Z",
  };
}
