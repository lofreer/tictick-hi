import { dataApi } from "@/services/api/data";
import type { CandleGap, DataSyncGapRepairResult, DataSyncTask } from "@/types/app";

import {
  chartGapRepairRequest,
  marketGapRepairRequest,
  repairResultMessageKey,
} from "./researchWorkspaceHelpers";

type RepairChartGapOptions = {
  exchange: string;
  gap: CandleGap;
  loadCandles: () => Promise<void>;
  loadTasks: () => Promise<unknown>;
  onSuccess: (messageKey: string) => void;
  repairInterval: string;
  sourceTask: DataSyncTask | null;
  symbol: string;
};

export async function repairChartGap(options: RepairChartGapOptions) {
  const { exchange, gap, loadCandles, loadTasks, onSuccess, repairInterval, sourceTask, symbol } = options;
  let result: DataSyncGapRepairResult;
  if (sourceTask) {
    result = await dataApi.repairTaskGap(sourceTask.id, chartGapRepairRequest(gap));
  } else {
    result = await dataApi.repairMarketCandleGap(marketGapRepairRequest(gap, exchange, symbol, repairInterval));
  }
  onSuccess(repairResultMessageKey(result));
  await Promise.all([loadTasks(), loadCandles()]);
  return result;
}
