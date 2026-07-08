<template>
  <NTooltip trigger="hover">
    <template #trigger>
      <NButton class="header-tool" quaternary @click="toggleLocale">
        <template #icon>
          <Languages :size="18" />
        </template>
        <span class="header-tool__label">{{ localeStore.locale === "zh-CN" ? "EN" : "中" }}</span>
      </NButton>
    </template>
    {{ t("common.language") }}
  </NTooltip>
</template>

<script setup lang="ts">
import { Languages } from "@lucide/vue";
import { NButton, NTooltip } from "naive-ui";
import { useI18n } from "vue-i18n";

import { useLocaleStore } from "@/stores/locale";

const { t, locale } = useI18n();
const localeStore = useLocaleStore();

function toggleLocale() {
  localeStore.toggleLocale();
  locale.value = localeStore.locale;
}
</script>
