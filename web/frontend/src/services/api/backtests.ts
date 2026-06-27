import { apiClient } from "@/services/api/client";
import type { BacktestOrder, BacktestTask, CreateBacktestTask, StrategyIntent } from "@/types/app";

export const backtestsApi = {
  listBacktests() {
    return apiClient.get<BacktestTask[]>("/backtests");
  },

  createBacktest(request: CreateBacktestTask) {
    return apiClient.post<BacktestTask>("/backtests", request);
  },

  getBacktest(id: string) {
    return apiClient.get<BacktestTask>(`/backtests/${encodeURIComponent(id)}`);
  },

  listOrders(id: string) {
    return apiClient.get<BacktestOrder[]>(`/backtests/${encodeURIComponent(id)}/orders`);
  },

  listIntents(id: string) {
    return apiClient.get<StrategyIntent[]>(`/backtests/${encodeURIComponent(id)}/intents`);
  },
};
