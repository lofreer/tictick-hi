import { describe, expect, it } from "vitest";

import source from "./ResearchPage.vue?raw";

describe("ResearchPage chart layout contract", () => {
  it("keeps the chart in a fixed flex viewport below the sync list", () => {
    const panelStyle = cssBlock(source, ".research-chart-panel");
    const bodyStyle = cssBlock(source, ".research-chart-body");

    expect(panelStyle).toContain("display: flex;");
    expect(panelStyle).toContain("flex-direction: column;");
    expect(panelStyle).toContain("--research-chart-viewport-height:");
    expect(panelStyle).toContain("height: auto;");
    expect(panelStyle).toContain("max-height: none;");
    expect(panelStyle).toContain("contain: layout paint;");
    expect(panelStyle).not.toContain("contain: size");
    expect(panelStyle).not.toContain("display: grid;");
    expect(panelStyle).not.toContain("grid-template-rows");
    expect(bodyStyle).toContain("flex: 0 0 var(--research-chart-viewport-height);");
    expect(bodyStyle).toContain("height: var(--research-chart-viewport-height);");
    expect(bodyStyle).toContain("max-height: var(--research-chart-viewport-height);");
    expect(cssDeclarations(bodyStyle)).not.toContain("height: 100%");
    expect(bodyStyle).toContain("contain: strict;");
    expect(source).toContain('class="research-chart-body" data-chart-viewport="fixed"');
  });

  it("uses exchange-specific symbol suggestions without locking the symbol input to a select", () => {
    expect(source).toContain('import { symbolOptionsForExchange } from "@/utils/marketSymbols";');
    expect(source).toContain(
      "const symbolOptions = computed<AutoCompleteOption[]>(() => symbolOptionsForExchange(exchange.value));",
    );
    expect(source).toContain(
      "const createSymbolOptions = computed<AutoCompleteOption[]>(() => symbolOptionsForExchange(createForm.exchange));",
    );
    expect(source).toContain('<NAutoComplete v-model:value="symbol"');
    expect(source).toContain('<NAutoComplete v-model:value="createForm.symbol"');
    expect(source).toContain(':options="createSymbolOptions"');
    expect(source).not.toContain('<NSelect v-model:value="symbol"');
    expect(source).not.toContain('<NSelect v-model:value="createForm.symbol"');
    expect(source).not.toContain(':options="symbolOptions" filterable tag');
  });
});

function cssBlock(source: string, selector: string) {
  const start = source.indexOf(`${selector} {`);
  if (start < 0) {
    throw new Error(`missing ${selector} style block`);
  }
  const bodyStart = source.indexOf("{", start) + 1;
  const bodyEnd = source.indexOf("}", bodyStart);
  if (bodyEnd < 0) {
    throw new Error(`unterminated ${selector} style block`);
  }
  return source.slice(bodyStart, bodyEnd);
}

function cssDeclarations(block: string) {
  return block
    .split(";")
    .map((line) => line.trim())
    .filter(Boolean);
}
