<template>
  <section class="page">
    <header class="page-header">
      <div>
        <h1 class="page-title">{{ t("page.notifications.title") }}</h1>
        <p class="page-subtitle">{{ t("system.notificationsSubtitle") }}</p>
      </div>
      <NButton type="primary" @click="createOpen = true">
        <template #icon><Plus :size="17" /></template>
        {{ t("system.createChannel") }}
      </NButton>
    </header>

    <section class="surface system-panel">
      <LoadingState v-if="loading" />
      <ErrorState v-else-if="error" :title="error" retryable @retry="loadChannels" />
      <EmptyState v-else-if="channels.length === 0" :title="t('system.noChannels')" />
      <div v-else class="system-table-wrap">
        <table class="system-table">
          <thead>
            <tr>
              <th>{{ t("system.name") }}</th>
              <th>{{ t("system.provider") }}</th>
              <th>{{ t("system.target") }}</th>
              <th>{{ t("system.enabled") }}</th>
              <th>{{ t("backtests.createdAt") }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="channel in channels" :key="channel.id">
              <td>{{ channel.name }}</td>
              <td>{{ channel.provider }}</td>
              <td>{{ channel.target }}</td>
              <td><NTag :type="channel.enabled ? 'success' : 'default'" size="small">{{ enabledLabel(channel.enabled) }}</NTag></td>
              <td>{{ formatDate(channel.createdAt) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <NModal v-model:show="createOpen" preset="card" :title="t('system.createChannel')" class="system-modal">
      <NForm label-placement="top">
        <NFormItem :label="t('system.name')"><NInput v-model:value="form.name" /></NFormItem>
        <NFormItem :label="t('system.provider')"><NInput v-model:value="form.provider" /></NFormItem>
        <NFormItem :label="t('system.target')"><NInput v-model:value="form.target" /></NFormItem>
        <NFormItem :label="t('system.enabled')"><NSwitch v-model:value="form.enabled" /></NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="createOpen = false">{{ t("common.cancel") }}</NButton>
          <NButton type="primary" :loading="creating" @click="createChannel">{{ t("common.create") }}</NButton>
        </NSpace>
      </template>
    </NModal>
  </section>
</template>

<script setup lang="ts">
import { Plus } from "@lucide/vue";
import { NButton, NForm, NFormItem, NInput, NModal, NSpace, NSwitch, NTag, useMessage } from "naive-ui";
import { onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";

import EmptyState from "@/components/common/EmptyState.vue";
import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import { systemApi } from "@/services/api/system";
import type { NotificationChannel } from "@/types/app";

const { t } = useI18n();
const message = useMessage();
const channels = ref<NotificationChannel[]>([]);
const loading = ref(false);
const creating = ref(false);
const error = ref("");
const createOpen = ref(false);
const form = reactive({ name: "", provider: "webhook", target: "", enabled: true });

onMounted(() => {
  void loadChannels();
});

async function loadChannels() {
  loading.value = true;
  error.value = "";
  try {
    channels.value = await systemApi.listNotificationChannels();
  } catch (loadError) {
    channels.value = [];
    error.value = errorMessage(loadError, t("system.channelsLoadFailed"));
  } finally {
    loading.value = false;
  }
}

async function createChannel() {
  creating.value = true;
  try {
    await systemApi.createNotificationChannel({ ...form });
    createOpen.value = false;
    message.success(t("system.created"));
    form.name = "";
    form.target = "";
    await loadChannels();
  } catch (loadError) {
    message.error(errorMessage(loadError, t("system.createFailed")));
  } finally {
    creating.value = false;
  }
}

function enabledLabel(enabled: boolean) {
  return enabled ? t("common.yes") : t("common.no");
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
  min-width: 760px;
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
