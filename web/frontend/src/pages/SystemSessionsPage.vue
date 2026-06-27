<template>
  <section class="page">
    <header class="page-header">
      <div>
        <h1 class="page-title">{{ t("page.sessions.title") }}</h1>
        <p class="page-subtitle">{{ t("system.sessionsSubtitle") }}</p>
      </div>
      <NButton secondary @click="loadSessions">
        <template #icon><RefreshCw :size="17" /></template>
        {{ t("common.retry") }}
      </NButton>
    </header>

    <section class="surface system-panel">
      <LoadingState v-if="loading" />
      <ErrorState v-else-if="error" :title="error" retryable @retry="loadSessions" />
      <EmptyState v-else-if="sessions.length === 0" :title="t('system.noSessions')" />
      <div v-else class="system-table-wrap">
        <table class="system-table">
          <thead>
            <tr>
              <th>{{ t("system.sessionId") }}</th>
              <th>{{ t("system.status") }}</th>
              <th>{{ t("backtests.createdAt") }}</th>
              <th>{{ t("system.expiresAt") }}</th>
              <th>{{ t("research.actions") }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="session in sessions" :key="session.id">
              <td><span class="session-id">{{ session.id }}</span></td>
              <td>
                <NTag :type="session.current ? 'success' : 'default'" size="small">
                  {{ session.current ? t("system.currentSession") : t("system.activeSession") }}
                </NTag>
              </td>
              <td>{{ formatDate(session.createdAt) }}</td>
              <td>{{ formatDate(session.expiresAt) }}</td>
              <td>
                <NButton
                  size="small"
                  type="error"
                  secondary
                  :disabled="session.current"
                  :loading="revokingSessionId === session.id"
                  @click="revokeSession(session)"
                >
                  {{ t("system.revokeSession") }}
                </NButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </section>
</template>

<script setup lang="ts">
import { RefreshCw } from "@lucide/vue";
import { NButton, NTag, useMessage } from "naive-ui";
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";

import EmptyState from "@/components/common/EmptyState.vue";
import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import { authApi } from "@/services/api/auth";
import type { OperatorSession } from "@/types/app";

const { t } = useI18n();
const message = useMessage();
const sessions = ref<OperatorSession[]>([]);
const loading = ref(false);
const error = ref("");
const revokingSessionId = ref("");

onMounted(() => {
  void loadSessions();
});

async function loadSessions() {
  loading.value = true;
  error.value = "";
  try {
    sessions.value = await authApi.listSessions();
  } catch (loadError) {
    sessions.value = [];
    error.value = errorMessage(loadError, t("system.sessionsLoadFailed"));
  } finally {
    loading.value = false;
  }
}

async function revokeSession(session: OperatorSession) {
  if (session.current) return;
  revokingSessionId.value = session.id;
  try {
    await authApi.revokeSession(session.id);
    message.success(t("system.sessionRevoked"));
    await loadSessions();
  } catch (loadError) {
    message.error(errorMessage(loadError, t("system.sessionRevokeFailed")));
  } finally {
    revokingSessionId.value = "";
  }
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
  vertical-align: middle;
}

.system-table th {
  color: var(--tt-muted);
  font-weight: 720;
}

.system-table tbody tr:last-child td {
  border-bottom: 0;
}

.session-id {
  font-family: var(--font-mono, ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace);
  color: var(--tt-muted);
}
</style>
