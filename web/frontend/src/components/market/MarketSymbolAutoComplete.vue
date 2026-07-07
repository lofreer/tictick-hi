<template>
  <div class="market-symbol-autocomplete">
    <NAutoComplete
      class="market-symbol-autocomplete__input"
      :loading="loading"
      :options="options"
      :size="size"
      :value="value"
      clearable
      @update:value="handleUpdateValue"
    />
    <NButton
      v-if="canSyncInstruments"
      circle
      quaternary
      :size="size"
      :loading="syncing"
      :title="t('research.refreshInstruments')"
      @click="syncInstruments"
    >
      <template #icon>
        <RefreshCw :size="15" />
      </template>
    </NButton>
  </div>
</template>

<script setup lang="ts">
import { RefreshCw } from "@lucide/vue";
import { NAutoComplete, NButton } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

import { marketApi } from "@/services/api/market";
import { useAuthStore } from "@/stores/auth";
import type { MarketInstrument } from "@/types/app";
import { normalizeSymbolInput, symbolOptionsForExchange, type MarketSymbolOption } from "@/utils/marketSymbols";

const props = withDefaults(defineProps<{
  exchange: string;
  showSyncButton?: boolean;
  size?: "small" | "medium" | "large";
  value: string;
}>(), {
  showSyncButton: true,
  size: "medium",
});

const emit = defineEmits<{
  "update:value": [value: string];
  synced: [];
}>();

const { t } = useI18n();
const authStore = useAuthStore();
const loading = ref(false);
const syncing = ref(false);
const canSyncInstruments = computed(() => props.showSyncButton && authStore.operator?.role === "admin");
const options = ref<MarketSymbolOption[]>(symbolOptionsForExchange(props.exchange));
let requestSequence = 0;

watch(
  () => props.exchange,
  () => {
    options.value = symbolOptionsForExchange(props.exchange);
    void loadOptions(props.value);
  },
  { immediate: true },
);

async function loadOptions(query = "") {
  const sequence = ++requestSequence;
  loading.value = true;
  try {
    const instruments = await marketApi.listInstruments({
      exchange: props.exchange,
      limit: 20,
      q: normalizeSymbolInput(query),
    });
    if (sequence !== requestSequence) return;
    options.value = instruments.length > 0 ? instruments.map(instrumentOption) : symbolOptionsForExchange(props.exchange);
  } catch {
    if (sequence === requestSequence) {
      options.value = symbolOptionsForExchange(props.exchange);
    }
  } finally {
    if (sequence === requestSequence) {
      loading.value = false;
    }
  }
}

function handleUpdateValue(nextValue: string) {
  emit("update:value", nextValue);
  void loadOptions(nextValue);
}

async function syncInstruments() {
  if (!canSyncInstruments.value) {
    return;
  }
  syncing.value = true;
  try {
    await marketApi.syncInstruments(props.exchange);
    await loadOptions(props.value);
  } catch {
    options.value = symbolOptionsForExchange(props.exchange);
  } finally {
    syncing.value = false;
    emit("synced");
  }
}

function instrumentOption(instrument: MarketInstrument): MarketSymbolOption {
  return {
    label: instrument.symbol,
    value: instrument.symbol,
  };
}
</script>

<style scoped>
.market-symbol-autocomplete {
  display: flex;
  align-items: center;
  gap: 6px;
  box-sizing: border-box;
  width: 100%;
  max-width: 100%;
  min-width: 0;
}

.market-symbol-autocomplete__input {
  min-width: 0;
  flex: 1 1 auto;
}
</style>
