import { describe, expect, it } from "vitest";

import source from "./StrategyTaskFormPage.vue?raw";

describe("StrategyTaskFormPage market source contract", () => {
  it("uses exchange-specific autocomplete suggestions for symbols", () => {
    expect(source).toContain("NAutoComplete");
    expect(source).toContain('<NAutoComplete v-model:value="form.symbol"');
    expect(source).toContain("symbolOptions,");
    expect(source).not.toContain('<NSelect v-model:value="form.symbol"');
    expect(source).not.toContain('filterable tag');
    expect(source).not.toContain('value: "BTC-USDT"');
  });
});
