<template>
  <NDropdown :options="options" trigger="click" @select="handleSelect">
    <NButton class="header-tool header-tool--menu" quaternary>
      <template #icon>
        <Settings :size="17" />
      </template>
      {{ t("nav.system") }}
      <ChevronDown :size="15" />
    </NButton>
  </NDropdown>
</template>

<script setup lang="ts">
import { Activity, Bell, ChevronDown, KeyRound, Settings, UsersRound } from "@lucide/vue";
import { NButton, NDropdown, type DropdownOption } from "naive-ui";
import { h, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";

const router = useRouter();
const { t } = useI18n();

const options = computed<DropdownOption[]>(() => [
  option("notifications", "system.notifications", Bell),
  option("exchange-accounts", "system.exchangeAccounts", KeyRound),
  option("operators", "system.operators", UsersRound),
  option("health", "system.health", Activity),
]);

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
