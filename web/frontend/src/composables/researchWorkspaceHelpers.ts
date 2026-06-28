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

export function toISOString(value: number | null) {
  return value === null ? undefined : new Date(value).toISOString();
}

export function errorMessage(error: unknown, fallback: string) {
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}
