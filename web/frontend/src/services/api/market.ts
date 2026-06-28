import { apiClient } from "@/services/api/client";
import type { MarketInstrument } from "@/types/app";

export type MarketInstrumentQuery = {
  exchange: string;
  limit?: number;
  q?: string;
};

export const marketApi = {
  listInstruments(query: MarketInstrumentQuery) {
    const params = new URLSearchParams({ exchange: query.exchange });
    if (query.q) {
      params.set("q", query.q);
    }
    if (query.limit) {
      params.set("limit", String(query.limit));
    }
    return apiClient.get<MarketInstrument[]>(`/market/instruments?${params.toString()}`);
  },
};
