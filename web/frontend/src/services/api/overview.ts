import { apiClient } from "@/services/api/client";
import type { OverviewRecentFacts } from "@/types/app";

export type OverviewRecentFactsOptions = {
  limit?: number;
  since?: string;
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
};
