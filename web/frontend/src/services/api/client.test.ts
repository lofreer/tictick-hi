import { afterEach, describe, expect, it, vi } from "vitest";

import { ApiClient } from "@/services/api/client";

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
});
