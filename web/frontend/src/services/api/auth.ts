import { apiClient } from "@/services/api/client";
import type { LoginCredentials, Operator } from "@/types/app";

export const authApi = {
  login(credentials: LoginCredentials) {
    return apiClient.post<Operator>("/auth/login", credentials);
  },

  me() {
    return apiClient.get<Operator>("/auth/me");
  },

  async logout() {
    await apiClient.post<{ status: string }>("/auth/logout");
  },
};
