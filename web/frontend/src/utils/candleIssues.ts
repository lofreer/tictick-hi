export function isRepairableCandleIssueCode(code?: string): boolean {
  return code !== "invalid_open_time";
}

export function isQuarantinableCandleIssueCode(code?: string): boolean {
  return code === "invalid_open_time";
}
