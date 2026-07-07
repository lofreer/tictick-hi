/// <reference types="node" />
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

import source from "./TradingPage.vue?raw";

const zhMessages = readFileSync("src/i18n/messages.zh.ts", "utf8");
const enMessages = readFileSync("src/i18n/messages.en.ts", "utf8");

describe("TradingPage layout contract", () => {
  it("filters trading tasks by status query context", () => {
    expect(source).toContain("useRoute");
    expect(source).toContain("useRouter");
    expect(source).toContain("taskStatusFilterFromQuery(route.query.status)");
    expect(source).toContain("NRadioGroup");
    expect(source).toContain(":value=\"statusFilter\"");
    expect(source).toContain(":aria-label=\"t('trading.statusFilter')\"");
    expect(source).toContain("@update:value=\"setStatusFilter\"");
    expect(source).toContain("filteredTasks");
    expect(source).toContain("taskMatchesStatusFilter");
    expect(source).toContain("taskStatusQueryValue");
    for (const messages of [zhMessages, enMessages]) {
      expect(messages).toContain("\"trading.statusFilter\"");
      expect(messages).toContain("\"trading.status.all\"");
      expect(messages).toContain("\"trading.noTasksForFilter\"");
      expect(messages).toContain("\"trading.tasks\"");
    }
  });
});
