<template>
  <NDropdown :options="options" trigger="click" @select="handleSelect">
    <NButton class="header-tool header-tool--menu" quaternary>
      <template #icon>
        <Settings :size="17" />
      </template>
      <span class="header-tool__label">{{ t("nav.system") }}</span>
      <ChevronDown :size="15" />
    </NButton>
  </NDropdown>
</template>

<script setup lang="ts">
import { Activity, Bell, ChevronDown, FileText, KeyRound, Settings, ShieldCheck, UsersRound } from "@lucide/vue";
import { NButton, NDropdown, type DropdownOption } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";

import { useAuthStore } from "@/stores/auth";

const router = useRouter();
const { t } = useI18n();
const authStore = useAuthStore();
const canManageSystemConfig = computed(() => authStore.operator?.role === "admin");

const options = computed<DropdownOption[]>(() => {
  const entries = canManageSystemConfig.value
    ? [
        option("notifications", "system.notifications", Bell),
        option("exchange-accounts", "system.exchangeAccounts", KeyRound),
        option("operators", "system.operators", UsersRound),
      ]
    : [];
  entries.push(option("sessions", "system.sessions", ShieldCheck));
  if (canManageSystemConfig.value) {
    entries.push(option("audit-events", "system.auditEvents", FileText));
  }
  entries.push(option("health", "system.health", Activity));
  return entries;
});

function option(key: string, labelKey: string, icon: typeof Bell): DropdownOption {
  return {
    key,
    label: t(labelKey),
    icon: () => h(icon, { size: 16 }),
  };
}

function handleSelect(key: string) {
  router.push(`/system/${key}`);
}
</script>
