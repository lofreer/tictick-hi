import { afterEach, describe, expect, it, vi } from "vitest";

import { systemApi } from "@/services/api/system";

describe("system api", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("creates notification channels", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          id: "nc_1",
          name: "Ops",
          provider: "webhook-demo",
          target: "demo://ops",
          enabled: true,
          createdAt: "2026-01-01T00:00:00Z",
          updatedAt: "2026-01-01T00:00:00Z",
        }),
        { status: 201 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    const channel = await systemApi.createNotificationChannel({
      name: "Ops",
      provider: "webhook-demo",
      target: "demo://ops",
      enabled: true,
    });

    expect(channel.id).toBe("nc_1");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/system/notifications/channels",
      expect.objectContaining({
        method: "POST",
        body: expect.stringContaining("\"provider\":\"webhook-demo\""),
      }),
    );
  });

  it("lists and retries notifications", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response(JSON.stringify([{ id: "nt_1", status: "failed" }]), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ id: "nt_1", status: "pending" }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(systemApi.listNotifications()).resolves.toEqual([{ id: "nt_1", status: "failed" }]);
    await expect(systemApi.retryNotification("nt_1")).resolves.toEqual({ id: "nt_1", status: "pending" });
    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "/api/system/notifications",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/system/notifications/nt_1/retry",
      expect.objectContaining({ method: "POST" }),
    );
  });
});
