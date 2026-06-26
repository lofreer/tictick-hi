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

export type BacktestTriggerMode = "closed_candle" | "minute_replay";

export type BacktestTask = {
  id: string;
  name: string;
  exchange: string;
  symbol: string;
  interval: string;
  startTime?: string;
  endTime?: string;
  strategyId: string;
  strategyParams: Record<string, unknown>;
  initialBalance: string;
  feeBps: string;
  slippageBps: string;
  triggerMode: BacktestTriggerMode;
  status: TaskStatus;
  startedAt?: string;
  finishedAt?: string;
  lastError?: string;
  attemptCount: number;
  resultSummary: Record<string, unknown>;
  createdAt?: string;
  updatedAt?: string;
};

export type CreateBacktestTask = {
  name: string;
  exchange: string;
  symbol: string;
  interval: string;
  startTime?: string;
  endTime?: string;
  strategyId: string;
  strategyParams: StrategyParamValues;
  initialBalance: string;
  feeBps: string;
  slippageBps: string;
  triggerMode: BacktestTriggerMode;
};

export type BacktestOrder = {
  id: string;
  backtestId: string;
  intentId?: string;
  side: string;
  price: string;
  quantity: string;
  status: string;
  occurredAt: string;
};

export type TradingTaskType = "paper" | "live";

export type TradingTask = {
  id: string;
  name: string;
  type: TradingTaskType;
  exchange: string;
  accountId: string;
  symbol: string;
  interval: string;
  strategyId: string;
  strategyParams: Record<string, unknown>;
  intentPolicy: Record<string, unknown>;
  status: TaskStatus;
  startedAt?: string;
  finishedAt?: string;
  lastError?: string;
  attemptCount: number;
  createdAt?: string;
  updatedAt?: string;
};

export type CreateTradingTask = {
  name: string;
  type: TradingTaskType;
  exchange: string;
  accountId: string;
  symbol: string;
  interval: string;
  strategyId: string;
  strategyParams: StrategyParamValues;
  intentPolicy: Record<string, unknown>;
};

export type StrategyIntent = {
  id: string;
  taskId: string;
  taskType: string;
  strategyId: string;
  intentType: string;
  idempotencyKey: string;
  payload: Record<string, unknown>;
  policy: string;
  status: string;
  createdAt: string;
};

export type Order = {
  id: string;
  taskId: string;
  taskType: string;
  intentId?: string;
  idempotencyKey: string;
  exchange: string;
  accountId: string;
  symbol: string;
  side: string;
  orderType: string;
  price: string;
  quantity: string;
  status: string;
  exchangeOrderId?: string;
  exchangeResponseSummary: Record<string, unknown>;
  lastError?: string;
  createdAt: string;
  updatedAt: string;
};

export type Notification = {
  id: string;
  intentId?: string;
  channel: string;
  title: string;
  body: string;
  status: string;
  error?: string;
  createdAt: string;
  sentAt?: string;
};

export type NotificationChannel = {
  id: string;
  name: string;
  provider: string;
  target: string;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
};

export type CreateNotificationChannel = {
  name: string;
  provider: string;
  target: string;
  enabled: boolean;
};

export type ExchangeAccount = {
  id: string;
  exchange: string;
  alias: string;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
};

export type CreateExchangeAccount = {
  exchange: string;
  alias: string;
  apiKey: string;
  apiSecret: string;
  enabled: boolean;
};

export type Operator = {
  id: string;
  username: string;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
};

export type CreateOperator = {
  username: string;
  password: string;
  enabled: boolean;
};

export type ServiceHealth = {
  name: string;
  status: string;
  detail?: string;
};

export type SystemHealth = {
  status: string;
  database: string;
  checkedAt: string;
  services: ServiceHealth[];
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
