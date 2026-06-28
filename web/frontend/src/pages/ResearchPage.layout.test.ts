import { describe, expect, it } from "vitest";

import windowControlsSource from "@/components/research/ResearchWindowControls.vue?raw";
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
    expect(bodyStyle).toContain("height: var(--research-chart-viewport-height) !important;");
    expect(bodyStyle).toContain("max-height: var(--research-chart-viewport-height) !important;");
    expect(bodyStyle).toContain("block-size: var(--research-chart-viewport-height) !important;");
    expect(bodyStyle).toContain("max-block-size: var(--research-chart-viewport-height) !important;");
    expect(cssDeclarations(bodyStyle)).not.toContain("height: 100%");
    expect(bodyStyle).toContain("contain: strict;");
    expect(source).toContain("height: 100% !important;");
    expect(source).toContain("max-height: 100% !important;");
    expect(source).toContain("block-size: 100% !important;");
    expect(source).toContain("max-block-size: 100% !important;");
    expect(source).toContain('class="research-chart-body" data-chart-viewport="fixed"');
  });

  it("reduces the fixed chart viewport when the app header stacks on narrow desktop widths", () => {
    expect(source).toContain("@media (min-width: 761px) and (max-width: 980px)");
    expect(source).toContain("--research-chart-viewport-height: clamp(300px, calc(100vh - 820px), 500px);");
    expect(source).toContain("--research-chart-viewport-height: clamp(300px, calc(100dvh - 820px), 500px);");
  });

  it("uses backend-backed symbol autocomplete without locking the symbol input to a select", () => {
    expect(source).toContain('import MarketSymbolAutoComplete from "@/components/market/MarketSymbolAutoComplete.vue";');
    expect(source).toContain('<MarketSymbolAutoComplete v-model:value="symbol"');
    expect(source).toContain(':exchange="exchange"');
    expect(source).toContain('<MarketSymbolAutoComplete v-model:value="createForm.symbol"');
    expect(source).toContain(':exchange="createForm.exchange"');
    expect(source).not.toContain('<NSelect v-model:value="symbol"');
    expect(source).not.toContain('<NSelect v-model:value="createForm.symbol"');
    expect(source).not.toContain(':options="symbolOptions" filterable tag');
  });

  it("exposes explicit candle window controls", () => {
    expect(source).toContain("ResearchWindowControls");
    expect(source).toContain("canLoadPreviousCandles");
    expect(source).toContain("canLoadNextCandles");
    expect(source).toContain('@previous="loadPreviousCandles"');
    expect(source).toContain('@next="loadNextCandles"');
    expect(windowControlsSource).toContain("research.previousWindow");
    expect(windowControlsSource).toContain("research.nextWindow");
  });

  it("shows the current candle window metadata", () => {
    expect(source).toContain("windowLabel");
    expect(source).toContain("research.candleWindow");
    expect(source).toContain("formatWindowTime");
  });

  it("shows full-history market gap scan metadata for the selected chart source", () => {
    expect(source).toContain('import MarketCandleGapTag from "@/components/research/MarketCandleGapTag.vue";');
    expect(source).toContain('<MarketCandleGapTag :exchange="exchange" :interval="interval" :symbol="symbol" />');
  });

  it("shows returned and total gap counts when gap details are limited", () => {
    expect(source).toContain("research.gapDetailsLimited");
    expect(source).toContain("returned: gapDetails.returnedCount");
    expect(source).toContain("total: gapDetails.totalCount");
    expect(source).toContain("limit: gapDetails.repairLimit");
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
