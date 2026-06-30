/// <reference types="node" />
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

import windowControlsSource from "@/components/research/ResearchWindowControls.vue?raw";
import source from "./ResearchPage.vue?raw";

const pageStyles = readFileSync("src/pages/ResearchPage.css", "utf8");
const chartStyles = readFileSync("src/components/chart/TradingViewChart.css", "utf8");
const klineChartStyles = readFileSync("src/pages/klineChartLayout.css", "utf8");

describe("ResearchPage chart layout contract", () => {
  it("keeps the chart in a fixed flex viewport below the sync list", () => {
    const tasksStyle = cssBlock(pageStyles, ".research-tasks-panel");
    const panelStyle = cssBlock(pageStyles, ".research-chart-panel");
    const toolbarStyle = cssBlock(pageStyles, ".research-toolbar");
    const controlsStyle = cssBlock(pageStyles, ".research-source-controls");
    const statusStyle = cssBlock(pageStyles, ".research-toolbar-status");
    const bodyStyle = cssBlock(pageStyles, ".research-chart-body");
    const viewportStyle = cssBlock(pageStyles, ".research-chart-viewport");
    const frameStyle = cssBlock(klineChartStyles, ".kline-chart-frame");
    const frameViewportStyle = cssBlock(klineChartStyles, ".kline-chart-frame__viewport");
    const frameViewportChartStyle = cssBlock(klineChartStyles, ".kline-chart-frame__viewport .state-block,\n.kline-chart-frame__viewport .trading-chart");

    expect(pageStyles).toContain(".research-workspace");
    expect(pageStyles).toContain("overflow-x: clip;");
    expect(tasksStyle).toContain("width: 100%;");
    expect(tasksStyle).toContain("max-width: 100%;");
    expect(tasksStyle).toContain("min-width: 0;");
    expect(tasksStyle).toContain("max-height: clamp(156px, 18vh, 188px);");
    expect(tasksStyle).toContain("overflow: auto;");
    expect(tasksStyle).toContain("overscroll-behavior: contain;");
    expect(tasksStyle).not.toContain("overflow: hidden;");
    expect(panelStyle).toContain("display: flex;");
    expect(panelStyle).toContain("flex-direction: column;");
    expect(panelStyle).toContain("width: 100%;");
    expect(panelStyle).toContain("max-width: 100%;");
    expect(panelStyle).toContain("min-width: 0;");
    expect(panelStyle).toContain("height: auto;");
    expect(panelStyle).toContain("max-height: none;");
    expect(panelStyle).toContain("contain: layout paint;");
    expect(panelStyle).not.toContain("contain: size");
    expect(panelStyle).not.toContain("display: grid;");
    expect(panelStyle).not.toContain("grid-template-rows");
    expect(bodyStyle).toContain("--kline-chart-plot-height:");
    expect(bodyStyle).toContain("--kline-chart-frame-height:");
    expect(bodyStyle).toContain("flex: 0 0 var(--kline-chart-frame-height);");
    expect(frameStyle).toContain("height: var(--kline-chart-frame-height);");
    expect(frameStyle).toContain("max-height: var(--kline-chart-frame-height);");
    expect(frameStyle).toContain("block-size: var(--kline-chart-frame-height);");
    expect(frameStyle).toContain("max-block-size: var(--kline-chart-frame-height);");
    expect(frameStyle).toContain("var(--kline-chart-padding-right)");
    expect(frameStyle).toContain("var(--kline-chart-padding-left)");
    expect(cssDeclarations(bodyStyle)).not.toContain("height: 100%");
    expect(frameStyle).toContain("overflow: hidden;");
    expect(frameStyle).toContain("contain: layout paint;");
    expect(frameViewportStyle).toContain("height: var(--kline-chart-plot-height);");
    expect(frameViewportStyle).toContain("max-height: var(--kline-chart-plot-height);");
    expect(frameViewportStyle).toContain("block-size: var(--kline-chart-plot-height);");
    expect(frameViewportStyle).toContain("max-block-size: var(--kline-chart-plot-height);");
    expect(frameViewportStyle).toContain("overflow: hidden;");
    expect(viewportStyle).toContain("isolation: isolate;");
    expect(frameViewportChartStyle).toContain("height: 100%;");
    expect(frameViewportChartStyle).toContain("max-height: 100%;");
    expect(frameViewportChartStyle).toContain("block-size: 100%;");
    expect(frameViewportChartStyle).toContain("max-block-size: 100%;");
    expect(frameViewportChartStyle).toContain("width: 100%;");
    expect(toolbarStyle).toContain("display: grid;");
    expect(toolbarStyle).toContain("grid-template-columns: max-content minmax(0, 1fr);");
    expect(toolbarStyle).toContain("padding: 6px 10px;");
    expect(toolbarStyle).toContain("overflow: hidden;");
    expect(controlsStyle).toContain("display: grid;");
    expect(controlsStyle).toContain("grid-template-columns: 100px 132px 28px 56px max-content;");
    expect(controlsStyle).toContain("width: max-content;");
    expect(controlsStyle).toContain("overflow-x: auto;");
    expect(statusStyle).toContain("justify-content: flex-end;");
    expect(statusStyle).toContain("flex-wrap: nowrap;");
    expect(statusStyle).toContain("overflow-x: auto;");
    expect(statusStyle).toContain("overflow-y: hidden;");
    expect(pageStyles).toContain(".research-toolbar-main");
    expect(pageStyles).toContain(".research-current-source");
    expect(pageStyles).toContain(".research-meta .n-tag__content");
    expect(pageStyles).toContain("overflow: hidden;");
    expect(pageStyles).toContain("text-overflow: ellipsis;");
    expect(pageStyles).toContain("white-space: nowrap;");
    expect(pageStyles).not.toContain("flex: 1 1 620px;");
    expect(pageStyles).not.toContain("flex: 0 1 680px;");
    expect(pageStyles).not.toContain("width: clamp(180px, 22vw, 360px);");
    expect(pageStyles).not.toContain("width: clamp(180px, 15vw, 240px);");
    expect(source).toContain('import "./ResearchPage.css";');
    expect(source).toContain('import "./klineChartLayout.css";');
    expect(source).toContain('class="surface research-chart-panel"');
    expect(source).not.toContain('class="surface chart-panel research-chart-panel"');
    expect(source).toContain('class="research-toolbar-main"');
    expect(source).toContain('class="research-source-controls"');
    expect(source).toContain('class="research-toolbar-status"');
    expect(source).toContain('class="research-refresh-button"');
    expect(source).toContain("RefreshCw");
    expect(source).toContain('class="research-current-source"');
    expect(source).toContain('class="research-select research-select--exchange"');
    expect(source).toContain(':show-sync-button="false"');
    expect(source).toContain('size="small"');
    expect(source).not.toContain('class="toolbar-row"');
    expect(source).toContain('class="kline-chart-frame research-chart-body"');
    expect(source).toContain('class="kline-chart-frame__viewport research-chart-viewport" data-chart-viewport="fixed"');
  });

  it("keeps a readable chart viewport when the app header stacks on narrow desktop widths", () => {
    expect(pageStyles).toContain("@media (min-width: 761px) and (max-width: 980px)");
    expect(pageStyles).toContain("--kline-chart-plot-height: clamp(720px, 78vh, 900px);");
    expect(pageStyles).toContain("--kline-chart-plot-height: 760px;");
    expect(pageStyles).toContain("--kline-chart-plot-height: 600px;");
    expect(pageStyles).toContain("--kline-chart-padding-left: 22px;");
    expect(pageStyles).toContain("--kline-chart-padding-left: 18px;");
    expect(pageStyles).toContain("--kline-chart-padding-left: 12px;");
    expect(pageStyles).toContain("--kline-chart-padding-right: 6px;");
    expect(pageStyles).toContain("--kline-chart-padding-right: 8px;");
    expect(pageStyles).toContain("grid-template-columns: 100px 132px 28px 56px max-content;");
    expect(pageStyles).toContain("width: 132px;");
    expect(pageStyles).toContain("grid-template-columns: 98px 126px 28px 54px max-content;");
    expect(pageStyles).toContain("grid-template-columns: 96px 120px 28px 54px max-content;");
    expect(pageStyles).toContain("width: 100%;");
    expect(pageStyles).toContain("max-width: min(240px, 30vw);");
    expect(pageStyles).toContain("overflow-x: auto;");
  });

  it("lets the chart renderer fill the external viewport without gutter shrinkage", () => {
    const rootStyle = cssBlock(chartStyles, ".trading-chart");
    const canvasStyle = cssBlock(chartStyles, ".trading-chart__canvas");
    const lightweightStyle = cssBlock(chartStyles, ".trading-chart__canvas > .tv-lightweight-charts");

    expect(rootStyle).toContain("position: relative;");
    expect(rootStyle).toContain("width: 100%;");
    expect(rootStyle).toContain("height: 100%;");
    expect(rootStyle).toContain("max-inline-size: 100%;");
    expect(rootStyle).toContain("max-block-size: 100%;");
    expect(rootStyle).toContain("overflow: hidden;");
    expect(rootStyle).toContain("contain: layout style;");
    expect(canvasStyle).toContain("position: absolute;");
    expect(canvasStyle).toContain("top: 0;");
    expect(canvasStyle).toContain("left: 0;");
    expect(canvasStyle).toContain("width: 100%;");
    expect(canvasStyle).toContain("height: 100%;");
    expect(canvasStyle).toContain("overflow: hidden;");
    expect(lightweightStyle).toContain("block-size: 100% !important;");
    expect(lightweightStyle).toContain("inline-size: 100% !important;");
    expect(lightweightStyle).toContain("height: 100% !important;");
    expect(lightweightStyle).toContain("width: 100% !important;");
    expect(lightweightStyle).toContain("max-block-size: 100% !important;");
    expect(lightweightStyle).toContain("max-height: 100% !important;");
    expect(lightweightStyle).not.toContain("max-width");
    expect(lightweightStyle).not.toContain("max-inline-size");
    expect(lightweightStyle).not.toContain("overflow:");
    expect(lightweightStyle).not.toContain("contain:");
    expect(chartStyles).not.toContain("--tt-chart-render-width");
    expect(chartStyles).not.toContain("--tt-chart-render-height");
    expect(chartStyles).not.toContain("--tt-chart-inline-end-gutter");
    expect(chartStyles).not.toContain("--tt-chart-block-end-gutter");
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

  it("shows full-history invalid candle scan metadata for the selected chart source", () => {
    expect(source).toContain('import MarketCandleInvalidIssueTag from "@/components/research/MarketCandleInvalidIssueTag.vue";');
    expect(source).toContain('<MarketCandleInvalidIssueTag :exchange="exchange" :interval="interval" :symbol="symbol" @repaired="loadTasks" />');
  });

  it("shows returned and total gap counts when gap details are limited", () => {
    expect(source).toContain("research.gapDetailsLimited");
    expect(source).toContain("returned: gapDetails.returnedCount");
    expect(source).toContain("total: gapDetails.totalCount");
    expect(source).toContain("limit: gapDetails.repairLimit");
  });

  it("keeps task invalid issue details reachable from the research page", () => {
    expect(source).toContain('import ResearchTaskInvalidIssueModal from "@/components/research/ResearchTaskInvalidIssueModal.vue";');
    expect(source).toContain('@view-invalid="viewTaskInvalidIssues"');
    expect(source).toContain('<ResearchTaskInvalidIssueModal ref="invalidIssueModal" @repaired="loadTasks" />');
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
