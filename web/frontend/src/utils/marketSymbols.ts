export type MarketSymbolOption = {
  label: string;
  value: string;
};

const fallbackExchange = "binance";
const binanceSymbolPattern = /^[A-Z0-9]{3,30}$/;
const okxSymbolPattern = /^[A-Z0-9]+-[A-Z0-9]+(?:-[A-Z0-9]+)?$/;

const marketSymbolOptions: Record<string, MarketSymbolOption[]> = {
  binance: [
    { label: "BTCUSDT", value: "BTCUSDT" },
    { label: "ETHUSDT", value: "ETHUSDT" },
  ],
  okx: [
    { label: "BTC-USDT", value: "BTC-USDT" },
    { label: "ETH-USDT", value: "ETH-USDT" },
  ],
};

export function symbolOptionsForExchange(exchange: string) {
  return [...optionsForExchange(exchange)];
}

export function defaultSymbolForExchange(exchange: string) {
  return optionsForExchange(exchange)[0]?.value ?? "";
}

export function isSymbolForExchange(exchange: string, symbol: string) {
  return isSymbolFormatForExchange(exchange, symbol);
}

export function isSymbolFormatForExchange(exchange: string, symbol: string) {
  const normalized = normalizeSymbolInput(symbol);
  if (exchange === "okx") {
    return okxSymbolPattern.test(normalized);
  }
  return binanceSymbolPattern.test(normalized);
}

export function coerceSymbolForExchange(exchange: string, symbol: string) {
  const normalized = normalizeSymbolInput(symbol);
  return isSymbolFormatForExchange(exchange, normalized) ? normalized : defaultSymbolForExchange(exchange);
}

export function normalizeSymbolInput(symbol: string) {
  return symbol.trim().toUpperCase();
}

function optionsForExchange(exchange: string) {
  return marketSymbolOptions[exchange] ?? marketSymbolOptions[fallbackExchange];
}
