import { describe, expect, it, vi } from "vitest";

import { dataApi } from "@/services/api/data";

describe("data api invalid details", () => {
  it("loads data sync task invalid issue details", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          taskId: "dst_1",
          issues: [
            {
              code: "invalid_open_price",
              message: "open price value must be positive",
              openTime: "2026-06-27T07:02:00Z",
            },
          ],
          limited: true,
          totalCount: 2,
          returnedCount: 1,
          issueLimit: 50,
          offset: 50,
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(dataApi.getTaskInvalidIssues("dst_1", {
      code: "invalid_close_price",
      from: "2026-06-27T07:00:00.000Z",
      limit: 50,
      offset: 50,
      to: "2026-06-27T08:00:00.000Z",
    })).resolves.toEqual({
      taskId: "dst_1",
      issues: [
        {
          code: "invalid_open_price",
          message: "open price value must be positive",
          openTime: "2026-06-27T07:02:00Z",
        },
      ],
      limited: true,
      totalCount: 2,
      returnedCount: 1,
      issueLimit: 50,
      offset: 50,
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/data/tasks/dst_1/invalid-issues?limit=50&offset=50&code=invalid_close_price&from=2026-06-27T07%3A00%3A00.000Z&to=2026-06-27T08%3A00%3A00.000Z",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("queues repair tasks for data sync task invalid issues", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          sourceTaskId: "dst_1",
          createdTasks: [
            {
              id: "dst_invalid_repair_1",
              exchange: "binance",
              symbol: "BTCUSDT",
              interval: "1m",
              startTime: "2026-06-27T07:02:00Z",
              endTime: "2026-06-27T07:03:00Z",
              repairSourceTaskId: "dst_1",
              realtimeEnabled: false,
              syncEnabled: true,
              status: "pending",
              marketStatus: "active",
              dataHealth: "syncing",
              attemptCount: 0,
              createdAt: "2026-06-27T07:02:01Z",
              updatedAt: "2026-06-27T07:02:01Z",
            },
          ],
          skippedExisting: 0,
          limited: false,
          totalCount: 1,
          repairLimit: 20,
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(dataApi.repairTaskInvalidIssues("dst_1", {
      code: "invalid_open_price",
      from: "2026-06-27T07:00:00.000Z",
      to: "2026-06-27T08:00:00.000Z",
    })).resolves.toMatchObject({
      sourceTaskId: "dst_1",
      createdTasks: [{ id: "dst_invalid_repair_1", repairSourceTaskId: "dst_1", syncEnabled: true }],
      totalCount: 1,
      repairLimit: 20,
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/data/tasks/dst_1/repair-invalid-issues",
      expect.objectContaining({
        body: JSON.stringify({
          code: "invalid_open_price",
          from: "2026-06-27T07:00:00.000Z",
          to: "2026-06-27T08:00:00.000Z",
        }),
        method: "POST",
      }),
    );
  });
});
