import { readFileSync } from "node:fs";

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

function flattenComponentRoutes() {
  return routes.flatMap((route) => [
    route,
    ...(route.children ?? []),
  ]).filter((route) => "component" in route);
}

function authenticatedStaticPaths() {
  return routes.flatMap((route) => {
    if (route.path !== "/" || !route.children) return [];
    return route.children
      .filter((child) => "component" in child)
      .map((child) => `/${child.path}`)
      .filter((path) => !path.includes(":"));
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
        "/system/sessions",
        "/system/audit-events",
        "/system/health",
      ]),
    );
  });

  it("lazy loads route components to keep the entry chunk small", () => {
    for (const route of flattenComponentRoutes()) {
      expect(typeof route.component).toBe("function");
    }
  });

  it("keeps the visual smoke in sync with authenticated static routes", () => {
    const smokeSource = readFileSync("../../scripts/stage8-visual-smoke.mjs", "utf8");

    for (const path of authenticatedStaticPaths()) {
      expect(smokeSource).toContain(`path: "${path}"`);
    }
  });
});
