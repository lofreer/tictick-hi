<template>
  <section class="page">
    <header class="page-header">
      <div>
        <h1 class="page-title">{{ t("page.auditEvents.title") }}</h1>
        <p class="page-subtitle">{{ t("system.auditEventsSubtitle") }}</p>
      </div>
      <NButton secondary @click="loadEvents">
        <template #icon><RefreshCw :size="17" /></template>
        {{ t("common.retry") }}
      </NButton>
    </header>

    <section class="surface system-panel">
      <LoadingState v-if="loading" />
      <ErrorState v-else-if="error" :title="error" retryable @retry="loadEvents" />
      <EmptyState v-else-if="events.length === 0" :title="t('system.noAuditEvents')" />
      <div v-else class="system-table-wrap">
        <table class="system-table">
          <thead>
            <tr>
              <th>{{ t("system.auditTime") }}</th>
              <th>{{ t("system.actor") }}</th>
              <th>{{ t("system.action") }}</th>
              <th>{{ t("system.resource") }}</th>
              <th>{{ t("system.outcome") }}</th>
              <th>{{ t("system.request") }}</th>
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
              <td><span class="audit-muted">{{ metadataText(event.metadata) }}</span></td>
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
import { systemApi } from "@/services/api/system";
import type { AuditEvent } from "@/types/app";

const { t } = useI18n();
const message = useMessage();
const events = ref<AuditEvent[]>([]);
const loading = ref(false);
const error = ref("");

onMounted(() => {
  void loadEvents();
});

async function loadEvents() {
  loading.value = true;
  error.value = "";
  try {
    events.value = await systemApi.listAuditEvents(100);
  } catch (loadError) {
    events.value = [];
    error.value = errorMessage(loadError, t("system.auditEventsLoadFailed"));
    message.error(error.value);
  } finally {
    loading.value = false;
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
  min-width: 1120px;
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
</style>
