<template>
  <section class="page">
    <header class="page-header">
      <div>
        <h1 class="page-title">{{ t(titleKey) }}</h1>
        <p class="page-subtitle">{{ t(subtitleKey) }}</p>
      </div>
      <NButton type="primary" :loading="submitLoading" :disabled="!canSubmit" @click="submit">
        <template #icon>
          <FlaskConical v-if="isBacktest" :size="17" />
          <Play v-else :size="17" />
        </template>
        {{ t(ctaKey) }}
      </NButton>
    </header>

    <div class="task-form-grid">
      <section class="surface task-form-panel">
        <NForm label-placement="top">
          <section class="task-form-section">
            <div class="task-section-heading">
              <h2 class="task-section-title">{{ t("strategy.market") }}</h2>
              <StrategyMarketCatalogStatus
                :detail="marketCatalogStatusDetail"
                :error="marketCatalogError"
                :loading="marketCatalogLoading"
                :status="marketCatalogStatus"
              />
            </div>
            <div class="task-field-grid">
              <NFormItem class="task-field--wide" :label="t('strategy.taskName')">
                <NInput v-model:value="form.name" class="task-control" />
              </NFormItem>
              <NFormItem :label="t('research.exchange')">
                <NSelect v-model:value="form.exchange" class="task-control" :options="exchangeOptions" />
              </NFormItem>
              <NFormItem :label="t('research.symbol')">
                <MarketSymbolAutoComplete v-model:value="form.symbol" class="task-control" :exchange="form.exchange" />
              </NFormItem>
              <NFormItem :label="t('research.interval')">
                <NSelect v-model:value="form.interval" class="task-control" :options="intervalOptions" />
              </NFormItem>
            </div>
          </section>

          <section class="task-form-section">
            <h2 class="task-section-title">
              {{ t(isBacktest ? "strategy.backtestSettings" : "strategy.tradingSettings") }}
            </h2>
            <div class="task-field-grid">
              <template v-if="isBacktest">
                <NFormItem :label="t('research.startTime')">
                  <NDatePicker v-model:value="form.startTime" class="task-control" type="datetime" clearable />
                </NFormItem>
                <NFormItem :label="t('research.endTime')">
                  <NDatePicker v-model:value="form.endTime" class="task-control" type="datetime" clearable />
                </NFormItem>
                <NFormItem :label="t('strategy.initialBalance')">
                  <NInputNumber
                    v-model:value="form.initialBalance"
                    class="task-control"
                    :min="0"
                    :step="100"
                  />
                </NFormItem>
                <NFormItem :label="t('strategy.feeBps')">
                  <NInputNumber
                    v-model:value="form.feeBps"
                    class="task-control"
                    :min="0"
                    :step="0.1"
                  />
                </NFormItem>
                <NFormItem :label="t('strategy.slippageBps')">
                  <NInputNumber
                    v-model:value="form.slippageBps"
                    class="task-control"
                    :min="0"
                    :step="0.1"
                  />
                </NFormItem>
                <NFormItem :label="t('strategy.triggerMode')">
                  <NSelect v-model:value="form.triggerMode" class="task-control" :options="triggerModeOptions" />
                </NFormItem>
              </template>
              <template v-else>
                <NFormItem :label="t('strategy.executionMode')">
                  <NSelect v-model:value="form.executionMode" class="task-control" :options="executionModeOptions" />
                </NFormItem>
                <NFormItem :label="t('trading.accountId')">
                  <NInput v-model:value="form.accountId" class="task-control" />
                </NFormItem>
                <NFormItem :label="t('strategy.riskLimitPct')">
                  <NInputNumber
                    v-model:value="form.riskLimitPct"
                    class="task-control"
                    :min="0"
                    :max="100"
                    :step="0.5"
                  />
                </NFormItem>
                <NFormItem :label="t('trading.orderIntent')">
                  <NSelect v-model:value="form.orderIntent" class="task-control" :options="orderIntentOptions" />
                </NFormItem>
                <NFormItem :label="t('trading.notificationChannel')">
                  <NInput v-model:value="form.notificationChannel" class="task-control" />
                </NFormItem>
                <NFormItem v-if="form.executionMode === 'live'" :label="t('trading.liveConfirm')">
                  <NInput
                    v-model:value="form.liveConfirmation"
                    class="task-control"
                    :placeholder="t('trading.liveConfirmPlaceholder')"
                  />
                </NFormItem>
              </template>
            </div>
          </section>

          <section class="task-form-section">
            <div class="task-section-heading">
              <h2 class="task-section-title">{{ t("strategy.strategy") }}</h2>
              <NTag v-if="selectedStrategy" size="small" round>{{ selectedStrategy.version }}</NTag>
            </div>
            <NFormItem :label="t('strategy.selectedStrategy')">
              <NSelect
                v-model:value="selectedStrategyId"
                class="task-control"
                :loading="loading"
                :options="strategyOptions"
              />
            </NFormItem>

            <ErrorState v-if="error" :title="error" retryable @retry="loadStrategies" />
            <LoadingState v-else-if="loading" />
            <StrategyParamForm
              v-else-if="selectedStrategy"
              v-model:value="paramValues"
              :params="selectedStrategy.params"
            />
            <EmptyState v-else :title="t('strategy.noStrategies')" />
          </section>
        </NForm>
      </section>

      <aside class="side-panel">
        <section class="surface task-summary">
          <h2 class="task-section-title">{{ t("strategy.summary") }}</h2>
          <dl class="task-summary__list">
            <template v-for="row in summaryRows" :key="row.label">
              <dt>{{ row.label }}</dt>
              <dd>{{ row.value }}</dd>
            </template>
          </dl>
          <div v-if="selectedStrategy" class="task-summary__strategy">
            <NText depth="3">{{ selectedStrategy.description }}</NText>
            <div class="task-summary__tags">
              <NTag v-for="intent in selectedStrategy.supportedIntents" :key="intent" size="small">
                {{ intent }}
              </NTag>
            </div>
          </div>
          <dl v-if="paramRows.length > 0" class="task-summary__list">
            <template v-for="row in paramRows" :key="row.key">
              <dt>{{ row.label }}</dt>
              <dd>{{ row.value }}</dd>
            </template>
          </dl>
        </section>
      </aside>
    </div>
  </section>
</template>

<script setup lang="ts">
import { FlaskConical, Play } from "@lucide/vue";
import {
  NButton,
  NDatePicker,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSelect,
  NTag,
  NText,
  type SelectOption,
} from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import EmptyState from "@/components/common/EmptyState.vue";
import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import MarketSymbolAutoComplete from "@/components/market/MarketSymbolAutoComplete.vue";
import StrategyMarketCatalogStatus from "@/components/strategy/StrategyMarketCatalogStatus.vue";
import StrategyParamForm from "@/components/strategy/StrategyParamForm.vue";
import { useStrategyTaskForm, type StrategyTaskMode } from "@/composables/useStrategyTaskForm";
import type { StrategyParamValue } from "@/types/app";
import { marketStatusBaseLabel } from "@/utils/marketStatusDisplay";

const props = defineProps<{
  mode: StrategyTaskMode;
}>();

const { t } = useI18n();
const {
  canSubmit,
  error,
  form,
  intervalOptions,
  loadStrategies,
  loading,
  marketCatalogError,
  marketCatalogLoading,
  marketCatalogStatus,
  marketCatalogStatusDetail,
  paramValues,
  selectedStrategy,
  selectedStrategyId,
  strategyOptions,
  submit,
  submitLoading,
} = useStrategyTaskForm(props.mode);

const isBacktest = computed(() => props.mode === "backtest");
const titleKey = computed(() => (isBacktest.value ? "page.backtestsNew.title" : "page.tradingNew.title"));
const subtitleKey = computed(() =>
  isBacktest.value ? "page.backtestsNew.subtitle" : "page.tradingNew.subtitle",
);
const ctaKey = computed(() => (isBacktest.value ? "strategy.createBacktest" : "strategy.createTradingTask"));

const exchangeOptions = computed<SelectOption[]>(() => [
  { label: "Binance", value: "binance" },
  { label: "OKX", value: "okx" },
]);

const executionModeOptions = computed<SelectOption[]>(() => [
  { label: t("strategy.paper"), value: "paper" },
  { label: t("strategy.live"), value: "live" },
]);

const orderIntentOptions = computed<SelectOption[]>(() => [
  { label: t("trading.orderIntentExecute"), value: "execute", disabled: form.executionMode === "live" },
  { label: t("trading.orderIntentNotify"), value: "notify" },
]);

const triggerModeOptions = computed<SelectOption[]>(() => [
  { label: t("strategy.closedCandle"), value: "closed_candle" },
  { label: t("strategy.minuteReplay"), value: "minute_replay" },
]);

const marketCatalogLabel = computed(() => {
  if (marketCatalogLoading.value) return t("strategy.marketCatalogChecking");
  if (
    marketCatalogStatus.value === "active" ||
    marketCatalogStatus.value === "inactive" ||
    marketCatalogStatus.value === "missing"
  ) {
    return marketStatusBaseLabel(t, marketCatalogStatus.value);
  }
  if (marketCatalogError.value) return t("strategy.marketCatalogError");
  return t("strategy.marketCatalogUnknown");
});

const summaryRows = computed(() => {
  const rows = [
    ...(isBacktest.value ? [{ label: t("strategy.taskName"), value: form.name }] : []),
    { label: t("research.exchange"), value: form.exchange },
    { label: t("research.symbol"), value: form.symbol },
    { label: t("research.interval"), value: form.interval },
    { label: t("research.marketStatus"), value: marketCatalogLabel.value },
    { label: t("strategy.selectedStrategy"), value: selectedStrategy.value?.name ?? "-" },
  ];

  if (isBacktest.value) {
    rows.push(
      { label: t("research.startTime"), value: formatDate(form.startTime) },
      { label: t("research.endTime"), value: formatDate(form.endTime) },
      { label: t("strategy.initialBalance"), value: String(form.initialBalance) },
      { label: t("strategy.feeBps"), value: String(form.feeBps) },
      { label: t("strategy.slippageBps"), value: String(form.slippageBps) },
      { label: t("strategy.triggerMode"), value: triggerModeLabel(form.triggerMode) },
    );
  } else {
    rows.push(
      { label: t("strategy.executionMode"), value: t(`strategy.${form.executionMode}`) },
      { label: t("trading.accountId"), value: form.accountId },
      { label: t("strategy.riskLimitPct"), value: `${form.riskLimitPct}%` },
      { label: t("trading.orderIntent"), value: orderIntentLabel(form.orderIntent) },
      { label: t("trading.notificationChannel"), value: form.notificationChannel },
    );
  }

  return rows;
});

const paramRows = computed(() =>
  (selectedStrategy.value?.params ?? []).map((param) => ({
    key: param.key,
    label: param.label,
    value: formatParamValue(paramValues.value[param.key]),
  })),
);

function formatDate(value: number | null) {
  return value === null ? "-" : new Date(value).toLocaleString();
}

function formatParamValue(value: StrategyParamValue | undefined) {
  if (value === undefined || value === null || value === "") {
    return "-";
  }
  if (typeof value === "boolean") {
    return value ? t("common.yes") : t("common.no");
  }
  return String(value);
}

function triggerModeLabel(value: string) {
  return value === "minute_replay" ? t("strategy.minuteReplay") : t("strategy.closedCandle");
}

function orderIntentLabel(value: string) {
  return value === "execute" ? t("trading.orderIntentExecute") : t("trading.orderIntentNotify");
}
</script>

<style scoped>
.task-form-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 340px;
  gap: 16px;
  align-items: start;
}

.task-form-panel,
.task-summary {
  padding: 16px;
}

.task-form-section + .task-form-section {
  margin-top: 18px;
  padding-top: 18px;
  border-top: 1px solid var(--tt-line);
}

.task-section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.task-section-title {
  margin: 0;
  font-size: 16px;
  font-weight: 720;
  line-height: 1.35;
}

.task-field-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(150px, 1fr));
  gap: 0 14px;
}

.task-control {
  width: 100%;
}

.task-field--wide {
  grid-column: 1 / -1;
}

.task-summary {
  position: sticky;
  top: 88px;
}

.task-summary__list {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.4fr);
  gap: 10px 12px;
  margin: 12px 0 0;
}

.task-summary__list dt,
.task-summary__list dd {
  min-width: 0;
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
}

.task-summary__list dt {
  color: var(--tt-muted);
}

.task-summary__list dd {
  overflow-wrap: anywhere;
  font-weight: 650;
  text-align: right;
}

.task-summary__strategy {
  display: grid;
  gap: 10px;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--tt-line);
}

.task-summary__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

@media (max-width: 1080px) {
  .task-form-grid {
    grid-template-columns: 1fr;
  }

  .task-summary {
    position: static;
  }
}

@media (max-width: 760px) {
  .task-field-grid {
    grid-template-columns: 1fr;
  }
}
</style>
