import { describe, expect, it } from "vitest";

import { serviceMatchesSystemHealthFocus, systemHealthFocusFromQuery, systemHealthFocusQueryValue } from "@/pages/systemHealthFilters";
import type { ServiceHealth } from "@/types/app";

describe("system health filters", () => {
  it("normalizes system health focus query values", () => {
    expect(systemHealthFocusFromQuery("unhealthy")).toBe("unhealthy");
    expect(systemHealthFocusFromQuery("stale")).toBe("stale");
    expect(systemHealthFocusFromQuery("backoff")).toBe("backoff");
    expect(systemHealthFocusFromQuery("unknown")).toBe("all");
    expect(systemHealthFocusFromQuery(["stale"])).toBe("all");
    expect(systemHealthFocusQueryValue("all")).toBeUndefined();
    expect(systemHealthFocusQueryValue("backoff")).toBe("backoff");
  });

  it("matches services by operational focus", () => {
    expect(serviceMatchesSystemHealthFocus(service({ status: "warning" }), "unhealthy")).toBe(true);
    expect(serviceMatchesSystemHealthFocus(service({ staleLeaseCount: 1 }), "stale")).toBe(true);
    expect(serviceMatchesSystemHealthFocus(service({ exchangeBackoffCount: 2 }), "backoff")).toBe(true);
    expect(serviceMatchesSystemHealthFocus(service({ status: "ok" }), "unhealthy")).toBe(false);
    expect(serviceMatchesSystemHealthFocus(service({}), "all")).toBe(true);
  });
});

function service(overrides: Partial<ServiceHealth>): ServiceHealth {
  return {
    name: "sync-worker",
    status: "ok",
    ...overrides,
  };
}
