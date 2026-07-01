import { describe, expect, it } from "vitest";

import source from "./StrategyTaskFormPage.vue?raw";

describe("StrategyTaskFormPage market source contract", () => {
  it("uses backend-backed autocomplete suggestions for symbols", () => {
    expect(source).toContain('import MarketSymbolAutoComplete from "@/components/market/MarketSymbolAutoComplete.vue";');
    expect(source).toContain('<MarketSymbolAutoComplete v-model:value="form.symbol"');
    expect(source).toContain(':exchange="form.exchange"');
    expect(source).toContain('import StrategyMarketCatalogStatus from "@/components/strategy/StrategyMarketCatalogStatus.vue";');
    expect(source).toContain("<StrategyMarketCatalogStatus");
    expect(source).toContain("marketCatalogLabel");
    expect(source).toContain('t("research.marketStatus")');
    expect(source).not.toContain('<NSelect v-model:value="form.symbol"');
    expect(source).not.toContain('filterable tag');
    expect(source).not.toContain('value: "BTC-USDT"');
  });
});
