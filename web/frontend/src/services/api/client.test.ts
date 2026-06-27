import { afterEach, describe, expect, it, vi } from "vitest";

import { ApiClient, ApiError } from "@/services/api/client";

describe("api client", () => {
  afterEach(() => {
    document.cookie = "tictick_hi_csrf=; Max-Age=0; path=/";
    vi.unstubAllGlobals();
  });

  it("adds the csrf header to unsafe requests when the csrf cookie is present", async () => {
    document.cookie = "tictick_hi_csrf=csrf-token; path=/";
    const fetchMock = vi.fn(async () => new Response(JSON.stringify({ status: "ok" }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await new ApiClient().post("/system/operators", { username: "ops" });

    expect(fetchMock).toHaveBeenCalledWith("/api/system/operators", expect.objectContaining({ method: "POST" }));
    const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
    const headers = init.headers as Headers;
    expect(headers.get("X-CSRF-Token")).toBe("csrf-token");
  });

  it("does not add the csrf header to safe requests", async () => {
    document.cookie = "tictick_hi_csrf=csrf-token; path=/";
    const fetchMock = vi.fn(async () => new Response(JSON.stringify({ status: "ok" }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await new ApiClient().get("/system/health");

    expect(fetchMock).toHaveBeenCalledWith("/api/system/health", expect.objectContaining({ method: "GET" }));
    const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
    const headers = init.headers as Headers;
    expect(headers.get("X-CSRF-Token")).toBeNull();
  });

  it("uses structured api error details when the server returns them", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(JSON.stringify({ code: "invalid_state", message: "invalid state", error: "invalid state" }), {
          status: 409,
          statusText: "Conflict",
        }),
      ),
    );

    await expect(new ApiClient().post("/data/tasks/dst_1/retry")).rejects.toMatchObject({
      name: "ApiError",
      message: "invalid state",
      status: 409,
      code: "invalid_state",
    } satisfies Partial<ApiError>);
  });

  it("keeps compatibility with legacy error payloads", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(JSON.stringify({ error: "legacy failure" }), { status: 400, statusText: "Bad Request" })),
    );

    await expect(new ApiClient().get("/broken")).rejects.toMatchObject({
      message: "legacy failure",
      status: 400,
    });
  });
});
