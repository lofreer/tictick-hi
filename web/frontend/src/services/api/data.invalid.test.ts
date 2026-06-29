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
});
