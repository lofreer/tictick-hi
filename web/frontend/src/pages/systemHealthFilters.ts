import type { ServiceHealth } from "@/types/app";

export type SystemHealthFocusFilter = "all" | "unhealthy" | "stale" | "backoff";

const systemHealthFocusFilters: SystemHealthFocusFilter[] = ["all", "unhealthy", "stale", "backoff"];

export function systemHealthFocusFromQuery(value: unknown): SystemHealthFocusFilter {
  return typeof value === "string" && systemHealthFocusFilters.includes(value as SystemHealthFocusFilter) ? (value as SystemHealthFocusFilter) : "all";
}

export function systemHealthFocusQueryValue(value: SystemHealthFocusFilter) {
  return value === "all" ? undefined : value;
}

export function serviceMatchesSystemHealthFocus(service: ServiceHealth, focus: SystemHealthFocusFilter) {
  if (focus === "unhealthy") return service.status !== "ok";
  if (focus === "stale") return (service.staleLeaseCount ?? 0) > 0;
  if (focus === "backoff") return (service.exchangeBackoffCount ?? 0) > 0;
  return true;
}
