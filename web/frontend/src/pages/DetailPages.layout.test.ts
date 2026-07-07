/// <reference types="node" />
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

import backtestSource from "./BacktestDetailPage.vue?raw";
import tradingSource from "./TradingDetailPage.vue?raw";

const detailChartStyles = readFileSync("src/pages/detailChartLayout.css", "utf8");
const klineChartStyles = readFileSync("src/pages/klineChartLayout.css", "utf8");

describe("strategy detail page layout contract", () => {
  it("keeps trading detail chart above summary and tabbed lists", () => {
    const chartIndex = tradingSource.indexOf('class="surface kline-chart-frame trading-detail-chart"');
    const gridIndex = tradingSource.indexOf('class="trading-detail-lower-grid"');
    const summaryIndex = tradingSource.indexOf('class="surface trading-detail-section trading-detail-summary"');
    const tabsIndex = tradingSource.indexOf('class="surface trading-detail-section trading-detail-tabs"');
    const styles = styleBlock(tradingSource);
    const frameStyle = cssBlock(klineChartStyles, ".kline-chart-frame");
    const frameViewportStyle = cssBlock(klineChartStyles, ".kline-chart-frame__viewport");

    expect(tradingSource).toContain('class="trading-detail-workspace"');
    expect(tradingSource).toContain('import "./detailChartLayout.css";');
    expect(tradingSource).toContain('import "./klineChartLayout.css";');
    expect(chartIndex).toBeGreaterThan(-1);
    expect(tradingSource).toContain('class="surface kline-chart-frame trading-detail-chart"');
    expect(tradingSource).toContain('class="kline-chart-frame__viewport trading-detail-chart-viewport" data-chart-viewport="fixed"');
    expect(gridIndex).toBeGreaterThan(chartIndex);
    expect(summaryIndex).toBeGreaterThan(gridIndex);
    expect(tabsIndex).toBeGreaterThan(summaryIndex);
    expect(tradingSource).toContain("<NTabs type=\"segment\" animated>");
    expect(tradingSource).toContain("notification.providerMessageId");
    expect(tradingSource).toContain("trading.providerMessageId");
    expect(tradingSource).not.toContain('v-else-if="task" class="workspace-grid"');
    expect(tradingSource).not.toContain('class="side-panel"');
    expect(tradingSource).not.toContain('class="surface chart-panel trading-detail-chart"');
    expect(detailChartStyles).not.toContain("--kline-chart-plot-height:");
    expect(detailChartStyles).not.toContain("--kline-chart-padding-left:");
    expect(detailChartStyles).not.toContain("--kline-chart-padding-right:");
    expect(klineChartStyles).toContain("--kline-chart-plot-height: clamp(680px, 72dvh, 820px);");
    expect(klineChartStyles).toContain("--kline-chart-padding-left: 14px;");
    expect(klineChartStyles).toContain("--kline-chart-padding-right: 2px;");
    expect(klineChartStyles).toContain("--kline-chart-frame-height:");
    expect(frameStyle).toContain("height: var(--kline-chart-frame-height);");
    expect(frameStyle).toContain("var(--kline-chart-padding-right)");
    expect(frameStyle).toContain("var(--kline-chart-padding-left)");
    expect(detailChartStyles).toContain(".trading-detail-chart-viewport,");
    expect(frameViewportStyle).toContain("height: var(--kline-chart-plot-height);");
    expect(styles).toContain(".trading-detail-lower-grid");
    expect(styles).toContain("grid-template-columns: minmax(220px, 260px) minmax(0, 1fr);");
    expect(styles).toContain("align-items: start;");
    expect(styles).toContain(".trading-detail-tabs");
    expect(styles).toContain("min-height: 380px;");
    expect(styles).toContain("align-self: stretch;");
    expect(klineChartStyles).toContain("@media (max-width: 980px)");
    expect(klineChartStyles).toContain("@media (max-width: 760px)");
    expect(klineChartStyles).toContain("--kline-chart-plot-height: 580px;");
    expect(styles).toContain("grid-template-columns: 1fr;");
  });

  it("keeps backtest detail chart above summary and tabbed lists", () => {
    const chartIndex = backtestSource.indexOf('class="surface kline-chart-frame backtest-chart-panel"');
    const gridIndex = backtestSource.indexOf('class="backtest-detail-lower-grid"');
    const summaryIndex = backtestSource.indexOf('class="surface backtest-side-section backtest-summary-panel"');
    const tabsIndex = backtestSource.indexOf('class="surface backtest-side-section backtest-detail-tabs"');
    const styles = styleBlock(backtestSource);
    const frameStyle = cssBlock(klineChartStyles, ".kline-chart-frame");
    const frameViewportStyle = cssBlock(klineChartStyles, ".kline-chart-frame__viewport");

    expect(backtestSource).toContain('class="backtest-detail-workspace"');
    expect(backtestSource).toContain('import "./detailChartLayout.css";');
    expect(backtestSource).toContain('import "./klineChartLayout.css";');
    expect(chartIndex).toBeGreaterThan(-1);
    expect(backtestSource).toContain('class="surface kline-chart-frame backtest-chart-panel"');
    expect(backtestSource).toContain('class="kline-chart-frame__viewport backtest-chart-viewport" data-chart-viewport="fixed"');
    expect(gridIndex).toBeGreaterThan(chartIndex);
    expect(summaryIndex).toBeGreaterThan(gridIndex);
    expect(tabsIndex).toBeGreaterThan(summaryIndex);
    expect(backtestSource).toContain("<NTabs type=\"segment\" animated>");
    expect(backtestSource).toContain('<NTabPane name="parameters"');
    expect(backtestSource).toContain('<NTabPane name="intents"');
    expect(backtestSource).toContain('<NTabPane name="orders"');
    expect(backtestSource).not.toContain('v-else-if="task" class="workspace-grid"');
    expect(backtestSource).not.toContain('class="side-panel"');
    expect(backtestSource).not.toContain('class="surface chart-panel backtest-chart-panel"');
    expect(detailChartStyles).not.toContain("--kline-chart-plot-height:");
    expect(detailChartStyles).not.toContain("--kline-chart-padding-left:");
    expect(detailChartStyles).not.toContain("--kline-chart-padding-right:");
    expect(klineChartStyles).toContain("--kline-chart-plot-height: clamp(680px, 72dvh, 820px);");
    expect(klineChartStyles).toContain("--kline-chart-padding-left: 14px;");
    expect(klineChartStyles).toContain("--kline-chart-padding-right: 2px;");
    expect(klineChartStyles).toContain("--kline-chart-frame-height:");
    expect(frameStyle).toContain("height: var(--kline-chart-frame-height);");
    expect(frameStyle).toContain("var(--kline-chart-padding-right)");
    expect(frameStyle).toContain("var(--kline-chart-padding-left)");
    expect(detailChartStyles).toContain(".backtest-chart-viewport");
    expect(frameViewportStyle).toContain("height: var(--kline-chart-plot-height);");
    expect(styles).toContain(".backtest-detail-lower-grid");
    expect(styles).toContain("grid-template-columns: minmax(220px, 260px) minmax(0, 1fr);");
    expect(styles).toContain("align-items: start;");
    expect(styles).toContain(".backtest-detail-tabs");
    expect(styles).toContain("min-height: 380px;");
    expect(styles).toContain("align-self: stretch;");
    expect(klineChartStyles).toContain("@media (max-width: 980px)");
    expect(klineChartStyles).toContain("@media (max-width: 760px)");
    expect(klineChartStyles).toContain("--kline-chart-plot-height: 580px;");
    expect(styles).toContain("grid-template-columns: 1fr;");
  });
});

function styleBlock(source: string) {
  const start = source.indexOf("<style scoped>");
  const end = source.indexOf("</style>", start);
  if (start < 0 || end < 0) {
    throw new Error("missing scoped style block");
  }
  return source.slice(start, end);
}

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
