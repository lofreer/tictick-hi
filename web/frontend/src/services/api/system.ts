import { apiClient } from "@/services/api/client";
import type {
  AuditEvent,
  CreateExchangeAccount,
  CreateNotificationChannel,
  CreateOperator,
  ExchangeAccount,
  Notification,
  NotificationChannel,
  Operator,
  SystemHealth,
} from "@/types/app";

export const systemApi = {
  listNotifications() {
    return apiClient.get<Notification[]>("/system/notifications");
  },

  retryNotification(id: string) {
    return apiClient.post<Notification>(`/system/notifications/${encodeURIComponent(id)}/retry`);
  },

  listNotificationChannels() {
    return apiClient.get<NotificationChannel[]>("/system/notifications/channels");
  },

  createNotificationChannel(request: CreateNotificationChannel) {
    return apiClient.post<NotificationChannel>("/system/notifications/channels", request);
  },

  listExchangeAccounts() {
    return apiClient.get<ExchangeAccount[]>("/system/exchange-accounts");
  },

  createExchangeAccount(request: CreateExchangeAccount) {
    return apiClient.post<ExchangeAccount>("/system/exchange-accounts", request);
  },

  listOperators() {
    return apiClient.get<Operator[]>("/system/operators");
  },

  createOperator(request: CreateOperator) {
    return apiClient.post<Operator>("/system/operators", request);
  },

  setOperatorEnabled(id: string, enabled: boolean) {
    const action = enabled ? "enable" : "disable";
    return apiClient.post<Operator>(`/system/operators/${encodeURIComponent(id)}/${action}`);
  },

  health() {
    return apiClient.get<SystemHealth>("/system/health");
  },

  listAuditEvents(limit = 100) {
    return apiClient.get<AuditEvent[]>(`/system/audit-events?limit=${encodeURIComponent(String(limit))}`);
  },
};
