import { afterEach, describe, expect, it, vi } from "vitest";

import { authApi } from "@/services/api/auth";

describe("auth api", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("lists operator sessions", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify([
          {
            id: "os_1",
            operatorId: "op_1",
            createdAt: "2026-01-01T00:00:00Z",
            expiresAt: "2026-01-02T00:00:00Z",
            current: true,
          },
        ]),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(authApi.listSessions()).resolves.toEqual([
      expect.objectContaining({ id: "os_1", current: true }),
    ]);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/auth/sessions",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("revokes operator sessions by id", async () => {
    const fetchMock = vi.fn(async () => new Response(JSON.stringify({ status: "ok" }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(authApi.revokeSession("os_1/unsafe")).resolves.toBeUndefined();
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/auth/sessions/os_1%2Funsafe",
      expect.objectContaining({ method: "DELETE" }),
    );
  });
});
