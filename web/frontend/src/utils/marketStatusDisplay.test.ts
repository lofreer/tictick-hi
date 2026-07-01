import { describe, expect, it } from "vitest";

import { i18n } from "@/i18n";
import {
  marketStatusDetailLabel,
  marketStatusExchangeDetail,
  marketStatusLabel,
  type Translate,
} from "@/utils/marketStatusDisplay";

const t = i18n.global.t as Translate;

describe("market status display", () => {
  it("formats status and exchange detail without leaking raw exchange codes", () => {
    expect(marketStatusLabel(t, "active", "TRADING")).toBe("可用 · 交易中");
    expect(marketStatusLabel(t, "inactive", "BREAK")).toBe("不可用 · 暂停交易");
    expect(marketStatusLabel(t, "inactive", "not_returned")).toBe("不可用 · 交易所未返回");
    expect(marketStatusLabel(t, "missing", "missing")).toBe("未入库");
  });

  it("formats strategy form exchange status hints", () => {
    expect(marketStatusExchangeDetail(t, "TRADING")).toBe("交易所状态：交易中");
    expect(marketStatusExchangeDetail(t, "BREAK")).toBe("交易所状态：暂停交易");
  });

  it("keeps unknown exchange detail visible for auditability", () => {
    expect(marketStatusDetailLabel(t, "HALT_ONLY")).toBe("HALT_ONLY");
    expect(marketStatusLabel(t, "inactive", "HALT_ONLY")).toBe("不可用 · HALT_ONLY");
  });
});
