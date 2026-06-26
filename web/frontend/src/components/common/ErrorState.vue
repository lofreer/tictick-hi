<template>
  <div class="state-block">
    <NSpace vertical align="center" :size="12">
      <CircleAlert :size="28" />
      <NText type="error">{{ title ?? t("common.error") }}</NText>
      <NButton v-if="retryable" size="small" @click="$emit('retry')">{{ t("common.retry") }}</NButton>
    </NSpace>
  </div>
</template>

<script setup lang="ts">
import { CircleAlert } from "@lucide/vue";
import { NButton, NSpace, NText } from "naive-ui";
import { useI18n } from "vue-i18n";

defineProps<{ title?: string; retryable?: boolean }>();
defineEmits<{ retry: [] }>();

const { t } = useI18n();
</script>
