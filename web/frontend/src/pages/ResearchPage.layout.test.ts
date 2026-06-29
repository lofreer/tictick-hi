/// <reference types="node" />
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

import windowControlsSource from "@/components/research/ResearchWindowControls.vue?raw";
import source from "./ResearchPage.vue?raw";

const pageStyles = readFileSync("src/pages/ResearchPage.css", "utf8");
const chartStyles = readFileSync("src/components/chart/TradingViewChart.css", "utf8");

describe("ResearchPage chart layout contract", () => {
  it("keeps the chart in a fixed flex viewport below the sync list", () => {
    const tasksStyle = cssBlock(pageStyles, ".research-tasks-panel");
    const panelStyle = cssBlock(pageStyles, ".research-chart-panel");
    const bodyStyle = cssBlock(pageStyles, ".research-chart-body");

    expect(pageStyles).toContain(".research-workspace");
    expect(pageStyles).toContain("overflow-x: clip;");
    expect(tasksStyle).toContain("width: 100%;");
    expect(tasksStyle).toContain("max-width: 100%;");
    expect(tasksStyle).toContain("min-width: 0;");
    expect(tasksStyle).toContain("max-height: 360px;");
    expect(tasksStyle).toContain("overflow: auto;");
    expect(tasksStyle).toContain("overscroll-behavior: contain;");
    expect(tasksStyle).not.toContain("overflow: hidden;");
    expect(panelStyle).toContain("display: flex;");
    expect(panelStyle).toContain("flex-direction: column;");
    expect(panelStyle).toContain("--research-chart-viewport-height:");
    expect(panelStyle).toContain("width: 100%;");
    expect(panelStyle).toContain("max-width: 100%;");
    expect(panelStyle).toContain("min-width: 0;");
    expect(panelStyle).toContain("height: auto;");
    expect(panelStyle).toContain("max-height: none;");
    expect(panelStyle).toContain("contain: layout paint;");
    expect(panelStyle).not.toContain("contain: size");
    expect(panelStyle).not.toContain("display: grid;");
    expect(panelStyle).not.toContain("grid-template-rows");
    expect(bodyStyle).toContain("flex: 0 0 var(--research-chart-viewport-height);");
    expect(bodyStyle).not.toContain("--tt-chart-inline-end-gutter");
    expect(bodyStyle).not.toContain("--tt-chart-block-end-gutter");
    expect(bodyStyle).toContain("height: var(--research-chart-viewport-height) !important;");
    expect(bodyStyle).toContain("max-height: var(--research-chart-viewport-height) !important;");
    expect(bodyStyle).toContain("block-size: var(--research-chart-viewport-height) !important;");
    expect(bodyStyle).toContain("max-block-size: var(--research-chart-viewport-height) !important;");
    expect(cssDeclarations(bodyStyle)).not.toContain("height: 100%");
    expect(bodyStyle).toContain("overflow: hidden;");
    expect(bodyStyle).toContain("contain: layout paint;");
    expect(pageStyles).toContain("height: 100% !important;");
    expect(pageStyles).toContain("max-height: 100% !important;");
    expect(pageStyles).toContain("block-size: 100% !important;");
    expect(pageStyles).toContain("max-block-size: 100% !important;");
    expect(pageStyles).toContain(".research-meta .n-tag__content");
    expect(pageStyles).toContain("overflow: hidden;");
    expect(pageStyles).toContain("text-overflow: ellipsis;");
    expect(pageStyles).toContain("white-space: nowrap;");
    expect(source).toContain('import "./ResearchPage.css";');
    expect(source).toContain('class="surface research-chart-panel"');
    expect(source).not.toContain('class="surface chart-panel research-chart-panel"');
    expect(source).toContain('class="research-chart-body" data-chart-viewport="fixed"');
  });

  it("reduces the fixed chart viewport when the app header stacks on narrow desktop widths", () => {
    expect(pageStyles).toContain("@media (min-width: 761px) and (max-width: 980px)");
    expect(pageStyles).toContain("--research-chart-viewport-height: clamp(240px, calc(100vh - 820px), 500px);");
    expect(pageStyles).toContain("--research-chart-viewport-height: clamp(240px, calc(100dvh - 820px), 500px);");
    expect(pageStyles).toContain("flex: 0 1 auto;");
  });

  it("lets the chart renderer fill the external viewport without inline size variables", () => {
    const rootStyle = cssBlock(chartStyles, ".trading-chart");
    const canvasStyle = cssBlock(chartStyles, ".trading-chart__canvas");
    const lightweightStyle = cssBlock(chartStyles, ".trading-chart__canvas > .tv-lightweight-charts");

    expect(rootStyle).toContain("position: relative;");
    expect(rootStyle).toContain("width: 100%;");
    expect(rootStyle).toContain("height: 100%;");
    expect(rootStyle).toContain("overflow: hidden;");
    expect(rootStyle).toContain("contain: layout style;");
    expect(canvasStyle).toContain("position: absolute;");
    expect(canvasStyle).toContain("inset: 0;");
    expect(canvasStyle).toContain("width: 100%;");
    expect(canvasStyle).toContain("height: 100%;");
    expect(canvasStyle).toContain("overflow: hidden;");
    expect(lightweightStyle).toContain("block-size: 100% !important;");
    expect(lightweightStyle).toContain("height: 100% !important;");
    expect(lightweightStyle).toContain("max-block-size: 100% !important;");
    expect(lightweightStyle).toContain("max-height: 100% !important;");
    expect(lightweightStyle).not.toContain("max-width");
    expect(lightweightStyle).not.toContain("max-inline-size");
    expect(lightweightStyle).not.toContain("overflow:");
    expect(lightweightStyle).not.toContain("contain:");
    expect(chartStyles).not.toContain("--tt-chart-render-width");
    expect(chartStyles).not.toContain("--tt-chart-render-height");
    expect(rootStyle).not.toContain("!important");
    expect(canvasStyle).not.toContain("!important");
  });

  it("does not override lightweight-charts internal table geometry", () => {
    expect(chartStyles).not.toContain(".trading-chart__canvas > .tv-lightweight-charts table");
    expect(chartStyles).not.toContain(".trading-chart__canvas > .tv-lightweight-charts tbody");
    expect(chartStyles).not.toContain(".trading-chart__canvas > .tv-lightweight-charts tr");
    expect(chartStyles).not.toContain(".trading-chart__canvas > .tv-lightweight-charts td");
    const canvasStyle = cssBlock(chartStyles, ".trading-chart__canvas canvas");
    expect(canvasStyle).not.toContain("--tt-chart-render-width");
    expect(canvasStyle).not.toContain("--tt-chart-render-height");
  });

  it("uses backend-backed symbol autocomplete without locking the symbol input to a select", () => {
    expect(source).toContain('import MarketSymbolAutoComplete from "@/components/market/MarketSymbolAutoComplete.vue";');
    expect(source).toMatch(/<MarketSymbolAutoComplete[\s\S]*?v-model:value="symbol"[\s\S]*?:exchange="exchange"[\s\S]*?@synced="loadMarketInstrumentSyncStatuses"/);
    expect(source).toContain(':exchange="exchange"');
    expect(source).toMatch(/<MarketSymbolAutoComplete[\s\S]*?v-model:value="createForm\.symbol"[\s\S]*?:exchange="createForm\.exchange"[\s\S]*?@synced="loadMarketInstrumentSyncStatuses"/);
    expect(source).toContain(':exchange="createForm.exchange"');
    expect(source).not.toContain('<NSelect v-model:value="symbol"');
    expect(source).not.toContain('<NSelect v-model:value="createForm.symbol"');
    expect(source).not.toContain(':options="symbolOptions" filterable tag');
  });

  it("exposes explicit candle window controls", () => {
    expect(source).toContain("ResearchWindowControls");
    expect(source).toContain("canLoadPreviousCandles");
    expect(source).toContain("canLoadNextCandles");
    expect(source).toContain('@range="applyTimeRange"');
    expect(source).toContain('@previous="loadPreviousCandles"');
    expect(source).toContain('@next="loadNextCandles"');
    expect(windowControlsSource).toContain("research.previousWindow");
    expect(windowControlsSource).toContain("research.nextWindow");
    expect(windowControlsSource).toContain("timeRangePresets");
    expect(windowControlsSource).toContain("research.timeRange");
  });

  it("shows the current candle window metadata", () => {
    expect(source).toContain("windowLabel");
    expect(source).toContain("research.candleWindow");
    expect(source).toContain("formatWindowTime");
  });

  it("passes candle gap markers into the chart renderer", () => {
    expect(source).toContain("chartMarkers");
    expect(source).toContain(':markers="chartMarkers"');
  });

  it("shows full-history market gap scan metadata for the selected chart source", () => {
    expect(source).toContain('import MarketCandleGapTag from "@/components/research/MarketCandleGapTag.vue";');
    expect(source).toContain('<MarketCandleGapTag :exchange="exchange" :interval="interval" :symbol="symbol" @repaired="loadTasks" />');
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
