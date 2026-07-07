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
      <div class="system-panel__header">
        <h2>{{ t("system.notifications") }}</h2>
        <NRadioGroup :value="notificationStatusFilter" size="small" :aria-label="t('system.notificationStatusFilter')" @update:value="setNotificationStatusFilter">
          <NRadioButton v-for="option in notificationStatusFilterOptions" :key="option.value" :value="option.value">
            {{ option.label }}
          </NRadioButton>
        </NRadioGroup>
      </div>
      <LoadingState v-if="loading" />
      <ErrorState v-else-if="error" :title="error" retryable @retry="loadAll" />
      <EmptyState v-else-if="notifications.length === 0" :title="t('system.noNotifications')" />
      <EmptyState v-else-if="filteredNotifications.length === 0" :title="t('system.noNotificationsForFilter')" />
      <div v-else class="system-table-wrap">
        <table class="system-table system-table--wide">
          <thead>
            <tr>
              <th>{{ t("system.status") }}</th>
              <th>{{ t("system.channel") }}</th>
              <th>{{ t("system.provider") }}</th>
              <th>{{ t("system.title") }}</th>
              <th>{{ t("system.attempts") }}</th>
              <th>{{ t("system.nextAttempt") }}</th>
              <th>{{ t("common.error") }}</th>
              <th>{{ t("research.actions") }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="notification in filteredNotifications" :key="notification.id">
              <td><NTag :type="statusType(notification.status)" size="small">{{ notification.status }}</NTag></td>
              <td>{{ notification.channel }}</td>
              <td>{{ notification.provider }}</td>
              <td>
                <strong>{{ notification.title }}</strong>
                <span class="system-table__muted">{{ notification.body }}</span>
              </td>
              <td>{{ notification.attemptCount }} / {{ notification.maxAttempts }}</td>
              <td>{{ formatDate(notification.nextAttemptAt) }}</td>
              <td><span class="system-table__error">{{ notification.error || "-" }}</span></td>
              <td>
                <NButton
                  size="tiny"
                  quaternary
                  :disabled="notification.status !== 'failed'"
                  :loading="retryingId === notification.id"
                  @click="retryNotification(notification.id)"
                >
                  {{ t("common.retry") }}
                </NButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="surface system-panel">
      <LoadingState v-if="channelsLoading" />
      <ErrorState v-else-if="channelsError" :title="channelsError" retryable @retry="loadChannels" />
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
              <th>{{ t("research.actions") }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="channel in channels" :key="channel.id">
              <td>{{ channel.name }}</td>
              <td>{{ channel.provider }}</td>
              <td>{{ channel.target }}</td>
              <td><NTag :type="channel.enabled ? 'success' : 'default'" size="small">{{ enabledLabel(channel.enabled) }}</NTag></td>
              <td>{{ formatDate(channel.createdAt) }}</td>
              <td>
                <NButton
                  size="small"
                  :type="channel.enabled ? 'warning' : 'primary'"
                  secondary
                  :loading="updatingChannelId === channel.id"
                  @click="toggleChannel(channel)"
                >
                  <template #icon>
                    <PowerOff v-if="channel.enabled" :size="16" />
                    <Power v-else :size="16" />
                  </template>
                  {{ channel.enabled ? t("system.disableChannel") : t("system.enableChannel") }}
                </NButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <NModal v-model:show="createOpen" preset="card" :title="t('system.createChannel')" class="system-modal">
      <NForm label-placement="top">
        <NFormItem :label="t('system.name')"><NInput v-model:value="form.name" /></NFormItem>
        <NFormItem :label="t('system.provider')"><NSelect v-model:value="form.provider" :options="providerOptions" /></NFormItem>
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
import { Plus, Power, PowerOff } from "@lucide/vue";
import {
  NButton,
  NForm,
  NFormItem,
  NInput,
  NModal,
  NRadioButton,
  NRadioGroup,
  NSelect,
  NSpace,
  NSwitch,
  NTag,
  type SelectOption,
  type TagProps,
  useMessage,
} from "naive-ui";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";

import EmptyState from "@/components/common/EmptyState.vue";
import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import { systemApi } from "@/services/api/system";
import type { Notification, NotificationChannel } from "@/types/app";
import {
  notificationMatchesStatusFilter,
  notificationStatusFilterFromQuery,
  notificationStatusQueryValue,
  type NotificationStatusFilter,
} from "./systemNotificationsFilters";

const { t } = useI18n();
const message = useMessage();
const route = useRoute();
const router = useRouter();
const channels = ref<NotificationChannel[]>([]);
const notifications = ref<Notification[]>([]);
const loading = ref(false);
const channelsLoading = ref(false);
const creating = ref(false);
const error = ref("");
const channelsError = ref("");
const updatingChannelId = ref("");
const createOpen = ref(false);
const retryingId = ref("");
const notificationStatusFilter = ref<NotificationStatusFilter>(notificationStatusFilterFromQuery(route.query.status));
const form = reactive({ name: "", provider: "local", target: "default", enabled: true });
const providerOptions: SelectOption[] = [
  { label: "local", value: "local" },
  { label: "email", value: "email" },
  { label: "telegram", value: "telegram" },
  { label: "feishu", value: "feishu" },
  { label: "webhook", value: "webhook" },
  { label: "webhook-demo", value: "webhook-demo" },
];
const notificationStatusFilterOptions = computed<SelectOption[]>(() => [
  { label: t("system.notificationStatus.all"), value: "all" },
  { label: t("system.notificationStatus.failed"), value: "failed" },
  { label: t("system.notificationStatus.pending"), value: "pending" },
  { label: t("system.notificationStatus.sent"), value: "sent" },
]);
const filteredNotifications = computed(() =>
  notifications.value.filter((notification) => notificationMatchesStatusFilter(notification, notificationStatusFilter.value)),
);

onMounted(() => {
  void loadAll();
});

watch(
  () => route.query.status,
  (value) => {
    notificationStatusFilter.value = notificationStatusFilterFromQuery(value);
  },
);

async function loadAll() {
  loading.value = true;
  error.value = "";
  try {
    await Promise.all([loadNotifications(), loadChannels()]);
  } catch (loadError) {
    error.value = errorMessage(loadError, t("system.notificationsLoadFailed"));
  } finally {
    loading.value = false;
  }
}

async function loadNotifications() {
  notifications.value = await systemApi.listNotifications();
}

async function loadChannels() {
  channelsLoading.value = true;
  channelsError.value = "";
  try {
    channels.value = await systemApi.listNotificationChannels();
  } catch (loadError) {
    channels.value = [];
    channelsError.value = errorMessage(loadError, t("system.channelsLoadFailed"));
  } finally {
    channelsLoading.value = false;
  }
}

async function createChannel() {
  creating.value = true;
  try {
    await systemApi.createNotificationChannel({ ...form });
    createOpen.value = false;
    message.success(t("system.created"));
    form.name = "";
    form.target = "default";
    await loadChannels();
  } catch (loadError) {
    message.error(errorMessage(loadError, t("system.createFailed")));
  } finally {
    creating.value = false;
  }
}

async function toggleChannel(channel: NotificationChannel) {
  updatingChannelId.value = channel.id;
  try {
    await systemApi.setNotificationChannelEnabled(channel.id, !channel.enabled);
    message.success(t("system.channelUpdated"));
    await loadChannels();
  } catch (loadError) {
    message.error(errorMessage(loadError, t("system.channelUpdateFailed")));
  } finally {
    updatingChannelId.value = "";
  }
}

async function retryNotification(id: string) {
  retryingId.value = id;
  try {
    await systemApi.retryNotification(id);
    message.success(t("system.notificationRetried"));
    await loadNotifications();
  } catch (loadError) {
    message.error(errorMessage(loadError, t("system.notificationRetryFailed")));
  } finally {
    retryingId.value = "";
  }
}

async function setNotificationStatusFilter(value: string) {
  notificationStatusFilter.value = notificationStatusFilterFromQuery(value);
  const nextQuery = { ...route.query };
  const status = notificationStatusQueryValue(notificationStatusFilter.value);
  if (status) nextQuery.status = status;
  else delete nextQuery.status;
  await router.replace({ query: nextQuery });
}

function enabledLabel(enabled: boolean) {
  return enabled ? t("common.yes") : t("common.no");
}

function statusType(status: string): TagProps["type"] {
  if (status === "sent" || status === "delivered") return "success";
  if (status === "failed") return "error";
  if (status === "retry_scheduled") return "warning";
  if (status === "running") return "info";
  return "default";
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

.system-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 14px;
}

.system-panel__header h2 {
  margin: 0;
  font-size: 16px;
  line-height: 1.35;
  font-weight: 760;
}

.system-panel__header :deep(.n-radio-button) {
  min-width: 64px;
  text-align: center;
}

.system-table-wrap {
  overflow-x: auto;
}

.system-table {
  width: 100%;
  min-width: 760px;
  border-collapse: collapse;
}

.system-table--wide {
  min-width: 1120px;
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

.system-table__muted,
.system-table__error {
  display: block;
  max-width: 280px;
  overflow: hidden;
  color: var(--tt-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.system-table__error {
  color: var(--tt-danger);
}

.system-table tbody tr:last-child td {
  border-bottom: 0;
}

:global(.system-modal) {
  width: min(560px, calc(100vw - 32px));
}
</style>
