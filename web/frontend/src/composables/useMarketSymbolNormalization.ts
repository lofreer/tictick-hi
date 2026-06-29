import { watch, type Ref } from "vue";

import type { ResearchForm } from "@/composables/researchWorkspaceHelpers";
import { coerceSymbolForExchange, normalizeSymbolInput } from "@/utils/marketSymbols";

export function useMarketSymbolNormalization(exchange: Ref<string>, symbol: Ref<string>, createForm: ResearchForm) {
  watch(exchange, (nextExchange) => {
    symbol.value = coerceSymbolForExchange(nextExchange, symbol.value);
  });
  watch(symbol, (nextSymbol) => {
    const normalized = normalizeSymbolInput(nextSymbol);
    if (normalized !== nextSymbol) {
      symbol.value = normalized;
    }
  });
  watch(
    () => createForm.exchange,
    (nextExchange) => {
      createForm.symbol = coerceSymbolForExchange(nextExchange, createForm.symbol);
    },
  );
  watch(
    () => createForm.symbol,
    (nextSymbol) => {
      const normalized = normalizeSymbolInput(nextSymbol);
      if (normalized !== nextSymbol) {
        createForm.symbol = normalized;
      }
    },
  );
}
