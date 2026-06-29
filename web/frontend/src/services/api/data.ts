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
  DataSyncGapList,
  DataSyncGapRepairResult,
  DataSyncInvalidIssueList,
  DataSyncTask,
  MarketCandleGapScan,
  RepairDataSyncInvalidIssuesRequest,
  RepairDataSyncTaskGapRequest,
  RepairMarketCandleGapRequest,
  RepairMarketCandleGapsRequest,
} from "@/types/app";
import { sanitizeExternalError } from "@/utils/errorText";

export type CandleQuery = {
  cursor?: string;
  exchange: string;
  symbol: string;
  interval: string;
  from?: string;
  limit?: number;
  to?: string;
};

export type DataSyncInvalidIssueQuery = {
  code?: string;
  from?: string;
  limit?: number;
  offset?: number;
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

  async getTaskGaps(id: string): Promise<DataSyncGapList> {
    return apiClient.get<DataSyncGapList>(`/data/tasks/${id}/gaps`);
  },

  async getTaskInvalidIssues(id: string, query: DataSyncInvalidIssueQuery = {}): Promise<DataSyncInvalidIssueList> {
    const params = new URLSearchParams();
    if (query.limit !== undefined) {
      params.set("limit", String(query.limit));
    }
    if (query.offset !== undefined) {
      params.set("offset", String(query.offset));
    }
    if (query.code) {
      params.set("code", query.code);
    }
    if (query.from) {
      params.set("from", query.from);
    }
    if (query.to) {
      params.set("to", query.to);
    }
    return apiClient.get<DataSyncInvalidIssueList>(`/data/tasks/${id}/invalid-issues?${params.toString()}`);
  },

  async repairTaskGaps(id: string): Promise<DataSyncGapRepairResult> {
    const response = await apiClient.post<DataSyncGapRepairResult>(`/data/tasks/${id}/repair-gaps`);
    return normalizeGapRepairResult(response);
  },

  async repairTaskInvalidIssues(id: string, request: RepairDataSyncInvalidIssuesRequest): Promise<DataSyncGapRepairResult> {
    const response = await apiClient.post<DataSyncGapRepairResult>(`/data/tasks/${id}/repair-invalid-issues`, request);
    return normalizeGapRepairResult(response);
  },

  async repairTaskGap(id: string, request: RepairDataSyncTaskGapRequest): Promise<DataSyncGapRepairResult> {
    const response = await apiClient.post<DataSyncGapRepairResult>(`/data/tasks/${id}/repair-gap`, request);
    return normalizeGapRepairResult(response);
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
    });
    if (query.cursor) {
      params.set("cursor", query.cursor);
    } else {
      params.set("limit", String(query.limit ?? 1000));
      if (query.from) {
        params.set("from", query.from);
      }
      if (query.to) {
        params.set("to", query.to);
      }
    }
    const response = await apiClient.get<CandleResultResponse>(`/candles?${params.toString()}`);
    return normalizeCandleResult(response, query.interval);
  },

  async scanMarketCandleGaps(query: CandleQuery): Promise<MarketCandleGapScan> {
    const params = new URLSearchParams({
      exchange: query.exchange,
      symbol: query.symbol,
      interval: query.interval,
      limit: String(query.limit ?? 20),
    });
    return apiClient.get<MarketCandleGapScan>(`/market/candle-gaps?${params.toString()}`);
  },

  async repairMarketCandleGap(request: RepairMarketCandleGapRequest): Promise<DataSyncGapRepairResult> {
    const response = await apiClient.post<DataSyncGapRepairResult>("/market/candle-gaps/repair", request);
    return normalizeGapRepairResult(response);
  },

  async repairMarketCandleGaps(request: RepairMarketCandleGapsRequest): Promise<DataSyncGapRepairResult> {
    const response = await apiClient.post<DataSyncGapRepairResult>("/market/candle-gaps/repair-batch", request);
    return normalizeGapRepairResult(response);
  },

  async listCandles(query: CandleQuery) {
    const result = await dataApi.getCandles(query);
    return result.candles;
  },
};

function normalizeGapRepairResult(response: DataSyncGapRepairResult): DataSyncGapRepairResult {
  return {
    ...response,
    createdTasks: response.createdTasks.map(normalizeTask),
  };
}

function normalizeTask(response: DataSyncTaskResponse): DataSyncTask {
  return {
    id: response.id,
    exchange: response.exchange,
    symbol: response.symbol,
    interval: response.interval,
    startTime: response.startTime,
    endTime: response.endTime,
    repairSourceTaskId: response.repairSourceTaskId,
    latestSyncedAt: response.latestSyncedAt,
    realtimeEnabled: response.realtimeEnabled,
    syncEnabled: response.syncEnabled,
    status: response.status,
    marketStatus: response.marketStatus ?? "missing",
    marketStatusDetail: response.marketStatusDetail ?? response.marketStatus ?? "missing",
    dataHealth: response.dataHealth,
    gapSummary: response.gapSummary,
    invalidSummary: response.invalidSummary,
    lastError: sanitizeExternalError(response.lastError),
    attemptCount: response.attemptCount,
    nextAttemptAt: response.nextAttemptAt,
    exchangeBackoffUntil: response.exchangeBackoffUntil,
    exchangeBackoffLastError: sanitizeExternalError(response.exchangeBackoffLastError),
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
  const volume = Number(response.volume);

  if (![time, open, high, low, close, volume].every(Number.isFinite)) {
    return null;
  }

  return {
    time: Math.floor(time / 1000),
    open,
    high,
    low,
    close,
    volume,
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
    issues: response.issues ?? [],
    coverage: response.coverage ?? {
      requestedLimit: 1000,
      returnedCandles: candles.length,
      limitedByBaseWindow: false,
    },
    window: response.window ?? {
      count: candles.length,
    },
    pagination: response.pagination ?? {
      hasPrevious: false,
      hasNext: false,
    },
  };
}
