<template>
  <section class="login-page">
    <NCard class="login-panel" :bordered="false">
      <NSpace vertical :size="22">
        <BrandMark />
        <div>
          <h1 class="page-title">{{ t("auth.loginTitle") }}</h1>
          <p class="page-subtitle">{{ t("auth.loginSubtitle") }}</p>
        </div>
        <NAlert type="info" :bordered="false">{{ t("auth.demoHint") }}</NAlert>
        <NForm @submit.prevent="submit">
          <NFormItem :label="t('auth.username')">
            <NInput v-model:value="username" :placeholder="t('auth.usernamePlaceholder')" />
          </NFormItem>
          <NFormItem :label="t('auth.password')">
            <NInput
              v-model:value="password"
              type="password"
              show-password-on="click"
              :placeholder="t('auth.passwordPlaceholder')"
            />
          </NFormItem>
          <NButton attr-type="submit" block type="primary">{{ t("auth.login") }}</NButton>
        </NForm>
      </NSpace>
    </NCard>
  </section>
</template>

<script setup lang="ts">
import { NAlert, NButton, NCard, NForm, NFormItem, NInput, NSpace, useMessage } from "naive-ui";
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";

import BrandMark from "@/components/layout/BrandMark.vue";
import { useAuthStore } from "@/stores/auth";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const message = useMessage();
const authStore = useAuthStore();

const username = ref("admin");
const password = ref("");

function submit() {
  if (!username.value.trim() || !password.value.trim()) {
    message.error(t("auth.required"));
    return;
  }

  authStore.login(username.value);
  const redirect = typeof route.query.redirect === "string" ? route.query.redirect : "/overview";
  router.push(redirect);
}
</script>

