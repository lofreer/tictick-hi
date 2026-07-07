/// <reference types="node" />
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

import source from "./SystemNotificationsPage.vue?raw";

const zhMessages = readFileSync("src/i18n/messages.zh.ts", "utf8");
const enMessages = readFileSync("src/i18n/messages.en.ts", "utf8");

describe("SystemNotificationsPage layout contract", () => {
  it("filters notifications by status query context", () => {
    expect(source).toContain("useRoute");
    expect(source).toContain("useRouter");
    expect(source).toContain("notificationStatusFilterFromQuery(route.query.status)");
    expect(source).toContain("NRadioGroup");
    expect(source).toContain(":value=\"notificationStatusFilter\"");
    expect(source).toContain(":aria-label=\"t('system.notificationStatusFilter')\"");
    expect(source).toContain("@update:value=\"setNotificationStatusFilter\"");
    expect(source).toContain("filteredNotifications");
    expect(source).toContain("notificationMatchesStatusFilter");
    expect(source).toContain("notificationStatusQueryValue");
    expect(source).toContain("notification.providerMessageId || \"-\"");
    for (const messages of [zhMessages, enMessages]) {
      expect(messages).toContain("\"system.notificationStatusFilter\"");
      expect(messages).toContain("\"system.notificationStatus.all\"");
      expect(messages).toContain("\"system.notificationStatus.failed\"");
      expect(messages).toContain("\"system.notificationStatus.pending\"");
      expect(messages).toContain("\"system.notificationStatus.sent\"");
      expect(messages).toContain("\"system.noNotificationsForFilter\"");
      expect(messages).toContain("\"system.providerMessageId\"");
    }
  });
});
