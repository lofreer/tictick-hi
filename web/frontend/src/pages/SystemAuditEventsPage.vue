<template>
  <section class="page">
    <header class="page-header">
      <div>
        <h1 class="page-title">{{ t("page.auditEvents.title") }}</h1>
        <p class="page-subtitle">{{ t("system.auditEventsSubtitle") }}</p>
      </div>
      <NSpace size="small">
        <NButton secondary :loading="verifyingHashChain" @click="verifyHashChain">
          <template #icon><ShieldCheck :size="17" /></template>
          {{ t("system.verifyAuditHashChain") }}
        </NButton>
        <NButton tag="a" href="/api/system/audit-events/export?limit=100" secondary>
          <template #icon><Download :size="17" /></template>
          {{ t("system.exportAuditEvents") }}
        </NButton>
        <NButton secondary @click="loadEvents">
          <template #icon><RefreshCw :size="17" /></template>
          {{ t("common.retry") }}
        </NButton>
      </NSpace>
    </header>

    <section class="surface system-panel">
      <LoadingState v-if="loading" />
      <ErrorState v-else-if="error" :title="error" retryable @retry="loadEvents" />
      <EmptyState v-else-if="events.length === 0" :title="t('system.noAuditEvents')" />
      <div v-else>
        <NAlert v-if="hashVerification" class="audit-verification" :type="hashVerificationType" :bordered="false">
          {{ t("system.auditHashVerification") }}: {{ hashVerification.message }}
          <span class="audit-muted">
            {{ t("system.auditHashChecked") }} {{ hashVerification.checkedCount }} /
            {{ t("system.auditHashSkipped") }} {{ hashVerification.skippedCount }}
          </span>
        </NAlert>
        <div class="system-table-wrap">
          <table class="system-table">
            <thead>
              <tr>
                <th>{{ t("system.auditTime") }}</th>
                <th>{{ t("system.actor") }}</th>
                <th>{{ t("system.action") }}</th>
                <th>{{ t("system.resource") }}</th>
                <th>{{ t("system.outcome") }}</th>
                <th>{{ t("system.request") }}</th>
                <th>{{ t("system.auditHash") }}</th>
                <th>{{ t("system.metadata") }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="event in events" :key="event.id">
                <td>{{ formatDate(event.createdAt) }}</td>
                <td>{{ event.actorUsername || "-" }}</td>
                <td><span class="audit-code">{{ event.action }}</span></td>
                <td>
                  <span class="audit-code">{{ event.resourceType }}</span>
                  <span v-if="event.resourceId" class="audit-muted"> / {{ event.resourceId }}</span>
                </td>
                <td>
                  <NTag :type="event.outcome === 'success' ? 'success' : 'error'" size="small">
                    {{ outcomeLabel(event.outcome) }}
                  </NTag>
                </td>
                <td>
                  <span class="audit-code">{{ event.requestMethod || "-" }}</span>
                  <span class="audit-muted"> {{ event.requestPath || "" }}</span>
                </td>
                <td>
                  <span class="audit-code">{{ shortHash(event.eventHash) }}</span>
                  <span v-if="event.previousHash" class="audit-muted audit-hash-prev">
                    {{ t("system.previousHashShort") }} {{ shortHash(event.previousHash) }}
                  </span>
                </td>
                <td><span class="audit-muted">{{ metadataText(event.metadata) }}</span></td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="nextCursor" class="audit-pagination">
          <NButton secondary :loading="loadingMore" @click="loadOlderEvents">
            <template #icon><ChevronDown :size="17" /></template>
            {{ t("system.loadOlderAuditEvents") }}
          </NButton>
        </div>
      </div>
    </section>
  </section>
</template>

<script setup lang="ts">
import { ChevronDown, Download, RefreshCw, ShieldCheck } from "@lucide/vue";
import { NAlert, NButton, NSpace, NTag, useMessage } from "naive-ui";
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";

import EmptyState from "@/components/common/EmptyState.vue";
import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import { systemApi } from "@/services/api/system";
import type { AuditEvent, AuditEventHashChainVerification } from "@/types/app";

const { t } = useI18n();
const message = useMessage();
const events = ref<AuditEvent[]>([]);
const loading = ref(false);
const loadingMore = ref(false);
const verifyingHashChain = ref(false);
const error = ref("");
const nextCursor = ref("");
const hashVerification = ref<AuditEventHashChainVerification | null>(null);

const hashVerificationType = computed(() => {
  if (!hashVerification.value) return "info";
  if (hashVerification.value.status === "failure") return "error";
  if (hashVerification.value.status === "warning") return "warning";
  return "success";
});

onMounted(() => {
  void loadEvents();
});

async function loadEvents() {
  loading.value = true;
  error.value = "";
  nextCursor.value = "";
  try {
    const page = await systemApi.listAuditEventPage(100);
    events.value = page.events;
    nextCursor.value = page.nextCursor ?? "";
  } catch (loadError) {
    events.value = [];
    error.value = errorMessage(loadError, t("system.auditEventsLoadFailed"));
    message.error(error.value);
  } finally {
    loading.value = false;
  }
}

async function loadOlderEvents() {
  if (loadingMore.value || nextCursor.value === "") return;
  loadingMore.value = true;
  try {
    const page = await systemApi.listAuditEventPage(100, nextCursor.value);
    events.value = [...events.value, ...page.events];
    nextCursor.value = page.nextCursor ?? "";
  } catch (loadError) {
    const loadMoreError = errorMessage(loadError, t("system.auditEventsLoadMoreFailed"));
    message.error(loadMoreError);
  } finally {
    loadingMore.value = false;
  }
}

async function verifyHashChain() {
  if (verifyingHashChain.value) return;
  verifyingHashChain.value = true;
  try {
    hashVerification.value = await systemApi.verifyAuditEventHashChain();
  } catch (verifyError) {
    const messageText = errorMessage(verifyError, t("system.auditHashVerificationFailed"));
    message.error(messageText);
  } finally {
    verifyingHashChain.value = false;
  }
}

function metadataText(metadata: Record<string, string>) {
  const entries = Object.entries(metadata);
  if (entries.length === 0) return "-";
  return entries.map(([key, value]) => `${key}=${value}`).join(", ");
}

function outcomeLabel(outcome: string) {
  return outcome === "success" ? t("system.outcomeSuccess") : t("system.outcomeFailure");
}

function formatDate(value?: string) {
  return value ? new Date(value).toLocaleString() : "-";
}

function shortHash(value?: string) {
  return value ? value.slice(0, 12) : "-";
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
  min-width: 1240px;
  border-collapse: collapse;
}

.system-table th,
.system-table td {
  padding: 12px 14px;
  border-bottom: 1px solid var(--tt-line);
  font-size: 13px;
  line-height: 1.5;
  text-align: left;
  vertical-align: top;
}

.system-table th {
  color: var(--tt-muted);
  font-weight: 720;
}

.system-table tbody tr:last-child td {
  border-bottom: 0;
}

.audit-code {
  font-family: var(--font-mono, ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace);
}

.audit-muted {
  color: var(--tt-muted);
}

.audit-verification {
  margin: 0;
  border-bottom: 1px solid var(--tt-line);
}

.audit-hash-prev {
  display: block;
  margin-top: 2px;
}

.audit-pagination {
  display: flex;
  justify-content: center;
  padding: 14px;
  border-top: 1px solid var(--tt-line);
}
</style>
