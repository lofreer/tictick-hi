import { createRouter, createWebHistory } from "vue-router";

import { routes } from "@/router/routes";
import { useAuthStore } from "@/stores/auth";

export const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach(async (to) => {
  const authStore = useAuthStore();

  if (to.meta.public) {
    return true;
  }

  await authStore.restoreSession();

  if (!authStore.isAuthenticated) {
    return { name: "login", query: { redirect: to.fullPath } };
  }

  return true;
});
