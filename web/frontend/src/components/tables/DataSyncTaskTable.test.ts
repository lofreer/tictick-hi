import { mount } from "@vue/test-utils";
import { NDataTable } from "naive-ui";
import { describe, expect, it } from "vitest";

import DataSyncTaskTable from "@/components/tables/DataSyncTaskTable.vue";
import { i18n } from "@/i18n";
import type { DataSyncTask } from "@/types/app";

describe("DataSyncTaskTable", () => {
  it("keeps long errors inside a bounded table viewport", () => {
    const longError =
      'binance klines: Get "https://api.binance.com/api/v3/klines?endTime=1782524388943&interval=1m&limit=500&startTime=1780277926000&symbol=BTCUSDT": EOF';

    const wrapper = mount(DataSyncTaskTable, {
      global: { plugins: [i18n] },
      props: {
        tasks: [
          dataSyncTask({
            id: "sync_1",
            exchange: "binance",
            symbol: "BTCUSDT",
            interval: "1m",
            lastError: longError,
          }),
        ],
      },
    });

    const table = wrapper.findComponent(NDataTable);
    expect(table.props("maxHeight")).toBe(260);
    expect(table.props("scrollX")).toBe(1140);

    const errorText = wrapper.get(".task-error-text");
    expect(errorText.attributes("title")).toBe(longError);
    expect(errorText.text()).toHaveLength(90);
    expect(errorText.text().endsWith("...")).toBe(true);
  });

  it("emits retry for failed tasks", async () => {
    const failedTask = dataSyncTask({
      id: "sync_1",
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      syncEnabled: false,
      lastError: "invalid symbol",
    });
    const wrapper = mount(DataSyncTaskTable, {
      global: { plugins: [i18n] },
      props: {
        tasks: [failedTask],
      },
    });

    await wrapper.get('button[title="重试"]').trigger("click");

    expect(wrapper.emitted("retry")).toEqual([[failedTask]]);
    expect(wrapper.find('button[title="同步"]').exists()).toBe(false);
  });
});

function dataSyncTask(overrides: Partial<DataSyncTask>): DataSyncTask {
  return {
    id: "sync_1",
    exchange: "binance",
    symbol: "BTCUSDT",
    interval: "1m",
    realtimeEnabled: false,
    syncEnabled: true,
    status: "failed",
    attemptCount: 1,
    createdAt: "2026-06-28T00:00:00Z",
    updatedAt: "2026-06-28T00:00:00Z",
    ...overrides,
  };
}
