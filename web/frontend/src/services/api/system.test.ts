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
          name: "Telegram Ops",
          provider: "telegram",
          target: "telegram://send?chat_id=ops&token_env=TELEGRAM_BOT_TOKEN",
          enabled: true,
          createdAt: "2026-01-01T00:00:00Z",
          updatedAt: "2026-01-01T00:00:00Z",
        }),
        { status: 201 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    const channel = await systemApi.createNotificationChannel({
      name: "Telegram Ops",
      provider: "telegram",
      target: "telegram://send?chat_id=ops&token_env=TELEGRAM_BOT_TOKEN",
      enabled: true,
    });

    expect(channel.id).toBe("nc_1");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/system/notifications/channels",
      expect.objectContaining({
        method: "POST",
        body: expect.stringContaining("\"provider\":\"telegram\""),
      }),
    );
  });

	it("updates notification channel enabled state with explicit actions", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: "nc_1",
            name: "Ops",
            provider: "local",
            target: "default",
            enabled: false,
            createdAt: "2026-01-01T00:00:00Z",
            updatedAt: "2026-01-01T00:00:00Z",
          }),
          { status: 200 },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: "nc_1",
            name: "Ops",
            provider: "local",
            target: "default",
            enabled: true,
            createdAt: "2026-01-01T00:00:00Z",
            updatedAt: "2026-01-01T00:00:00Z",
          }),
          { status: 200 },
        ),
      );
    vi.stubGlobal("fetch", fetchMock);

    await expect(systemApi.setNotificationChannelEnabled("nc_1", false)).resolves.toEqual(
      expect.objectContaining({ id: "nc_1", enabled: false }),
    );
    await expect(systemApi.setNotificationChannelEnabled("nc_1", true)).resolves.toEqual(
      expect.objectContaining({ id: "nc_1", enabled: true }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "/api/system/notifications/channels/nc_1/disable",
      expect.objectContaining({ method: "POST" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/system/notifications/channels/nc_1/enable",
      expect.objectContaining({ method: "POST" }),
		);
	});

	it("updates and deletes notification channels", async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValueOnce(
				new Response(
					JSON.stringify({
						id: "nc_1",
						name: "Ops Email",
						provider: "email",
						target: "smtp://smtp.example.com:587?from=bot@example.com&to=ops@example.com",
						enabled: true,
						createdAt: "2026-01-01T00:00:00Z",
						updatedAt: "2026-01-01T00:02:00Z",
					}),
					{ status: 200 },
				),
			)
			.mockResolvedValueOnce(new Response(null, { status: 204 }));
		vi.stubGlobal("fetch", fetchMock);

		const request = {
			name: "Ops Email",
			provider: "email",
			target: "smtp://smtp.example.com:587?from=bot@example.com&to=ops@example.com",
			enabled: true,
		};
		await expect(systemApi.updateNotificationChannel("nc_1", request)).resolves.toEqual(
			expect.objectContaining({ id: "nc_1", name: "Ops Email" }),
		);
		await expect(systemApi.deleteNotificationChannel("nc_1")).resolves.toBeUndefined();
		expect(fetchMock).toHaveBeenNthCalledWith(
			1,
			"/api/system/notifications/channels/nc_1",
			expect.objectContaining({
				method: "PUT",
				body: expect.stringContaining("\"provider\":\"email\""),
			}),
		);
		expect(fetchMock).toHaveBeenNthCalledWith(
			2,
			"/api/system/notifications/channels/nc_1",
			expect.objectContaining({ method: "DELETE" }),
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

  it("updates operator enabled state with explicit actions", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: "op_1",
            username: "ops",
            enabled: false,
            createdAt: "2026-01-01T00:00:00Z",
            updatedAt: "2026-01-01T00:00:00Z",
          }),
          { status: 200 },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: "op_1",
            username: "ops",
            enabled: true,
            createdAt: "2026-01-01T00:00:00Z",
            updatedAt: "2026-01-01T00:00:00Z",
          }),
          { status: 200 },
        ),
      );
    vi.stubGlobal("fetch", fetchMock);

    await expect(systemApi.setOperatorEnabled("op_1", false)).resolves.toEqual(
      expect.objectContaining({ id: "op_1", enabled: false }),
    );
    await expect(systemApi.setOperatorEnabled("op_1", true)).resolves.toEqual(
      expect.objectContaining({ id: "op_1", enabled: true }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "/api/system/operators/op_1/disable",
      expect.objectContaining({ method: "POST" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/system/operators/op_1/enable",
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("lists audit events with an explicit limit", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify([
          {
            id: "ae_1",
            actorOperatorId: "op_1",
            actorUsername: "admin",
            action: "operator.disable",
            resourceType: "operator",
            resourceId: "op_2",
            outcome: "success",
            metadata: { enabled: "false" },
            createdAt: "2026-01-01T00:00:00Z",
          },
        ]),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(systemApi.listAuditEvents(50)).resolves.toEqual([
      expect.objectContaining({ id: "ae_1", action: "operator.disable" }),
    ]);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/system/audit-events?limit=50",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("lists paginated audit events with cursor", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          events: [
            {
              id: "ae_1",
              actorOperatorId: "op_1",
              actorUsername: "admin",
              action: "operator.disable",
              resourceType: "operator",
              resourceId: "op_2",
              outcome: "success",
              metadata: { enabled: "false" },
              createdAt: "2026-01-01T00:00:00Z",
            },
          ],
          nextCursor: "next cursor",
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(systemApi.listAuditEventPage(50, "older cursor")).resolves.toEqual(
      expect.objectContaining({
        events: [expect.objectContaining({ id: "ae_1", action: "operator.disable" })],
        nextCursor: "next cursor",
      }),
    );
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/system/audit-events/page?limit=50&cursor=older+cursor",
      expect.objectContaining({ method: "GET" }),
    );
  });
});
