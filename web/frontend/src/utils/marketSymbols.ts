export type MarketSymbolOption = {
  label: string;
  value: string;
};

const fallbackExchange = "binance";

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
  return optionsForExchange(exchange).some((option) => option.value === symbol);
}

export function coerceSymbolForExchange(exchange: string, symbol: string) {
  return isSymbolForExchange(exchange, symbol) ? symbol : defaultSymbolForExchange(exchange);
}

function optionsForExchange(exchange: string) {
  return marketSymbolOptions[exchange] ?? marketSymbolOptions[fallbackExchange];
}
