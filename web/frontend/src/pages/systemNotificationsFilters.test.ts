import { describe, expect, it } from "vitest";

import {
  notificationMatchesStatusFilter,
  notificationStatusFilterFromQuery,
  notificationStatusQueryValue,
} from "@/pages/systemNotificationsFilters";
import type { Notification } from "@/types/app";

describe("system notification filters", () => {
  it("normalizes status query values", () => {
    expect(notificationStatusFilterFromQuery("failed")).toBe("failed");
    expect(notificationStatusFilterFromQuery("pending")).toBe("pending");
    expect(notificationStatusFilterFromQuery("sent")).toBe("sent");
    expect(notificationStatusFilterFromQuery("retry_scheduled")).toBe("all");
    expect(notificationStatusFilterFromQuery(["failed"])).toBe("all");
    expect(notificationStatusQueryValue("all")).toBeUndefined();
    expect(notificationStatusQueryValue("failed")).toBe("failed");
  });

  it("matches grouped notification status filters", () => {
    expect(notificationMatchesStatusFilter(notification("failed"), "failed")).toBe(true);
    expect(notificationMatchesStatusFilter(notification("retry_scheduled"), "pending")).toBe(true);
    expect(notificationMatchesStatusFilter(notification("pending"), "pending")).toBe(true);
    expect(notificationMatchesStatusFilter(notification("delivered"), "sent")).toBe(true);
    expect(notificationMatchesStatusFilter(notification("sent"), "sent")).toBe(true);
    expect(notificationMatchesStatusFilter(notification("sent"), "failed")).toBe(false);
    expect(notificationMatchesStatusFilter(notification("anything"), "all")).toBe(true);
  });
});

function notification(status: string): Notification {
  return {
    attemptCount: 1,
    body: "body",
    channel: "ops",
    createdAt: "2026-07-07T08:00:00Z",
    id: `nt_${status}`,
    maxAttempts: 3,
    provider: "local",
    status,
    target: "default",
    title: status,
  };
}
