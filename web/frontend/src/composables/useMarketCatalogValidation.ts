import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";

import {
  readMarketInstrumentCatalogLookup,
  type MarketInstrumentCatalogStatus,
} from "@/utils/marketInstrumentCatalog";
import { isSymbolFormatForExchange, normalizeSymbolInput } from "@/utils/marketSymbols";

export type MarketCatalogValidationStatus = MarketInstrumentCatalogStatus | "unknown";

export function useMarketCatalogValidation(fields: {
  exchange: () => string;
  symbol: () => string;
}) {
  const { t } = useI18n();
  const marketCatalogError = ref("");
  const marketCatalogLoading = ref(false);
  const marketCatalogStatus = ref<MarketCatalogValidationStatus>("unknown");
  const marketCatalogStatusDetail = ref("");
  let requestSequence = 0;

  watch(
    [fields.exchange, fields.symbol],
    () => {
      void refreshMarketInstrumentCatalogStatus();
    },
    { immediate: true },
  );

  async function refreshMarketInstrumentCatalogStatus(): Promise<MarketCatalogValidationStatus> {
    const sequence = ++requestSequence;
    const exchange = fields.exchange().trim();
    const symbol = normalizeSymbolInput(fields.symbol());
    marketCatalogError.value = "";
    marketCatalogStatusDetail.value = "";

    if (exchange === "" || symbol === "" || !isSymbolFormatForExchange(exchange, symbol)) {
      marketCatalogLoading.value = false;
      marketCatalogStatus.value = "unknown";
      return "unknown";
    }

    marketCatalogLoading.value = true;
    try {
      const lookup = await readMarketInstrumentCatalogLookup(exchange, symbol);
      if (sequence !== requestSequence) return marketCatalogStatus.value;
      marketCatalogStatus.value = lookup.status;
      marketCatalogStatusDetail.value = lookup.instrument?.exchangeStatus ?? lookup.instrument?.status ?? "";
      return lookup.status;
    } catch {
      if (sequence === requestSequence) {
        marketCatalogStatus.value = "unknown";
        marketCatalogError.value = t("research.instrumentValidationFailed");
      }
      return "unknown";
    } finally {
      if (sequence === requestSequence) {
        marketCatalogLoading.value = false;
      }
    }
  }

  return {
    marketCatalogError,
    marketCatalogLoading,
    marketCatalogStatus,
    marketCatalogStatusDetail,
    refreshMarketInstrumentCatalogStatus,
  };
}
