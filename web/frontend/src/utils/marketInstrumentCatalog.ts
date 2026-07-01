import { marketApi } from "@/services/api/market";
import type { MarketInstrument } from "@/types/app";

export type MarketInstrumentCatalogStatus = "active" | "inactive" | "missing";

export type MarketInstrumentCatalogLookup = {
  instrument?: MarketInstrument;
  status: MarketInstrumentCatalogStatus;
};

export async function readMarketInstrumentCatalogLookup(
  exchange: string,
  symbol: string,
): Promise<MarketInstrumentCatalogLookup> {
  const instruments = await marketApi.listInstruments({
    exchange,
    limit: 1,
    q: symbol,
    status: "all",
  });
  const exact = instruments.find((instrument) => instrument.exchange === exchange && instrument.symbol === symbol);
  if (!exact) return { status: "missing" };
  return {
    instrument: exact,
    status: exact.status === "active" ? "active" : "inactive",
  };
}

export async function readMarketInstrumentCatalogStatus(
  exchange: string,
  symbol: string,
): Promise<MarketInstrumentCatalogStatus> {
  return (await readMarketInstrumentCatalogLookup(exchange, symbol)).status;
}
