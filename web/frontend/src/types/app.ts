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
