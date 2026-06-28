import { describe, expect, it } from "vitest";

import {
  coerceSymbolForExchange,
  defaultSymbolForExchange,
  isSymbolForExchange,
  symbolOptionsForExchange,
} from "@/utils/marketSymbols";

describe("marketSymbols", () => {
  it("returns exchange-specific symbol options", () => {
    expect(symbolOptionsForExchange("binance").map((option) => option.value)).toEqual(["BTCUSDT", "ETHUSDT"]);
    expect(symbolOptionsForExchange("okx").map((option) => option.value)).toEqual(["BTC-USDT", "ETH-USDT"]);
  });

  it("coerces mismatched symbols to the selected exchange default", () => {
    expect(coerceSymbolForExchange("binance", "BTC-USDT")).toBe("BTCUSDT");
    expect(coerceSymbolForExchange("okx", "BTCUSDT")).toBe("BTC-USDT");
    expect(coerceSymbolForExchange("okx", "ETH-USDT")).toBe("ETH-USDT");
  });

  it("falls back to binance options for unknown exchanges", () => {
    expect(defaultSymbolForExchange("unknown")).toBe("BTCUSDT");
    expect(isSymbolForExchange("unknown", "BTCUSDT")).toBe(true);
    expect(isSymbolForExchange("unknown", "BTC-USDT")).toBe(false);
  });
});
