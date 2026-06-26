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
          provider: "webhook",
          target: "https://example.test",
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
      provider: "webhook",
      target: "https://example.test",
      enabled: true,
    });

    expect(channel.id).toBe("nc_1");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/system/notifications/channels",
      expect.objectContaining({
        method: "POST",
        body: expect.stringContaining("\"provider\":\"webhook\""),
      }),
    );
  });
});
