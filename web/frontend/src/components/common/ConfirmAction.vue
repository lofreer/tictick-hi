<template>
  <NPopconfirm :positive-text="t('common.confirm')" :negative-text="t('common.cancel')" @positive-click="$emit('confirm')">
    <template #trigger>
      <slot />
    </template>
    {{ message }}
  </NPopconfirm>
</template>

<script setup lang="ts">
import { NPopconfirm } from "naive-ui";
import { useI18n } from "vue-i18n";

defineProps<{ message: string }>();
defineEmits<{ confirm: [] }>();

const { t } = useI18n();
</script>

