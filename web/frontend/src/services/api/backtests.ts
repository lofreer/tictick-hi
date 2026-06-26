import { apiClient } from "@/services/api/client";
import type { BacktestOrder, BacktestTask, CreateBacktestTask } from "@/types/app";

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
};
