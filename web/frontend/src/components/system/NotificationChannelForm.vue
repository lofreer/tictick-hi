<template>
  <NForm label-placement="top">
    <NFormItem :label="t('system.name')"><NInput v-model:value="form.name" /></NFormItem>
    <NFormItem :label="t('system.provider')"><NSelect v-model:value="form.provider" :options="providerOptions" /></NFormItem>
    <NFormItem :label="t('system.target')"><NInput v-model:value="form.target" /></NFormItem>
    <NFormItem :label="t('system.enabled')"><NSwitch v-model:value="form.enabled" /></NFormItem>
  </NForm>
</template>

<script setup lang="ts">
import { NForm, NFormItem, NInput, NSelect, NSwitch, type SelectOption } from "naive-ui";
import { useI18n } from "vue-i18n";

type NotificationChannelFormModel = {
  name: string;
  provider: string;
  target: string;
  enabled: boolean;
};

defineProps<{
  form: NotificationChannelFormModel;
  providerOptions: SelectOption[];
}>();

const { t } = useI18n();
</script>
