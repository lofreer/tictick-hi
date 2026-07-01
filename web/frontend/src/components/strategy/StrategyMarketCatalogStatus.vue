<template>
  <div class="strategy-market-catalog-status">
    <NTag size="small" :type="tagType" round>
      {{ label }}
    </NTag>
    <p v-if="hint" class="strategy-market-catalog-status__hint" :data-status="status">
      {{ hint }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { NTag, type TagProps } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import type { MarketCatalogValidationStatus } from "@/composables/useMarketCatalogValidation";
import {
  marketStatusBaseLabel,
  marketStatusExchangeDetail,
} from "@/utils/marketStatusDisplay";

const props = defineProps<{
  detail: string;
  error: string;
  loading: boolean;
  status: MarketCatalogValidationStatus;
}>();

const { t } = useI18n();

const label = computed(() => marketCatalogStatusLabel(t, props.status, props.loading, props.error));
const exchangeDetail = computed(() => marketStatusExchangeDetail(t, props.detail));
const hint = computed(() => {
  if (props.error) return props.error;
  if (props.status === "inactive") return [t("research.instrumentInactive"), exchangeDetail.value].filter(Boolean).join(" ");
  if (props.status === "missing") return t("research.instrumentNotInCatalog");
  if (props.status === "active") return exchangeDetail.value;
  return "";
});

const tagType = computed<TagProps["type"]>(() => {
  if (props.status === "active") return "success";
  if (props.status === "inactive" || props.status === "missing") return "warning";
  if (props.error) return "error";
  return "default";
});

function marketCatalogStatusLabel(
  t: (key: string) => string,
  status: MarketCatalogValidationStatus,
  loading: boolean,
  error: string,
) {
  if (loading) return t("strategy.marketCatalogChecking");
  if (status === "active" || status === "inactive" || status === "missing") return marketStatusBaseLabel(t, status);
  if (error) return t("strategy.marketCatalogError");
  return t("strategy.marketCatalogUnknown");
}
</script>

<style scoped>
.strategy-market-catalog-status {
  display: grid;
  min-width: 0;
  justify-items: end;
  gap: 4px;
}

.strategy-market-catalog-status__hint {
  max-width: min(360px, 60vw);
  margin: 0;
  color: var(--tt-muted);
  font-size: 12px;
  line-height: 1.45;
  text-align: right;
}

.strategy-market-catalog-status__hint[data-status="inactive"],
.strategy-market-catalog-status__hint[data-status="missing"] {
  color: var(--tt-warning);
}
</style>
