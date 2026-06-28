<template>
  <div class="market-symbol-autocomplete">
    <NAutoComplete
      class="market-symbol-autocomplete__input"
      :loading="loading"
      :options="options"
      :value="value"
      clearable
      @update:value="handleUpdateValue"
    />
    <NButton
      circle
      quaternary
      size="small"
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
import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";

import { marketApi } from "@/services/api/market";
import type { MarketInstrument } from "@/types/app";
import { normalizeSymbolInput, symbolOptionsForExchange, type MarketSymbolOption } from "@/utils/marketSymbols";

const props = defineProps<{
  exchange: string;
  value: string;
}>();

const emit = defineEmits<{
  "update:value": [value: string];
}>();

const { t } = useI18n();
const loading = ref(false);
const syncing = ref(false);
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
  syncing.value = true;
  try {
    await marketApi.syncInstruments(props.exchange);
    await loadOptions(props.value);
  } catch {
    options.value = symbolOptionsForExchange(props.exchange);
  } finally {
    syncing.value = false;
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
  width: 100%;
}

.market-symbol-autocomplete__input {
  min-width: 0;
  flex: 1 1 auto;
}
</style>
