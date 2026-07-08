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
          <NButton attr-type="submit" block type="primary" :loading="submitting">
            {{ t("auth.login") }}
          </NButton>
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
const submitting = ref(false);

async function submit() {
  if (!username.value.trim() || !password.value.trim()) {
    message.error(t("auth.required"));
    return;
  }

  submitting.value = true;
  try {
    await authStore.login(username.value, password.value);
    const redirect = typeof route.query.redirect === "string" ? route.query.redirect : "/overview";
    await router.push(redirect);
  } catch {
    message.error(t("auth.invalid"));
  } finally {
    submitting.value = false;
  }
}
</script>

<style scoped>
.login-panel {
  background:
    linear-gradient(180deg, var(--tt-surface) 0, color-mix(in srgb, var(--tt-surface-raised) 72%, var(--tt-surface)) 100%),
    var(--tt-surface);
}

.login-panel :deep(.n-card__content) {
  padding: 26px;
}

.login-panel :deep(.n-alert) {
  border: 1px solid var(--tt-line-soft);
  background: var(--tt-surface-subtle);
}

.login-panel :deep(.n-button) {
  min-height: 38px;
}

@media (max-width: 620px) {
  .login-panel :deep(.n-card__content) {
    padding: 22px;
  }
}
</style>
