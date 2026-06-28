<template>
  <NAutoComplete
    :loading="loading"
    :options="options"
    :value="value"
    clearable
    @update:value="handleUpdateValue"
  />
</template>

<script setup lang="ts">
import { NAutoComplete } from "naive-ui";
import { ref, watch } from "vue";

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

const loading = ref(false);
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

function instrumentOption(instrument: MarketInstrument): MarketSymbolOption {
  return {
    label: instrument.symbol,
    value: instrument.symbol,
  };
}
</script>
