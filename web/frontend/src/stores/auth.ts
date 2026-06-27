import { defineStore } from "pinia";
import { computed, ref } from "vue";

import { ApiError } from "@/services/api/client";
import { authApi } from "@/services/api/auth";
import type { Operator } from "@/types/app";

export const useAuthStore = defineStore("auth", () => {
  const operator = ref<Operator | null>(null);
  const initialized = ref(false);
  const isAuthenticated = computed(() => operator.value !== null);
  const operatorName = computed(() => operator.value?.username ?? "");

  async function restoreSession() {
    if (initialized.value) {
      return operator.value;
    }
    try {
      operator.value = await authApi.me();
      return operator.value;
    } catch (error) {
      if (!(error instanceof ApiError) || error.status !== 401) {
        throw error;
      }
      operator.value = null;
      return null;
    } finally {
      initialized.value = true;
    }
  }

  async function login(username: string, password: string) {
    operator.value = await authApi.login({ username: username.trim(), password });
    initialized.value = true;
    return operator.value;
  }

  async function logout() {
    try {
      await authApi.logout();
    } finally {
      operator.value = null;
      initialized.value = true;
    }
  }

  return { operator, operatorName, initialized, isAuthenticated, restoreSession, login, logout };
});
