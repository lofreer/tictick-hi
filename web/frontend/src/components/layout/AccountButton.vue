<template>
  <NButton class="header-tool" quaternary>
    <template #icon>
      <UserRound :size="18" />
    </template>
    <span class="header-tool__label">{{ authStore.operatorName || t("auth.account") }}</span>
  </NButton>
</template>

<script setup lang="ts">
import { UserRound } from "@lucide/vue";
import { NButton } from "naive-ui";
import { useI18n } from "vue-i18n";

import { useAuthStore } from "@/stores/auth";

const { t } = useI18n();
const authStore = useAuthStore();
</script>
