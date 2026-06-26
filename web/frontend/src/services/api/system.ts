import { apiClient } from "@/services/api/client";
import type {
  CreateExchangeAccount,
  CreateNotificationChannel,
  CreateOperator,
  ExchangeAccount,
  NotificationChannel,
  Operator,
  SystemHealth,
} from "@/types/app";

export const systemApi = {
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

  health() {
    return apiClient.get<SystemHealth>("/system/health");
  },
};
