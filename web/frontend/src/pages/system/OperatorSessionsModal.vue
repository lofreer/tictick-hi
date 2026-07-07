<template>
  <NModal :show="show" preset="card" :title="title" class="system-modal system-sessions-modal" @update:show="emit('update:show', $event)">
    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" :title="error" retryable @retry="emit('retry')" />
    <EmptyState v-else-if="sessions.length === 0" :title="t('system.noSessions')" />
    <div v-else class="system-table-wrap">
      <table class="system-table">
        <thead>
          <tr>
            <th>{{ t("system.sessionId") }}</th>
            <th>{{ t("system.status") }}</th>
            <th>{{ t("system.remoteAddr") }}</th>
            <th>{{ t("system.userAgent") }}</th>
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
            <td>{{ emptyText(session.remoteAddr) }}</td>
            <td>
              <span class="session-user-agent" :title="session.userAgent || undefined">
                {{ emptyText(session.userAgent) }}
              </span>
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
                :title="session.current ? t('system.currentOperatorSessionRevokeBlocked') : undefined"
                @click="emit('revoke', session)"
              >
                <template #icon><LogOut :size="16" /></template>
                {{ t("system.revokeSession") }}
              </NButton>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <template #footer>
      <NSpace justify="end">
        <NButton secondary @click="emit('retry')">{{ t("common.retry") }}</NButton>
        <NButton @click="emit('update:show', false)">{{ t("common.close") }}</NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<script setup lang="ts">
import { LogOut } from "@lucide/vue";
import { NButton, NModal, NSpace, NTag } from "naive-ui";
import { useI18n } from "vue-i18n";

import EmptyState from "@/components/common/EmptyState.vue";
import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import type { OperatorSession } from "@/types/app";

defineProps<{
  error: string;
  loading: boolean;
  revokingSessionId: string;
  sessions: OperatorSession[];
  show: boolean;
  title: string;
}>();

const emit = defineEmits<{
  (event: "retry"): void;
  (event: "revoke", session: OperatorSession): void;
  (event: "update:show", value: boolean): void;
}>();

const { t } = useI18n();

function formatDate(value?: string) {
  return value ? new Date(value).toLocaleString() : "-";
}

function emptyText(value?: string) {
  return value || "-";
}
</script>

<style scoped>
.system-table-wrap {
  overflow-x: auto;
}

.system-table {
  width: 100%;
  min-width: 860px;
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

.session-id,
.session-user-agent {
  display: inline-block;
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}

:global(.system-modal) {
  width: min(560px, calc(100vw - 32px));
}

:global(.system-sessions-modal) {
  width: min(960px, calc(100vw - 32px));
}
</style>
