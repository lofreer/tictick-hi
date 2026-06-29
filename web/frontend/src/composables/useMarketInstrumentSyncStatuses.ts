import { computed, ref, type Ref } from "vue";

import { marketApi } from "@/services/api/market";
import type { MarketInstrumentSyncStatus } from "@/types/app";

export function useMarketInstrumentSyncStatuses(
  currentExchange: Ref<string>,
  createExchange: Ref<string>,
  fallbackMessage: (error: unknown) => string,
) {
  const statuses = ref<MarketInstrumentSyncStatus[]>([]);
  const error = ref("");

  const byExchange = computed(() => new Map(statuses.value.map((status) => [status.exchange, status])));
  const current = computed(() => byExchange.value.get(currentExchange.value) ?? null);
  const create = computed(() => byExchange.value.get(createExchange.value) ?? null);

  async function load() {
    error.value = "";
    try {
      statuses.value = await marketApi.listInstrumentSyncStatuses();
    } catch (loadError) {
      error.value = fallbackMessage(loadError);
    }
  }

  return {
    create,
    current,
    error,
    load,
  };
}
