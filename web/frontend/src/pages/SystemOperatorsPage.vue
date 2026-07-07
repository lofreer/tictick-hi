<template>
  <section class="page">
    <header class="page-header">
      <div>
        <h1 class="page-title">{{ t("page.operators.title") }}</h1>
        <p class="page-subtitle">{{ t("system.operatorsSubtitle") }}</p>
      </div>
      <NButton v-if="canManageOperators" type="primary" @click="createOpen = true">
        <template #icon><Plus :size="17" /></template>
        {{ t("system.createOperator") }}
      </NButton>
    </header>

    <section class="surface system-panel">
      <LoadingState v-if="loading" />
      <ErrorState v-else-if="error" :title="error" retryable @retry="loadOperators" />
      <EmptyState v-else-if="operators.length === 0" :title="t('system.noOperators')" />
      <div v-else class="system-table-wrap">
        <table class="system-table">
          <thead>
            <tr>
              <th>{{ t("auth.username") }}</th>
              <th>{{ t("system.operatorRole") }}</th>
              <th>{{ t("system.enabled") }}</th>
              <th>{{ t("backtests.createdAt") }}</th>
              <th v-if="canManageOperators">{{ t("research.actions") }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="operator in operators" :key="operator.id">
              <td>{{ operator.username }}</td>
              <td>
                <NTag :type="operator.role === 'admin' ? 'warning' : 'info'" size="small">
                  {{ operatorRoleLabel(operator.role) }}
                </NTag>
              </td>
              <td><NTag :type="operator.enabled ? 'success' : 'default'" size="small">{{ enabledLabel(operator.enabled) }}</NTag></td>
              <td>{{ formatDate(operator.createdAt) }}</td>
              <td v-if="canManageOperators">
                <NSpace size="small" :wrap="false">
                  <NButton
                    size="small"
                    secondary
                    :disabled="operatorSelfRoleBlocked(operator)"
                    :title="operatorSelfRoleBlocked(operator) ? t('system.currentOperatorRoleBlocked') : undefined"
                    @click="openRoleModal(operator)"
                  >
                    <template #icon><ShieldCheck :size="16" /></template>
                    {{ t("system.editOperatorRole") }}
                  </NButton>
                  <NButton
                    size="small"
                    :type="operator.enabled ? 'warning' : 'primary'"
                    secondary
                    :disabled="operatorSelfDisableBlocked(operator)"
                    :loading="updatingOperatorId === operator.id"
                    :title="operatorSelfDisableBlocked(operator) ? t('system.currentOperatorDisableBlocked') : undefined"
                    @click="toggleOperator(operator)"
                  >
                    {{ operator.enabled ? t("system.disableOperator") : t("system.enableOperator") }}
                  </NButton>
                </NSpace>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <NModal v-model:show="createOpen" preset="card" :title="t('system.createOperator')" class="system-modal">
      <NForm label-placement="top">
        <NFormItem :label="t('auth.username')"><NInput v-model:value="form.username" /></NFormItem>
        <NFormItem :label="t('auth.password')"><NInput v-model:value="form.password" type="password" /></NFormItem>
        <NFormItem :label="t('system.operatorRole')"><NSelect v-model:value="form.role" :options="roleOptions" /></NFormItem>
        <NFormItem :label="t('system.enabled')"><NSwitch v-model:value="form.enabled" /></NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="createOpen = false">{{ t("common.cancel") }}</NButton>
          <NButton type="primary" :loading="creating" @click="createOperator">{{ t("common.create") }}</NButton>
        </NSpace>
      </template>
    </NModal>

    <NModal v-model:show="roleOpen" preset="card" :title="t('system.updateOperatorRole')" class="system-modal">
      <NForm label-placement="top">
        <NFormItem :label="t('auth.username')"><NInput :value="roleOperator?.username ?? ''" disabled /></NFormItem>
        <NFormItem :label="t('system.operatorRole')"><NSelect v-model:value="roleForm.role" :options="roleOptions" /></NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="roleOpen = false">{{ t("common.cancel") }}</NButton>
          <NButton type="primary" :loading="roleUpdating" @click="updateOperatorRole">{{ t("common.save") }}</NButton>
        </NSpace>
      </template>
    </NModal>
  </section>
</template>

<script setup lang="ts">
import { Plus, ShieldCheck } from "@lucide/vue";
import { NButton, NForm, NFormItem, NInput, NModal, NSelect, NSpace, NSwitch, NTag, useMessage } from "naive-ui";
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";

import EmptyState from "@/components/common/EmptyState.vue";
import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import { systemApi } from "@/services/api/system";
import { useAuthStore } from "@/stores/auth";
import type { Operator } from "@/types/app";

const { t } = useI18n();
const message = useMessage();
const authStore = useAuthStore();
const operators = ref<Operator[]>([]);
const loading = ref(false);
const creating = ref(false);
const error = ref("");
const updatingOperatorId = ref("");
const createOpen = ref(false);
const roleOpen = ref(false);
const roleUpdating = ref(false);
const roleOperator = ref<Operator | null>(null);
const form = reactive({ username: "", password: "", role: "operator", enabled: true });
const roleForm = reactive({ role: "operator" });
const roleOptions = computed(() => [
  { label: t("system.operatorRoleOperator"), value: "operator" },
  { label: t("system.operatorRoleAdmin"), value: "admin" },
]);
const canManageOperators = computed(() => authStore.operator?.role === "admin");

onMounted(() => {
  void loadOperators();
});

async function loadOperators() {
  loading.value = true;
  error.value = "";
  try {
    operators.value = await systemApi.listOperators();
  } catch (loadError) {
    operators.value = [];
    error.value = errorMessage(loadError, t("system.operatorsLoadFailed"));
  } finally {
    loading.value = false;
  }
}

async function createOperator() {
  creating.value = true;
  try {
    await systemApi.createOperator({ ...form });
    createOpen.value = false;
    message.success(t("system.created"));
    form.username = "";
    form.password = "";
    form.role = "operator";
    await loadOperators();
  } catch (loadError) {
    message.error(errorMessage(loadError, t("system.createFailed")));
  } finally {
    creating.value = false;
  }
}

function openRoleModal(operator: Operator) {
  if (operatorSelfRoleBlocked(operator)) {
    return;
  }
  roleOperator.value = operator;
  roleForm.role = operator.role === "admin" ? "admin" : "operator";
  roleOpen.value = true;
}

async function updateOperatorRole() {
  if (!roleOperator.value) {
    return;
  }
  roleUpdating.value = true;
  try {
    await systemApi.setOperatorRole(roleOperator.value.id, { role: roleForm.role });
    roleOpen.value = false;
    roleOperator.value = null;
    message.success(t("system.operatorUpdated"));
    await loadOperators();
  } catch (loadError) {
    message.error(errorMessage(loadError, t("system.operatorUpdateFailed")));
  } finally {
    roleUpdating.value = false;
  }
}

async function toggleOperator(operator: Operator) {
  if (operatorSelfDisableBlocked(operator)) {
    return;
  }
  updatingOperatorId.value = operator.id;
  try {
    await systemApi.setOperatorEnabled(operator.id, !operator.enabled);
    message.success(t("system.operatorUpdated"));
    await loadOperators();
  } catch (loadError) {
    message.error(errorMessage(loadError, t("system.operatorUpdateFailed")));
  } finally {
    updatingOperatorId.value = "";
  }
}

function operatorSelfDisableBlocked(operator: Operator) {
  return operator.enabled && authStore.operator?.id === operator.id;
}

function operatorSelfRoleBlocked(operator: Operator) {
  return authStore.operator?.id === operator.id;
}

function enabledLabel(enabled: boolean) {
  return enabled ? t("common.yes") : t("common.no");
}

function operatorRoleLabel(role: string) {
  return role === "admin" ? t("system.operatorRoleAdmin") : t("system.operatorRoleOperator");
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
