/// <reference types="node" />
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

import source from "./SystemAuditEventsPage.vue?raw";

const zhMessages = readFileSync("src/i18n/messages.zh.ts", "utf8");
const enMessages = readFileSync("src/i18n/messages.en.ts", "utf8");

describe("SystemAuditEventsPage layout contract", () => {
  it("exposes audit CSV export from the page header", () => {
    expect(source).toContain("Download");
    expect(source).toContain("NSpace");
    expect(source).toContain('href="/api/system/audit-events/export?limit=100"');
    expect(source).toContain('t("system.exportAuditEvents")');
    expect(source).toContain("@click=\"loadEvents\"");
    expect(zhMessages).toContain("\"system.exportAuditEvents\"");
    expect(enMessages).toContain("\"system.exportAuditEvents\"");
  });
});
