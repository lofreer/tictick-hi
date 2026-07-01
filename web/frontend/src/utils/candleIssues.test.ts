import { describe, expect, it } from "vitest";

import { isRepairableCandleIssueCode } from "@/utils/candleIssues";

describe("candle issue helpers", () => {
  it("keeps invalid open-time issues out of normal resync repair", () => {
    expect(isRepairableCandleIssueCode("invalid_open_time")).toBe(false);
    expect(isRepairableCandleIssueCode("invalid_close_time")).toBe(true);
    expect(isRepairableCandleIssueCode("invalid_open_price")).toBe(true);
    expect(isRepairableCandleIssueCode(undefined)).toBe(true);
  });
});
