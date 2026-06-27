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

  it("lists exchange accounts without credential fields", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify([
          {
            id: "ea_1",
            exchange: "binance",
            alias: "main",
            enabled: true,
            credentialStatus: "encrypted",
            createdAt: "2026-01-01T00:00:00Z",
            updatedAt: "2026-01-01T00:00:00Z",
          },
        ]),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    const accounts = await systemApi.listExchangeAccounts();

    expect(accounts).toEqual([expect.objectContaining({ id: "ea_1", credentialStatus: "encrypted" })]);
    expect(accounts[0]).not.toHaveProperty("apiKey");
    expect(accounts[0]).not.toHaveProperty("apiSecret");
  });

  it("creates exchange accounts without expecting credentials in the response", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          id: "ea_1",
          exchange: "binance",
          alias: "main",
          enabled: true,
          credentialStatus: "encrypted",
          createdAt: "2026-01-01T00:00:00Z",
          updatedAt: "2026-01-01T00:00:00Z",
        }),
        { status: 201 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    const account = await systemApi.createExchangeAccount({
      exchange: "binance",
      alias: "main",
      apiKey: "key",
      apiSecret: "secret",
      enabled: true,
    });

    expect(account).toEqual(
      expect.objectContaining({
        id: "ea_1",
        credentialStatus: "encrypted",
      }),
    );
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/system/exchange-accounts",
      expect.objectContaining({
        body: expect.stringContaining("\"apiKey\":\"key\""),
        method: "POST",
      }),
    );
    expect(account).not.toHaveProperty("apiKey");
    expect(account).not.toHaveProperty("apiSecret");
  });
});
