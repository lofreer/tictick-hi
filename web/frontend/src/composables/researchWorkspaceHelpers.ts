import type {
  CandleGap,
  CandleResult,
  CreateDataSyncTask,
  DataSyncGapRepairResult,
  DataSyncTask,
  RepairDataSyncTaskGapRequest,
} from "@/types/app";

export type ResearchForm = {
  exchange: string;
  symbol: string;
  interval: string;
  startTime: number | null;
  endTime: number | null;
};

export function readQuery(value: unknown, fallback: string) {
  return typeof value === "string" && value.length > 0 ? value : fallback;
}

export function readOptionalQuery(value: unknown) {
  return typeof value === "string" && value.length > 0 ? value : "";
}

export function researchQuery(exchange: string, symbol: string, interval: string, from: string, to: string) {
  const query: Record<string, string> = { exchange, symbol, interval };
  if (from) query.from = from;
  if (to) query.to = to;
  return query;
}

export function candleQuery(exchange: string, symbol: string, interval: string, from: string, to: string) {
  const query: { exchange: string; symbol: string; interval: string; from?: string; to?: string } = {
    exchange,
    symbol,
    interval,
  };
  if (from) query.from = from;
  if (to) query.to = to;
  return query;
}

export function canLoadPreviousCandleWindow(result: CandleResult | null) {
  return Boolean(result?.pagination.hasPrevious && result.pagination.previousFrom && result.pagination.previousTo);
}

export function canLoadNextCandleWindow(result: CandleResult | null) {
  return Boolean(result?.pagination.hasNext && result.pagination.nextFrom && result.pagination.nextTo);
}

export function toISOString(value: number | null) {
  return value === null ? undefined : new Date(value).toISOString();
}

export function errorMessage(error: unknown, fallback: string) {
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}

export function selectedTaskMatchesMarket(task: DataSyncTask, exchange: string, symbol: string) {
  return task.exchange === exchange && task.symbol === symbol;
}

export function repairSourceTask(
  task: DataSyncTask | null,
  exchange: string,
  symbol: string,
  repairInterval: string,
) {
  if (!task || !selectedTaskMatchesMarket(task, exchange, symbol) || task.interval !== repairInterval) {
    return null;
  }
  return task;
}

export function chartGapRepairRequest(gap: CandleGap): RepairDataSyncTaskGapRequest {
  return {
    from: gap.from,
    to: gap.to,
  };
}

export function fallbackGapRepairTask(
  gap: CandleGap,
  exchange: string,
  symbol: string,
  interval: string,
): CreateDataSyncTask {
  return {
    exchange,
    symbol,
    interval,
    startTime: gap.from,
    endTime: gap.to,
  };
}

export function repairResultMessageKey(result: DataSyncGapRepairResult) {
  if (result.createdTasks.length > 0) return "research.gapRepairQueued";
  if (result.skippedExisting > 0) return "research.taskGapRepairAlreadyQueued";
  return "research.noRepairableTaskGaps";
}
