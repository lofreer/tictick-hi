<template>
  <NConfigProvider
    :theme="naiveTheme"
    :theme-overrides="themeOverrides"
    :locale="naiveLocale"
    :date-locale="naiveDateLocale"
  >
    <NLoadingBarProvider>
      <NDialogProvider>
        <NMessageProvider>
          <slot />
        </NMessageProvider>
      </NDialogProvider>
    </NLoadingBarProvider>
  </NConfigProvider>
</template>

<script setup lang="ts">
import {
  darkTheme,
  dateEnUS,
  dateZhCN,
  enUS,
  NConfigProvider,
  NDialogProvider,
  NLoadingBarProvider,
  NMessageProvider,
  zhCN,
} from "naive-ui";
import { computed, watchEffect } from "vue";
import { useI18n } from "vue-i18n";

import { useLocaleStore } from "@/stores/locale";
import { useThemeStore } from "@/stores/theme";
import { themeOverrides } from "@/theme/tokens";

const themeStore = useThemeStore();
const localeStore = useLocaleStore();
const { locale } = useI18n({ useScope: "global" });

watchEffect(() => {
  locale.value = localeStore.locale;
});

const naiveTheme = computed(() => (themeStore.mode === "dark" ? darkTheme : null));
const naiveLocale = computed(() => (localeStore.locale === "zh-CN" ? zhCN : enUS));
const naiveDateLocale = computed(() => (localeStore.locale === "zh-CN" ? dateZhCN : dateEnUS));
</script>

