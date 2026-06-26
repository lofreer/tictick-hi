<template>
  <div class="state-block">
    <NSpace vertical align="center" :size="10">
      <NSpin size="small" />
      <NText>{{ title ?? t("common.loading") }}</NText>
    </NSpace>
  </div>
</template>

<script setup lang="ts">
import { NSpace, NSpin, NText } from "naive-ui";
import { useI18n } from "vue-i18n";

defineProps<{ title?: string }>();

const { t } = useI18n();
</script>

