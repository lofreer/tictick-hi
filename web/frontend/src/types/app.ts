export type ThemeMode = "light" | "dark";

export type LocaleCode = "zh-CN" | "en-US";

export type TaskStatus =
  | "pending"
  | "running"
  | "stopping"
  | "paused"
  | "succeeded"
  | "failed"
  | "cancelled"
  | "gap";

export type DataSyncTask = {
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

export type ChartCandle = {
  time: number;
  open: number;
  high: number;
  low: number;
  close: number;
};

export type CreateDataSyncTask = {
  exchange: string;
  symbol: string;
  interval: string;
  startTime?: string;
  endTime?: string;
};

export type StrategyParamValue = string | number | boolean | null;

export type StrategyParamValues = Record<string, StrategyParamValue>;

export type StrategyOption = {
  label: string;
  value: string;
};

export type StrategyParamType = "number" | "select" | "text" | "boolean";

export type StrategyParamSpec = {
  key: string;
  label: string;
  type: StrategyParamType;
  required: boolean;
  default?: StrategyParamValue;
  min?: number;
  max?: number;
  step?: number;
  options: StrategyOption[];
  description?: string;
};

export type StrategyDefinition = {
  id: string;
  name: string;
  version: string;
  description: string;
  supportedIntervals: string[];
  supportedIntents: string[];
  params: StrategyParamSpec[];
};
