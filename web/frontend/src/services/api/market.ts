import { apiClient } from "@/services/api/client";
import type { MarketInstrument, MarketInstrumentSyncResult } from "@/types/app";

export type MarketInstrumentQuery = {
  exchange: string;
  limit?: number;
  q?: string;
  status?: "active" | "inactive" | "all";
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
    if (query.status) {
      params.set("status", query.status);
    }
    return apiClient.get<MarketInstrument[]>(`/market/instruments?${params.toString()}`);
  },
  syncInstruments(exchange: string) {
    const params = new URLSearchParams({ exchange });
    return apiClient.post<MarketInstrumentSyncResult>(`/market/instruments/sync?${params.toString()}`);
  },
};
