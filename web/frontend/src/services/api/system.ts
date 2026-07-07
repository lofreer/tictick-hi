import { apiClient } from "@/services/api/client";
import type {
  AuditEvent,
  AuditEventHashChainVerification,
  AuditEventPage,
  CreateExchangeAccount,
  CreateNotificationChannel,
  CreateOperator,
  ExchangeAccount,
  Notification,
  NotificationChannel,
  Operator,
  OperatorSessionRevokeResult,
  SystemHealth,
  UpdateOperatorRole,
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

  updateNotificationChannel(id: string, request: CreateNotificationChannel) {
    return apiClient.put<NotificationChannel>(`/system/notifications/channels/${encodeURIComponent(id)}`, request);
  },

  async deleteNotificationChannel(id: string) {
    await apiClient.delete<null>(`/system/notifications/channels/${encodeURIComponent(id)}`);
  },

  setNotificationChannelEnabled(id: string, enabled: boolean) {
    const action = enabled ? "enable" : "disable";
    return apiClient.post<NotificationChannel>(`/system/notifications/channels/${encodeURIComponent(id)}/${action}`);
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

  setOperatorRole(id: string, request: UpdateOperatorRole) {
    return apiClient.post<Operator>(`/system/operators/${encodeURIComponent(id)}/role`, request);
  },

  revokeOperatorSessions(id: string) {
    return apiClient.post<OperatorSessionRevokeResult>(`/system/operators/${encodeURIComponent(id)}/sessions/revoke`);
  },

  health() {
    return apiClient.get<SystemHealth>("/system/health");
  },

  listAuditEvents(limit = 100) {
    return apiClient.get<AuditEvent[]>(`/system/audit-events?limit=${encodeURIComponent(String(limit))}`);
  },

  listAuditEventPage(limit = 100, cursor = "") {
    const params = new URLSearchParams({ limit: String(limit) });
    if (cursor !== "") {
      params.set("cursor", cursor);
    }
    return apiClient.get<AuditEventPage>(`/system/audit-events/page?${params.toString()}`);
  },

  verifyAuditEventHashChain() {
    return apiClient.get<AuditEventHashChainVerification>("/system/audit-events/hash-chain/verify");
  },
};
