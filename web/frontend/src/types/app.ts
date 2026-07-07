import type {
  AuditEvent as APIAuditEvent,
  BacktestOrder as APIBacktestOrder,
  BacktestTask as APIBacktestTask,
  CandleCoverage as APICandleCoverage,
  CandleGap as APICandleGap,
  CandleHealth as APICandleHealth,
  CandleIssue as APICandleIssue,
  CandlePagination as APICandlePagination,
  CandleWindow as APICandleWindow,
  CandleSource as APICandleSource,
  CreateBacktestTask as APICreateBacktestTask,
  CreateDataSyncTask as APICreateDataSyncTask,
  CreateExchangeAccount as APICreateExchangeAccount,
  CreateNotificationChannel as APICreateNotificationChannel,
  CreateOperator as APICreateOperator,
  CreateTradingTask as APICreateTradingTask,
  DataSyncGapRepairResult as APIDataSyncGapRepairResult,
  DataSyncGapList as APIDataSyncGapList,
  DataSyncInvalidIssueList as APIDataSyncInvalidIssueList,
  DataSyncTask as APIDataSyncTask,
  ExchangeAccount as APIExchangeAccount,
  Execution as APIExecution,
  LoginRequest as APILoginRequest,
  MarketCandleGapScan as APIMarketCandleGapScan,
  MarketCandleInvalidIssueScan as APIMarketCandleInvalidIssueScan,
  MarketCandleQuarantineResult as APIMarketCandleQuarantineResult,
  MarketInstrument as APIMarketInstrument,
  MarketInstrumentSyncStatus as APIMarketInstrumentSyncStatus,
  Notification as APINotification,
  NotificationChannel as APINotificationChannel,
  Operator as APIOperator,
  OperatorSession as APIOperatorSession,
  Order as APIOrder,
  OverviewOrderFact as APIOverviewOrderFact,
  OverviewRecentFacts as APIOverviewRecentFacts,
  OverviewStrategyIntentFact as APIOverviewStrategyIntentFact,
  OverviewTrendBucket as APIOverviewTrendBucket,
  OverviewTrends as APIOverviewTrends,
  Position as APIPosition,
  RepairDataSyncInvalidIssuesRequest as APIRepairDataSyncInvalidIssuesRequest,
  RepairDataSyncTaskGapRequest as APIRepairDataSyncTaskGapRequest,
  RepairMarketCandleGapRequest as APIRepairMarketCandleGapRequest,
  RepairMarketCandleGapsRequest as APIRepairMarketCandleGapsRequest,
  RepairMarketCandleInvalidIssuesRequest as APIRepairMarketCandleInvalidIssuesRequest,
  QuarantineMarketCandleInvalidIssuesRequest as APIQuarantineMarketCandleInvalidIssuesRequest,
  ServiceHealth as APIServiceHealth,
  StrategyDefinition as APIStrategyDefinition,
  StrategyIntent as APIStrategyIntent,
  StrategyOption as APIStrategyOption,
  StrategyParamSpec as APIStrategyParamSpec,
  SystemHealth as APISystemHealth,
  TaskStatus as APITaskStatus,
  TradingTask as APITradingTask,
} from "@/types/api.generated";

export type ThemeMode = "light" | "dark";

export type LocaleCode = "zh-CN" | "en-US";

export type TaskStatus = APITaskStatus | "gap";

export type DataSyncTask = APIDataSyncTask;

export type DataSyncGapList = APIDataSyncGapList;

export type DataSyncInvalidIssueList = APIDataSyncInvalidIssueList;

export type DataSyncGapRepairResult = APIDataSyncGapRepairResult;

export type RepairDataSyncInvalidIssuesRequest = APIRepairDataSyncInvalidIssuesRequest;

export type RepairDataSyncTaskGapRequest = APIRepairDataSyncTaskGapRequest;

export type RepairMarketCandleGapRequest = APIRepairMarketCandleGapRequest;

export type RepairMarketCandleGapsRequest = APIRepairMarketCandleGapsRequest;

export type RepairMarketCandleInvalidIssuesRequest = APIRepairMarketCandleInvalidIssuesRequest;

export type QuarantineMarketCandleInvalidIssuesRequest = APIQuarantineMarketCandleInvalidIssuesRequest;

export type MarketCandleQuarantineResult = APIMarketCandleQuarantineResult;

export type ChartCandle = {
  time: number;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
};

export type ChartMarker = {
  id?: string;
  time: number;
  position: "aboveBar" | "belowBar" | "inBar";
  shape: "circle" | "square" | "arrowUp" | "arrowDown";
  color: string;
  text?: string;
  size?: number;
};

export type CandleSource = APICandleSource;

export type CandleHealth = APICandleHealth;

export type CandleGap = APICandleGap;

export type CandleIssue = APICandleIssue;

export type CandleCoverage = APICandleCoverage;

export type CandlePagination = APICandlePagination;

export type CandleWindow = APICandleWindow;

export type CandleResult = {
  candles: ChartCandle[];
  source: CandleSource;
  requestedInterval: string;
  baseInterval?: string;
  health: CandleHealth;
  gaps: CandleGap[];
  issues: CandleIssue[];
  coverage: CandleCoverage;
  window: CandleWindow;
  pagination: CandlePagination;
};

export type CreateDataSyncTask = APICreateDataSyncTask;

export type BacktestTriggerMode = "closed_candle" | "minute_replay";

export type BacktestTask = APIBacktestTask;

export type CreateBacktestTask = Omit<APICreateBacktestTask, "strategyParams" | "triggerMode"> & {
  strategyParams: StrategyParamValues;
  triggerMode: BacktestTriggerMode;
};

export type BacktestOrder = APIBacktestOrder;

export type StrategyIntent = APIStrategyIntent;

export type TradingTaskType = "paper" | "live";

export type TradingTask = Omit<APITradingTask, "type"> & {
  type: TradingTaskType;
};

export type CreateTradingTask = Omit<APICreateTradingTask, "strategyParams" | "type"> & {
  type: TradingTaskType;
  strategyParams: StrategyParamValues;
};

export type Order = APIOrder;

export type OverviewOrderFact = APIOverviewOrderFact;

export type OverviewRecentFacts = APIOverviewRecentFacts;

export type OverviewStrategyIntentFact = APIOverviewStrategyIntentFact;

export type OverviewTrendBucket = APIOverviewTrendBucket;

export type OverviewTrends = APIOverviewTrends;

export type Execution = APIExecution;

export type Position = APIPosition;

export type Notification = APINotification;

export type NotificationChannel = APINotificationChannel;

export type CreateNotificationChannel = APICreateNotificationChannel;

export type ExchangeAccount = APIExchangeAccount;

export type CreateExchangeAccount = APICreateExchangeAccount;

export type MarketInstrument = APIMarketInstrument;

export type MarketCandleGapScan = APIMarketCandleGapScan;

export type MarketCandleInvalidIssueScan = APIMarketCandleInvalidIssueScan;

export type MarketInstrumentSyncStatus = APIMarketInstrumentSyncStatus;

export type MarketInstrumentSyncResult = {
  exchange: string;
  activeCount: number;
  inactiveCount: number;
  pausedDataSyncTaskCount: number;
  restoredDataSyncTaskCount: number;
  syncedAt: string;
};

export type Operator = APIOperator;

export type CreateOperator = APICreateOperator;

export type LoginCredentials = APILoginRequest;

export type OperatorSession = APIOperatorSession;

export type AuditEvent = APIAuditEvent;

export type ServiceHealth = APIServiceHealth;

export type SystemHealth = APISystemHealth;

export type StrategyParamValue = string | number | boolean | null;

export type StrategyParamValues = Record<string, StrategyParamValue>;

export type StrategyOption = APIStrategyOption;

export type StrategyParamType = "number" | "select" | "text" | "boolean";

export type StrategyParamSpec = Omit<APIStrategyParamSpec, "default" | "options" | "type"> & {
  type: StrategyParamType;
  default?: StrategyParamValue;
  options: StrategyOption[];
};

export type StrategyDefinition = Omit<APIStrategyDefinition, "params"> & {
  params: StrategyParamSpec[];
};
