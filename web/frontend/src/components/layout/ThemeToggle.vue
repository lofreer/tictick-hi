<template>
  <NTooltip trigger="hover">
    <template #trigger>
      <NButton class="header-tool" circle quaternary :aria-label="label" @click="themeStore.toggleTheme">
        <template #icon>
          <Sun v-if="themeStore.mode === 'dark'" :size="18" />
          <Moon v-else :size="18" />
        </template>
      </NButton>
    </template>
    {{ label }}
  </NTooltip>
</template>

<script setup lang="ts">
import { Moon, Sun } from "@lucide/vue";
import { NButton, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import { useThemeStore } from "@/stores/theme";

const { t } = useI18n();
const themeStore = useThemeStore();
const label = computed(() =>
  themeStore.mode === "dark" ? t("common.themeLight") : t("common.themeDark"),
);
</script>
