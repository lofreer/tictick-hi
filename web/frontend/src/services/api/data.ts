import { apiClient } from "@/services/api/client";
import type {
  Candle as CandleResponse,
  CandleResult as CandleResultResponse,
  DataSyncTask as DataSyncTaskResponse,
} from "@/types/api.generated";
import type {
  CandleResult,
  ChartCandle,
  CreateDataSyncTask,
  DataSyncTask,
} from "@/types/app";

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

  async retryTask(id: string) {
    const response = await apiClient.post<DataSyncTaskResponse>(`/data/tasks/${id}/retry`);
    return normalizeTask(response);
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

  async getCandles(query: CandleQuery): Promise<CandleResult> {
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
    const response = await apiClient.get<CandleResultResponse>(`/candles?${params.toString()}`);
    return normalizeCandleResult(response, query.interval);
  },

  async listCandles(query: CandleQuery) {
    const result = await dataApi.getCandles(query);
    return result.candles;
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
    attemptCount: response.attemptCount,
    nextAttemptAt: response.nextAttemptAt,
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

function normalizeCandleResult(response: CandleResultResponse, requestedInterval: string): CandleResult {
  const candles = (response.candles ?? [])
    .map(toChartCandle)
    .filter((item): item is ChartCandle => item !== null);

  return {
    candles,
    source: response.source ?? "none",
    requestedInterval: response.requestedInterval ?? requestedInterval,
    baseInterval: response.baseInterval,
    health: response.health ?? (candles.length > 0 ? "ok" : "insufficient"),
    gaps: response.gaps ?? [],
    coverage: response.coverage ?? {
      requestedLimit: 1000,
      returnedCandles: candles.length,
      limitedByBaseWindow: false,
    },
  };
}
