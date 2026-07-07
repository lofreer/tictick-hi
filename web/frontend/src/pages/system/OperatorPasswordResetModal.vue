<template>
  <NModal :show="show" preset="card" :title="title" class="system-modal" @update:show="emit('update:show', $event)">
    <NForm label-placement="top">
      <NFormItem :label="t('auth.username')"><NInput :value="username" disabled /></NFormItem>
      <NFormItem :label="t('system.newPassword')">
        <NInput
          :value="newPassword"
          type="password"
          show-password-on="mousedown"
          @update:value="emit('update:newPassword', $event)"
        />
      </NFormItem>
    </NForm>
    <template #footer>
      <NSpace justify="end">
        <NButton @click="emit('update:show', false)">{{ t("common.cancel") }}</NButton>
        <NButton type="primary" :loading="resetting" @click="emit('submit')">
          {{ t("system.updateOperatorPassword") }}
        </NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<script setup lang="ts">
import { NButton, NForm, NFormItem, NInput, NModal, NSpace } from "naive-ui";
import { useI18n } from "vue-i18n";

defineProps<{
  newPassword: string;
  resetting: boolean;
  show: boolean;
  title: string;
  username: string;
}>();

const emit = defineEmits<{
  (event: "submit"): void;
  (event: "update:newPassword", value: string): void;
  (event: "update:show", value: boolean): void;
}>();

const { t } = useI18n();
</script>

<style scoped>
:global(.system-modal) {
  width: min(560px, calc(100vw - 32px));
}
</style>
