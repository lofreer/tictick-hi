import { dataApi } from "@/services/api/data";
import type { CandleIssue, DataSyncGapRepairResult } from "@/types/app";

type InvalidIssueRepairFeedback = {
  messageKey: string;
  values?: Record<string, number>;
};

type RepairChartInvalidIssueOptions = {
  exchange: string;
  interval: string;
  issue: CandleIssue;
  loadCandles: () => Promise<void>;
  loadTasks: () => Promise<unknown>;
  onSuccess: (messageKey: string, values?: Record<string, number>) => void;
  symbol: string;
};

export async function repairChartInvalidIssue(options: RepairChartInvalidIssueOptions) {
  const { exchange, interval, issue, loadCandles, loadTasks, onSuccess, symbol } = options;
  if (!issue.openTime) {
    throw new Error("invalid candle issue open time is required");
  }
  const result = await dataApi.repairMarketCandleInvalidIssues({
    exchange,
    symbol,
    interval,
    openTimes: [issue.openTime],
  });
  const feedback = invalidIssueRepairFeedback(result);
  onSuccess(feedback.messageKey, feedback.values);
  await Promise.all([loadTasks(), loadCandles()]);
  return result;
}

export function invalidIssueRepairFeedback(result: DataSyncGapRepairResult): InvalidIssueRepairFeedback {
  if (result.createdTasks.length > 0) {
    return {
      messageKey: "research.invalidIssueRepairQueued",
      values: { count: result.createdTasks.length },
    };
  }
  if (result.skippedExisting > 0) return { messageKey: "research.invalidIssueRepairAlreadyQueued" };
  return { messageKey: "research.noRepairableInvalidIssues" };
}
