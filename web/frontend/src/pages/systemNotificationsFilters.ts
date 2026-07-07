import type { Notification } from "@/types/app";

export type NotificationStatusFilter = "all" | "failed" | "pending" | "sent";

const statusFilters: NotificationStatusFilter[] = ["all", "failed", "pending", "sent"];
const pendingStatuses = new Set(["pending", "retry_scheduled"]);
const sentStatuses = new Set(["sent", "delivered"]);

export function notificationStatusFilterFromQuery(value: unknown): NotificationStatusFilter {
  return typeof value === "string" && statusFilters.includes(value as NotificationStatusFilter) ? (value as NotificationStatusFilter) : "all";
}

export function notificationStatusQueryValue(value: NotificationStatusFilter) {
  return value === "all" ? undefined : value;
}

export function notificationMatchesStatusFilter(notification: Notification, filter: NotificationStatusFilter) {
  if (filter === "all") return true;
  if (filter === "failed") return notification.status === "failed";
  if (filter === "pending") return pendingStatuses.has(notification.status);
  return sentStatuses.has(notification.status);
}
