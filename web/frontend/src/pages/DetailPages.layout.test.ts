/// <reference types="node" />
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

import backtestSource from "./BacktestDetailPage.vue?raw";
import tradingSource from "./TradingDetailPage.vue?raw";

const detailChartStyles = readFileSync("src/pages/detailChartLayout.css", "utf8");

describe("strategy detail page layout contract", () => {
  it("keeps trading detail chart above summary and tabbed lists", () => {
    const chartIndex = tradingSource.indexOf('class="surface trading-detail-chart"');
    const gridIndex = tradingSource.indexOf('class="trading-detail-lower-grid"');
    const summaryIndex = tradingSource.indexOf('class="surface trading-detail-section trading-detail-summary"');
    const tabsIndex = tradingSource.indexOf('class="surface trading-detail-section trading-detail-tabs"');
    const styles = styleBlock(tradingSource);

    expect(tradingSource).toContain('class="trading-detail-workspace"');
    expect(tradingSource).toContain('import "./detailChartLayout.css";');
    expect(chartIndex).toBeGreaterThan(-1);
    expect(tradingSource).toContain('class="trading-detail-chart-viewport" data-chart-viewport="fixed"');
    expect(gridIndex).toBeGreaterThan(chartIndex);
    expect(summaryIndex).toBeGreaterThan(gridIndex);
    expect(tabsIndex).toBeGreaterThan(summaryIndex);
    expect(tradingSource).toContain("<NTabs type=\"segment\" animated>");
    expect(tradingSource).not.toContain('v-else-if="task" class="workspace-grid"');
    expect(tradingSource).not.toContain('class="side-panel"');
    expect(tradingSource).not.toContain('class="surface chart-panel trading-detail-chart"');
    expect(detailChartStyles).toContain(".trading-detail-chart,");
    expect(detailChartStyles).toContain("height: clamp(560px, 68dvh, 800px);");
    expect(detailChartStyles).toContain("padding: 16px 18px 18px;");
    expect(detailChartStyles).toContain(".trading-detail-chart-viewport,");
    expect(detailChartStyles).toContain(".trading-detail-chart-viewport .trading-chart,");
    expect(styles).toContain(".trading-detail-lower-grid");
    expect(styles).toContain("grid-template-columns: minmax(280px, 340px) minmax(0, 1fr);");
    expect(detailChartStyles).toContain("@media (max-width: 980px)");
    expect(styles).toContain("grid-template-columns: 1fr;");
  });

  it("keeps backtest detail chart above summary and tabbed lists", () => {
    const chartIndex = backtestSource.indexOf('class="surface backtest-chart-panel"');
    const gridIndex = backtestSource.indexOf('class="backtest-detail-lower-grid"');
    const summaryIndex = backtestSource.indexOf('class="surface backtest-side-section backtest-summary-panel"');
    const tabsIndex = backtestSource.indexOf('class="surface backtest-side-section backtest-detail-tabs"');
    const styles = styleBlock(backtestSource);

    expect(backtestSource).toContain('class="backtest-detail-workspace"');
    expect(backtestSource).toContain('import "./detailChartLayout.css";');
    expect(chartIndex).toBeGreaterThan(-1);
    expect(backtestSource).toContain('class="backtest-chart-viewport" data-chart-viewport="fixed"');
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
    expect(detailChartStyles).toContain(".backtest-chart-panel");
    expect(detailChartStyles).toContain("height: clamp(560px, 68dvh, 800px);");
    expect(detailChartStyles).toContain("padding: 16px 18px 18px;");
    expect(detailChartStyles).toContain(".backtest-chart-viewport");
    expect(detailChartStyles).toContain(".backtest-chart-viewport .trading-chart");
    expect(styles).toContain(".backtest-detail-lower-grid");
    expect(styles).toContain("grid-template-columns: minmax(280px, 340px) minmax(0, 1fr);");
    expect(detailChartStyles).toContain("@media (max-width: 980px)");
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
