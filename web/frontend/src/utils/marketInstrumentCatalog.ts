import { marketApi } from "@/services/api/market";

export type MarketInstrumentCatalogStatus = "active" | "inactive" | "missing";

export async function readMarketInstrumentCatalogStatus(
  exchange: string,
  symbol: string,
): Promise<MarketInstrumentCatalogStatus> {
  const instruments = await marketApi.listInstruments({
    exchange,
    limit: 1,
    q: symbol,
    status: "all",
  });
  const exact = instruments.find((instrument) => instrument.exchange === exchange && instrument.symbol === symbol);
  if (!exact) return "missing";
  return exact.status === "active" ? "active" : "inactive";
}
