import { apiClient } from "@/services/api/client";
import type {
  ChangeOperatorPasswordRequest,
  ChangeOperatorPasswordResult,
  LoginCredentials,
  Operator,
  OperatorSession,
} from "@/types/app";

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

  changePassword(request: ChangeOperatorPasswordRequest) {
    return apiClient.post<ChangeOperatorPasswordResult>("/auth/password", request);
  },

  listSessions() {
    return apiClient.get<OperatorSession[]>("/auth/sessions");
  },

  async revokeSession(id: string) {
    await apiClient.delete<{ status: string }>(`/auth/sessions/${encodeURIComponent(id)}`);
  },
};
