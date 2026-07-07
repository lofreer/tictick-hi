/// <reference types="node" />
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

import source from "./OverviewPage.vue?raw";

const zhMessages = readFileSync("src/i18n/messages.zh.ts", "utf8");
const enMessages = readFileSync("src/i18n/messages.en.ts", "utf8");

describe("OverviewPage layout contract", () => {
  it("renders overview summary cards as navigation entries", () => {
    expect(source).toContain("<RouterLink v-for=\"card in summaryCards\"");
    expect(source).toContain(":to=\"card.to\"");
    expect(source).toContain(":aria-label=\"t('overview.openMetric', { target: card.label })\"");
    expect(source).toContain("ChevronRight");
    for (const messages of [zhMessages, enMessages]) {
      expect(messages).toContain("\"overview.openMetric\"");
    }
  });

  it("exposes a compact recent activity time window control", () => {
    expect(source).toContain("NRadioGroup");
    expect(source).toContain("NRadioButton");
    expect(source).toContain(":value=\"recentActivityWindow\"");
    expect(source).toContain(":aria-label=\"t('overview.recentActivityWindow')\"");
    expect(source).toContain("@update:value=\"setRecentActivityWindow\"");
    expect(source).toContain("recentActivityWindowOptions");
    for (const messages of [zhMessages, enMessages]) {
      expect(messages).toContain("\"overview.recentActivityWindow\"");
      expect(messages).toContain("\"overview.recentWindow.24h\"");
      expect(messages).toContain("\"overview.recentWindow.7d\"");
      expect(messages).toContain("\"overview.recentWindow.30d\"");
    }
  });
});
