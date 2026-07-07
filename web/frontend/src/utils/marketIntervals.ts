import type { SelectOption } from "naive-ui";

export const FIRST_VERSION_MARKET_INTERVALS = ["1m", "5m", "15m", "1h", "4h", "1d"] as const;

export type FirstVersionMarketInterval = (typeof FIRST_VERSION_MARKET_INTERVALS)[number];

export function marketIntervalOptions(intervals: readonly string[] = FIRST_VERSION_MARKET_INTERVALS): SelectOption[] {
  return intervals.map((interval) => ({ label: interval, value: interval }));
}
