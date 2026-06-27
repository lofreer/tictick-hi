import { apiClient } from "@/services/api/client";
import type { CreateTradingTask, Execution, Notification, Order, Position, StrategyIntent, TradingTask } from "@/types/app";

export const tradingApi = {
  listTasks() {
    return apiClient.get<TradingTask[]>("/trading/tasks");
  },

  createTask(request: CreateTradingTask) {
    return apiClient.post<TradingTask>("/trading/tasks", request);
  },

  getTask(id: string) {
    return apiClient.get<TradingTask>(`/trading/tasks/${encodeURIComponent(id)}`);
  },

  startTask(id: string) {
    return apiClient.post<TradingTask>(`/trading/tasks/${encodeURIComponent(id)}/start`);
  },

  pauseTask(id: string) {
    return apiClient.post<TradingTask>(`/trading/tasks/${encodeURIComponent(id)}/pause`);
  },

  stopTask(id: string) {
    return apiClient.post<TradingTask>(`/trading/tasks/${encodeURIComponent(id)}/stop`);
  },

  listIntents(id: string) {
    return apiClient.get<StrategyIntent[]>(`/trading/tasks/${encodeURIComponent(id)}/intents`);
  },

  listOrders(id: string) {
    return apiClient.get<Order[]>(`/trading/tasks/${encodeURIComponent(id)}/orders`);
  },

  listExecutions(id: string) {
    return apiClient.get<Execution[]>(`/trading/tasks/${encodeURIComponent(id)}/executions`);
  },

  listPositions(id: string) {
    return apiClient.get<Position[]>(`/trading/tasks/${encodeURIComponent(id)}/positions`);
  },

  listNotifications(id: string) {
    return apiClient.get<Notification[]>(`/trading/tasks/${encodeURIComponent(id)}/notifications`);
  },
};
