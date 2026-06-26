import { apiClient } from "@/services/api/client";
import type { ChartCandle, CreateDataSyncTask, DataSyncTask, TaskStatus } from "@/types/app";

type DataSyncTaskResponse = {
  id: string;
  exchange: string;
  symbol: string;
  interval: string;
  startTime?: string;
  endTime?: string;
  latestSyncedAt?: string;
  realtimeEnabled: boolean;
  syncEnabled: boolean;
  status: TaskStatus;
  lastError?: string;
  createdAt?: string;
  updatedAt?: string;
};

type CandleResponse = {
  openTime: string;
  open: string;
  high: string;
  low: string;
  close: string;
};

export type CandleQuery = {
  exchange: string;
  symbol: string;
  interval: string;
  from?: string;
  limit?: number;
  to?: string;
};

export const dataApi = {
  async listTasks() {
    const response = await apiClient.get<DataSyncTaskResponse[]>("/data/tasks");
    return response.map(normalizeTask);
  },

  async createTask(request: CreateDataSyncTask) {
    const response = await apiClient.post<DataSyncTaskResponse>("/data/tasks", request);
    return normalizeTask(response);
  },

  async deleteTask(id: string) {
    await apiClient.delete<null>(`/data/tasks/${id}`);
  },

  async setSync(id: string, enabled: boolean) {
    const action = enabled ? "start" : "stop";
    const response = await apiClient.post<DataSyncTaskResponse>(`/data/tasks/${id}/sync/${action}`);
    return normalizeTask(response);
  },

  async setRealtime(id: string, enabled: boolean) {
    const action = enabled ? "start" : "stop";
    const response = await apiClient.post<DataSyncTaskResponse>(`/data/tasks/${id}/realtime/${action}`);
    return normalizeTask(response);
  },

  async listCandles(query: CandleQuery) {
    const params = new URLSearchParams({
      exchange: query.exchange,
      symbol: query.symbol,
      interval: query.interval,
      limit: String(query.limit ?? 1000),
    });
    if (query.from) {
      params.set("from", query.from);
    }
    if (query.to) {
      params.set("to", query.to);
    }
    const response = await apiClient.get<CandleResponse[]>(`/candles?${params.toString()}`);
    return response.map(toChartCandle).filter((item): item is ChartCandle => item !== null);
  },
};

function normalizeTask(response: DataSyncTaskResponse): DataSyncTask {
  return {
    id: response.id,
    exchange: response.exchange,
    symbol: response.symbol,
    interval: response.interval,
    startTime: response.startTime,
    endTime: response.endTime,
    latestSyncedAt: response.latestSyncedAt,
    realtimeEnabled: response.realtimeEnabled,
    syncEnabled: response.syncEnabled,
    status: response.status,
    lastError: response.lastError,
    createdAt: response.createdAt,
    updatedAt: response.updatedAt,
  };
}

function toChartCandle(response: CandleResponse): ChartCandle | null {
  const time = Date.parse(response.openTime);
  const open = Number(response.open);
  const high = Number(response.high);
  const low = Number(response.low);
  const close = Number(response.close);

  if (![time, open, high, low, close].every(Number.isFinite)) {
    return null;
  }

  return {
    time: Math.floor(time / 1000),
    open,
    high,
    low,
    close,
  };
}
