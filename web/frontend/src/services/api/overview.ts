import { apiClient } from "@/services/api/client";
import type { OverviewRecentFacts } from "@/types/app";

export const overviewApi = {
  recentFacts(limit?: number) {
    if (limit === undefined) {
      return apiClient.get<OverviewRecentFacts>("/overview/recent-facts");
    }
    return apiClient.get<OverviewRecentFacts>(`/overview/recent-facts?limit=${encodeURIComponent(String(limit))}`);
  },
};
