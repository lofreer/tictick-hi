import { describe, expect, it } from "vitest";

import {
  coerceSymbolForExchange,
  defaultSymbolForExchange,
  isSymbolFormatForExchange,
  isSymbolForExchange,
  normalizeSymbolInput,
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

  it("accepts exchange-formatted symbols beyond the suggestion options", () => {
    expect(isSymbolFormatForExchange("binance", "SOLUSDT")).toBe(true);
    expect(isSymbolFormatForExchange("binance", "1000PEPEUSDT")).toBe(true);
    expect(isSymbolFormatForExchange("okx", "SOL-USDT")).toBe(true);
    expect(isSymbolFormatForExchange("okx", "BTC-USDT-SWAP")).toBe(true);
    expect(isSymbolFormatForExchange("binance", "SOL-USDT")).toBe(false);
    expect(isSymbolFormatForExchange("okx", "SOLUSDT")).toBe(false);
  });

  it("normalizes user input before validation and coercion", () => {
    expect(normalizeSymbolInput(" solusdt ")).toBe("SOLUSDT");
    expect(normalizeSymbolInput(" btc-usdt-swap ")).toBe("BTC-USDT-SWAP");
    expect(coerceSymbolForExchange("binance", "solusdt")).toBe("SOLUSDT");
    expect(coerceSymbolForExchange("okx", "btc-usdt")).toBe("BTC-USDT");
  });

  it("falls back to binance options for unknown exchanges", () => {
    expect(defaultSymbolForExchange("unknown")).toBe("BTCUSDT");
    expect(isSymbolForExchange("unknown", "SOLUSDT")).toBe(true);
    expect(isSymbolForExchange("unknown", "BTC-USDT")).toBe(false);
  });
});
