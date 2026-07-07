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

    <section class="surface system-panel password-panel">
      <div class="panel-heading">
        <div>
          <h2 class="section-title">{{ t("system.changePassword") }}</h2>
          <p class="section-subtitle">{{ t("system.changePasswordSubtitle") }}</p>
        </div>
      </div>
      <NForm class="password-form" :show-feedback="false" @submit.prevent="changePassword">
        <div class="password-grid">
          <NFormItem :label="t('system.currentPassword')">
            <NInput v-model:value="passwordForm.currentPassword" type="password" show-password-on="mousedown" />
          </NFormItem>
          <NFormItem :label="t('system.newPassword')">
            <NInput v-model:value="passwordForm.newPassword" type="password" show-password-on="mousedown" />
          </NFormItem>
          <NFormItem :label="t('system.confirmNewPassword')">
            <NInput v-model:value="passwordForm.confirmPassword" type="password" show-password-on="mousedown" />
          </NFormItem>
        </div>
        <div class="password-actions">
          <NButton type="primary" attr-type="submit" :loading="changingPassword">
            <template #icon><KeyRound :size="16" /></template>
            {{ t("system.changePassword") }}
          </NButton>
        </div>
      </NForm>
    </section>

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
              <td>
                <div class="session-context-cell">
                  <span class="session-remote-addr">{{ emptyText(session.remoteAddr) }}</span>
                  <NTag v-if="session.remoteAddrChanged" type="warning" size="small">
                    {{ t("system.contextChanged") }}
                  </NTag>
                </div>
              </td>
              <td>
                <div class="session-context-cell">
                  <span class="session-user-agent" :title="session.userAgent || undefined">
                    {{ emptyText(session.userAgent) }}
                  </span>
                  <NTag v-if="session.userAgentChanged" type="warning" size="small">
                    {{ t("system.contextChanged") }}
                  </NTag>
                </div>
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
import { KeyRound, RefreshCw } from "@lucide/vue";
import { NButton, NForm, NFormItem, NInput, NTag, useMessage } from "naive-ui";
import { onMounted, reactive, ref } from "vue";
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
const changingPassword = ref(false);
const passwordForm = reactive({ currentPassword: "", newPassword: "", confirmPassword: "" });

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

async function changePassword() {
  if (
    passwordForm.currentPassword.trim() === "" ||
    passwordForm.newPassword.trim() === "" ||
    passwordForm.confirmPassword.trim() === ""
  ) {
    message.error(t("system.passwordChangeRequired"));
    return;
  }
  if (passwordForm.newPassword !== passwordForm.confirmPassword) {
    message.error(t("system.passwordConfirmMismatch"));
    return;
  }
  changingPassword.value = true;
  try {
    await authApi.changePassword({
      currentPassword: passwordForm.currentPassword,
      newPassword: passwordForm.newPassword,
    });
    resetPasswordForm();
    message.success(t("system.passwordChanged"));
    await loadSessions();
  } catch (loadError) {
    message.error(errorMessage(loadError, t("system.passwordChangeFailed")));
  } finally {
    changingPassword.value = false;
  }
}

function resetPasswordForm() {
  passwordForm.currentPassword = "";
  passwordForm.newPassword = "";
  passwordForm.confirmPassword = "";
}

function formatDate(value?: string) {
  return value ? new Date(value).toLocaleString() : "-";
}

function emptyText(value?: string) {
  return value || "-";
}

function errorMessage(loadError: unknown, fallback: string) {
  return loadError instanceof Error && loadError.message ? loadError.message : fallback;
}
</script>

<style scoped>
.system-panel {
  overflow: hidden;
}

.password-panel {
  margin-bottom: 16px;
}

.panel-heading {
  margin-bottom: 14px;
}

.section-title {
  margin: 0;
  color: var(--tt-ink);
  font-size: 16px;
  font-weight: 760;
}

.section-subtitle {
  margin: 4px 0 0;
  color: var(--tt-muted);
  font-size: 13px;
  line-height: 1.5;
}

.password-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
  gap: 12px;
}

.password-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 2px;
}

.system-table-wrap {
  overflow-x: auto;
}

.system-table {
  width: 100%;
  min-width: 960px;
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

.session-remote-addr {
  font-family: var(--font-mono, ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace);
}

.session-context-cell {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 8px;
}

.session-user-agent {
  display: inline-block;
  max-width: 280px;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}
</style>
