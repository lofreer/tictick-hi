/// <reference types="node" />
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

import source from "./SystemHealthPage.vue?raw";

const zhMessages = readFileSync("src/i18n/messages.zh.ts", "utf8");
const enMessages = readFileSync("src/i18n/messages.en.ts", "utf8");

describe("SystemHealthPage layout contract", () => {
  it("filters services by focus query context", () => {
    expect(source).toContain("useRoute");
    expect(source).toContain("useRouter");
    expect(source).toContain("systemHealthFocusFromQuery(route.query.focus)");
    expect(source).toContain("NRadioGroup");
    expect(source).toContain(":value=\"focusFilter\"");
    expect(source).toContain(":aria-label=\"t('system.healthFocusFilter')\"");
    expect(source).toContain("@update:value=\"setFocusFilter\"");
    expect(source).toContain("filteredServices");
    expect(source).toContain("serviceMatchesSystemHealthFocus");
    expect(source).toContain("systemHealthFocusQueryValue");
    for (const messages of [zhMessages, enMessages]) {
      expect(messages).toContain("\"system.healthFocusFilter\"");
      expect(messages).toContain("\"system.healthFocus.backoff\"");
      expect(messages).toContain("\"system.healthFocus.stale\"");
      expect(messages).toContain("\"system.healthFocus.unhealthy\"");
      expect(messages).toContain("\"system.noServicesForFilter\"");
    }
  });
});
