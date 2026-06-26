import { defineStore } from "pinia";
import { computed, ref } from "vue";

const storageKey = "tictick-hi.operator";

export const useAuthStore = defineStore("auth", () => {
  const operator = ref(localStorage.getItem(storageKey) ?? "");
  const isAuthenticated = computed(() => operator.value.length > 0);

  function login(username: string) {
    operator.value = username.trim();
    localStorage.setItem(storageKey, operator.value);
  }

  function logout() {
    operator.value = "";
    localStorage.removeItem(storageKey);
  }

  return { operator, isAuthenticated, login, logout };
});

