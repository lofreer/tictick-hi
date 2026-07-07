import { apiClient } from "@/services/api/client";
import type { OverviewRecentFacts, OverviewTrends } from "@/types/app";

export type OverviewRecentFactsOptions = {
  limit?: number;
  since?: string;
};

export type OverviewTrendsOptions = {
  days?: number;
};

export const overviewApi = {
  recentFacts(options: OverviewRecentFactsOptions = {}) {
    const params = new URLSearchParams();
    if (options.limit !== undefined) params.set("limit", String(options.limit));
    if (options.since) params.set("since", options.since);
    const query = params.toString();
    if (query) {
      return apiClient.get<OverviewRecentFacts>(`/overview/recent-facts?${query}`);
    }
    return apiClient.get<OverviewRecentFacts>("/overview/recent-facts");
  },
  trends(options: OverviewTrendsOptions = {}) {
    const params = new URLSearchParams();
    if (options.days !== undefined) params.set("days", String(options.days));
    const query = params.toString();
    if (query) {
      return apiClient.get<OverviewTrends>(`/overview/trends?${query}`);
    }
    return apiClient.get<OverviewTrends>("/overview/trends");
  },
};
