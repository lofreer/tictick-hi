/// <reference types="node" />
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

import source from "./SystemAuditEventsPage.vue?raw";

const zhMessages = readFileSync("src/i18n/messages.zh.ts", "utf8");
const enMessages = readFileSync("src/i18n/messages.en.ts", "utf8");

describe("SystemAuditEventsPage layout contract", () => {
  it("exposes audit CSV export from the page header", () => {
    expect(source).toContain("Download");
    expect(source).toContain("ChevronDown");
    expect(source).toContain("ShieldCheck");
    expect(source).toContain("NAlert");
    expect(source).toContain("NSpace");
    expect(source).toContain("useAuthStore");
    expect(source).toContain("canManageAudit");
    expect(source).toContain('v-if="canManageAudit"');
    expect(source).toContain('href="/api/system/audit-events/export?limit=100"');
    expect(source).toContain("systemApi.listAuditEventPage(100)");
    expect(source).toContain("systemApi.listAuditEventPage(100, nextCursor.value)");
    expect(source).toContain("systemApi.verifyAuditEventHashChain()");
    expect(source).toContain("hashVerificationType");
    expect(source).toContain("outcomeTagType(event.outcome)");
    expect(source).toContain('outcome === "warning"');
    expect(source).toContain('t("system.outcomeWarning")');
    expect(source).toContain('t("system.verifyAuditHashChain")');
    expect(source).toContain('t("system.auditHashVerification")');
    expect(source).toContain('v-if="nextCursor"');
    expect(source).toContain('t("system.auditHash")');
    expect(source).toContain("shortHash(event.eventHash)");
    expect(source).toContain("shortHash(event.previousHash)");
    expect(source).toContain("function shortHash");
    expect(source).toContain('t("system.loadOlderAuditEvents")');
    expect(source).toContain('t("system.exportAuditEvents")');
    expect(source).toContain("@click=\"loadEvents\"");
    expect(source).toContain("@click=\"loadOlderEvents\"");
    expect(zhMessages).toContain("\"system.exportAuditEvents\"");
    expect(zhMessages).toContain("\"system.auditHash\"");
    expect(zhMessages).toContain("\"system.auditHashVerification\"");
    expect(zhMessages).toContain("\"system.auditHashVerificationFailed\"");
    expect(zhMessages).toContain("\"system.verifyAuditHashChain\"");
    expect(zhMessages).toContain("\"system.previousHashShort\"");
    expect(zhMessages).toContain("\"system.loadOlderAuditEvents\"");
    expect(zhMessages).toContain("\"system.auditEventsLoadMoreFailed\"");
    expect(zhMessages).toContain("\"system.outcomeWarning\"");
    expect(enMessages).toContain("\"system.exportAuditEvents\"");
    expect(enMessages).toContain("\"system.auditHash\"");
    expect(enMessages).toContain("\"system.auditHashVerification\"");
    expect(enMessages).toContain("\"system.auditHashVerificationFailed\"");
    expect(enMessages).toContain("\"system.verifyAuditHashChain\"");
    expect(enMessages).toContain("\"system.previousHashShort\"");
    expect(enMessages).toContain("\"system.loadOlderAuditEvents\"");
    expect(enMessages).toContain("\"system.auditEventsLoadMoreFailed\"");
    expect(enMessages).toContain("\"system.outcomeWarning\"");
  });
});
