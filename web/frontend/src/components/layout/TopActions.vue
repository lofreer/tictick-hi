<template>
  <div class="top-actions">
    <SystemMenu />
    <AccountButton />
    <LocaleSwitch />
    <ThemeToggle />
    <NTooltip trigger="hover">
      <template #trigger>
        <NButton class="header-tool" circle quaternary :aria-label="t('auth.logout')" @click="logout">
          <template #icon>
            <LogOut :size="18" />
          </template>
        </NButton>
      </template>
      {{ t("auth.logout") }}
    </NTooltip>
  </div>
</template>

<script setup lang="ts">
import { LogOut } from "@lucide/vue";
import { NButton, NTooltip } from "naive-ui";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";

import AccountButton from "@/components/layout/AccountButton.vue";
import LocaleSwitch from "@/components/layout/LocaleSwitch.vue";
import SystemMenu from "@/components/layout/SystemMenu.vue";
import ThemeToggle from "@/components/layout/ThemeToggle.vue";
import { useAuthStore } from "@/stores/auth";

const { t } = useI18n();
const router = useRouter();
const authStore = useAuthStore();

async function logout() {
  await authStore.logout();
  await router.push("/login");
}
</script>
