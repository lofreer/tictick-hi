<template>
  <div class="state-block">
    <NSpace vertical align="center" :size="10">
      <Inbox :size="28" />
      <NText>{{ title ?? t("common.noData") }}</NText>
    </NSpace>
  </div>
</template>

<script setup lang="ts">
import { Inbox } from "@lucide/vue";
import { NSpace, NText } from "naive-ui";
import { useI18n } from "vue-i18n";

defineProps<{ title?: string }>();

const { t } = useI18n();
</script>
