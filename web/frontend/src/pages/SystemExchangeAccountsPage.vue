<template>
  <section class="page">
    <header class="page-header">
      <div>
        <h1 class="page-title">{{ t("page.exchangeAccounts.title") }}</h1>
        <p class="page-subtitle">{{ t("system.exchangeAccountsSubtitle") }}</p>
      </div>
      <NButton type="primary" @click="createOpen = true">
        <template #icon><Plus :size="17" /></template>
        {{ t("system.createAccount") }}
      </NButton>
    </header>

    <section class="surface system-panel">
      <LoadingState v-if="loading" />
      <ErrorState v-else-if="error" :title="error" retryable @retry="loadAccounts" />
      <EmptyState v-else-if="accounts.length === 0" :title="t('system.noExchangeAccounts')" />
      <div v-else class="system-table-wrap">
        <table class="system-table">
          <thead>
            <tr>
              <th>{{ t("research.exchange") }}</th>
              <th>{{ t("system.alias") }}</th>
              <th>{{ t("system.enabled") }}</th>
              <th>{{ t("system.credentials") }}</th>
              <th>{{ t("backtests.createdAt") }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="account in accounts" :key="account.id">
              <td>{{ account.exchange }}</td>
              <td>{{ account.alias }}</td>
              <td><NTag :type="account.enabled ? 'success' : 'default'" size="small">{{ enabledLabel(account.enabled) }}</NTag></td>
              <td><NTag :type="credentialType(account.credentialStatus)" size="small">{{ credentialLabel(account.credentialStatus) }}</NTag></td>
              <td>{{ formatDate(account.createdAt) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <NModal v-model:show="createOpen" preset="card" :title="t('system.createAccount')" class="system-modal">
      <NForm label-placement="top">
        <NFormItem :label="t('research.exchange')"><NInput v-model:value="form.exchange" /></NFormItem>
        <NFormItem :label="t('system.alias')"><NInput v-model:value="form.alias" /></NFormItem>
        <NFormItem :label="t('system.apiKey')"><NInput v-model:value="form.apiKey" type="password" /></NFormItem>
        <NFormItem :label="t('system.apiSecret')"><NInput v-model:value="form.apiSecret" type="password" /></NFormItem>
        <NFormItem :label="t('system.enabled')"><NSwitch v-model:value="form.enabled" /></NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="createOpen = false">{{ t("common.cancel") }}</NButton>
          <NButton type="primary" :loading="creating" @click="createAccount">{{ t("common.create") }}</NButton>
        </NSpace>
      </template>
    </NModal>
  </section>
</template>

<script setup lang="ts">
import { Plus } from "@lucide/vue";
import { NButton, NForm, NFormItem, NInput, NModal, NSpace, NSwitch, NTag, type TagProps, useMessage } from "naive-ui";
import { onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";

import EmptyState from "@/components/common/EmptyState.vue";
import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import { systemApi } from "@/services/api/system";
import type { ExchangeAccount } from "@/types/app";

const { t } = useI18n();
const message = useMessage();
const accounts = ref<ExchangeAccount[]>([]);
const loading = ref(false);
const creating = ref(false);
const error = ref("");
const createOpen = ref(false);
const form = reactive({ exchange: "binance", alias: "", apiKey: "", apiSecret: "", enabled: true });

onMounted(() => {
  void loadAccounts();
});

async function loadAccounts() {
  loading.value = true;
  error.value = "";
  try {
    accounts.value = await systemApi.listExchangeAccounts();
  } catch (loadError) {
    accounts.value = [];
    error.value = errorMessage(loadError, t("system.accountsLoadFailed"));
  } finally {
    loading.value = false;
  }
}

async function createAccount() {
  creating.value = true;
  try {
    await systemApi.createExchangeAccount({ ...form });
    createOpen.value = false;
    message.success(t("system.created"));
    form.alias = "";
    form.apiKey = "";
    form.apiSecret = "";
    await loadAccounts();
  } catch (loadError) {
    message.error(errorMessage(loadError, t("system.createFailed")));
  } finally {
    creating.value = false;
  }
}

function enabledLabel(enabled: boolean) {
  return enabled ? t("common.yes") : t("common.no");
}

function credentialLabel(status: string) {
  return status === "encrypted" ? t("system.credentialsEncrypted") : t("system.credentialsLegacy");
}

function credentialType(status: string): TagProps["type"] {
  return status === "encrypted" ? "success" : "warning";
}

function formatDate(value?: string) {
  return value ? new Date(value).toLocaleString() : "-";
}

function errorMessage(loadError: unknown, fallback: string) {
  return loadError instanceof Error && loadError.message ? loadError.message : fallback;
}
</script>

<style scoped>
.system-panel {
  overflow: hidden;
}

.system-table-wrap {
  overflow-x: auto;
}

.system-table {
  width: 100%;
  min-width: 720px;
  border-collapse: collapse;
}

.system-table th,
.system-table td {
  padding: 12px 14px;
  border-bottom: 1px solid var(--tt-line);
  font-size: 13px;
  line-height: 1.5;
  text-align: left;
}

.system-table th {
  color: var(--tt-muted);
  font-weight: 720;
}

.system-table tbody tr:last-child td {
  border-bottom: 0;
}

:global(.system-modal) {
  width: min(560px, calc(100vw - 32px));
}
</style>
