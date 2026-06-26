import { describe, expect, it } from "vitest";

import { routes } from "@/router/routes";

function flattenPaths() {
  return routes.flatMap((route) => {
    const parent = route.path === "/" ? "" : route.path;
    return [
      route.path,
      ...(route.children ?? []).map((child) => `${parent}/${child.path}`.replace(/\/$/, "")),
    ];
  });
}

describe("routes", () => {
  it("defines the planned console routes", () => {
    expect(flattenPaths()).toEqual(
      expect.arrayContaining([
        "/login",
        "/overview",
        "/research",
        "/backtests",
        "/backtests/new",
        "/backtests/:id",
        "/trading",
        "/trading/new",
        "/trading/:id",
        "/system/notifications",
        "/system/exchange-accounts",
        "/system/operators",
        "/system/health",
      ]),
    );
  });
});

