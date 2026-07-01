/// <reference types="node" />
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

import chartInvalidIssueRepairSource from "@/components/research/ChartInvalidIssueRepairAction.vue?raw";
import invalidIssueModalSource from "@/components/research/ResearchTaskInvalidIssueModal.vue?raw";
import gapDetailsModalSource from "@/components/research/ResearchTaskGapDetailsModal.vue?raw";
import windowControlsSource from "@/components/research/ResearchWindowControls.vue?raw";
import source from "./ResearchPage.vue?raw";

const pageStyles = readFileSync("src/pages/ResearchPage.css", "utf8");
const chartStyles = readFileSync("src/components/chart/TradingViewChart.css", "utf8");
const klineChartStyles = readFileSync("src/pages/klineChartLayout.css", "utf8");
const zhResearchMessages = readFileSync("src/i18n/messages.research.zh.ts", "utf8");
const enResearchMessages = readFileSync("src/i18n/messages.research.en.ts", "utf8");

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
    expect(bodyStyle).toContain("flex: 0 0 var(--kline-chart-frame-height);");
    expect(bodyStyle).not.toContain("--kline-chart-plot-height:");
    expect(bodyStyle).not.toContain("--kline-chart-padding-left:");
    expect(bodyStyle).not.toContain("--kline-chart-padding-right:");
    expect(klineChartStyles).toContain("--kline-chart-plot-height: clamp(680px, 72dvh, 820px);");
    expect(klineChartStyles).toContain("--kline-chart-frame-height:");
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
    expect(toolbarStyle).toContain("padding: 7px 12px;");
    expect(toolbarStyle).toContain("overflow: hidden;");
    expect(controlsStyle).toContain("display: grid;");
    expect(controlsStyle).toContain("grid-template-columns: 96px 96px 28px 52px max-content;");
    expect(controlsStyle).toContain("width: max-content;");
    expect(controlsStyle).toContain("overflow-x: auto;");
    expect(statusStyle).toContain("justify-content: flex-start;");
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
    expect(klineChartStyles).toContain("--kline-chart-plot-height: clamp(680px, 72dvh, 820px);");
    expect(klineChartStyles).toContain("--kline-chart-plot-height: 700px;");
    expect(klineChartStyles).toContain("--kline-chart-plot-height: 580px;");
    expect(klineChartStyles).toContain("--kline-chart-padding-left: 14px;");
    expect(klineChartStyles).toContain("--kline-chart-padding-left: 12px;");
    expect(klineChartStyles).toContain("--kline-chart-padding-left: 10px;");
    expect(klineChartStyles).toContain("--kline-chart-padding-right: 2px;");
    expect(pageStyles).not.toContain("--kline-chart-plot-height: 700px;");
    expect(pageStyles).not.toContain("--kline-chart-padding-left: 12px;");
    expect(pageStyles).toContain("grid-template-columns: 96px 96px 28px 52px max-content;");
    expect(pageStyles).toContain("width: 96px;");
    expect(pageStyles).toContain("grid-template-columns: 92px 96px 28px 50px max-content;");
    expect(pageStyles).toContain("grid-template-columns: 88px 92px 28px 50px max-content;");
    expect(pageStyles).toContain("width: 100%;");
    expect(pageStyles).toContain("max-width: min(220px, 24vw);");
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
    expect(lightweightStyle).toContain("max-inline-size: 100% !important;");
    expect(lightweightStyle).toContain("max-width: 100% !important;");
    expect(lightweightStyle).toContain("overflow: hidden;");
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
    expect(source).toContain('@range="applyChartTimeRange"');
    expect(source).toContain('@previous="loadPreviousChartCandles"');
    expect(source).toContain('@next="loadNextChartCandles"');
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

  it("shows candle coverage when non-ok metadata is returned", () => {
    expect(source).toContain("candleCoverageLabel");
    expect(source).toContain("candleCoverageTagType");
    expect(source).toContain("shouldShowCandleCoverage");
    expect(source).toContain("coverageVisible");
    expect(source).toContain("coverageTagType");
    expect(source).not.toContain('v-if="coverageLimited"');
  });

  it("passes candle gap markers into the chart renderer", () => {
    expect(source).toContain("chartMarkers");
    expect(source).toContain(':markers="chartMarkers"');
  });

  it("shows full-history market gap scan metadata for the selected chart source", () => {
    expect(source).toContain('import MarketCandleGapTag from "@/components/research/MarketCandleGapTag.vue";');
    expect(source).toContain('<MarketCandleGapTag :exchange="exchange" :interval="interval" :symbol="symbol" :tasks="tasks" @repaired="startRepairPollingForResult" />');
  });

  it("shows full-history invalid candle scan metadata for the selected chart source", () => {
    expect(source).toContain('import MarketCandleInvalidIssueTag from "@/components/research/MarketCandleInvalidIssueTag.vue";');
    expect(source).toContain('<MarketCandleInvalidIssueTag :exchange="exchange" :interval="interval" :symbol="symbol" :tasks="tasks" @repaired="startRepairPollingForResult" />');
  });

  it("shows returned and total gap counts when gap details are limited", () => {
    expect(source).toContain('import ResearchTaskGapDetailsModal from "@/components/research/ResearchTaskGapDetailsModal.vue";');
    expect(source).toContain("<ResearchTaskGapDetailsModal");
    expect(source).toContain('v-model:show="gapDetailsModalOpen"');
    expect(gapDetailsModalSource).toContain("research.gapDetailsLimited");
    expect(gapDetailsModalSource).toContain("returned: details.returnedCount");
    expect(gapDetailsModalSource).toContain("total: details.totalCount");
    expect(gapDetailsModalSource).toContain("limit: details.repairLimit");
  });

  it("keeps task gap repair results visible in the gap details modal", () => {
    expect(source).toContain("taskGapRepairResult");
    expect(source).toContain("taskGapRepairNotice");
    expect(source).toContain('@repair="gapDetailsTask && repairTaskGapsAndPoll(gapDetailsTask)"');
    expect(source).toContain(':tasks="tasks"');
    expect(gapDetailsModalSource).toContain('import MarketRepairResultTags from "@/components/research/MarketRepairResultTags.vue";');
    expect(gapDetailsModalSource).toContain('<MarketRepairResultTags :result="repairResult" :tasks="tasks" />');
    expect(gapDetailsModalSource).toContain("@click=\"emit('repair')\"");
  });

  it("keeps chart gap repair results visible in the research toolbar", () => {
    expect(source).toContain('import MarketRepairResultTags from "@/components/research/MarketRepairResultTags.vue";');
    expect(source).toContain("chartGapRepairResult");
    expect(source).toContain("refreshChartCandles");
    expect(source).toContain('@click="repairFirstChartGap"');
    expect(source).toContain("startRepairPollingForResult(chartGapRepairResult.value, { immediate: false })");
    expect(source).toContain('<MarketRepairResultTags :candle-result="candleResult" :result="chartGapRepairResult" :tasks="tasks" />');
  });

  it("keeps task invalid issue details reachable from the research page", () => {
    expect(source).toContain('import ResearchTaskInvalidIssueModal from "@/components/research/ResearchTaskInvalidIssueModal.vue";');
    expect(source).toContain('@view-invalid="viewTaskInvalidIssues"');
    expect(source).toContain('<ResearchTaskInvalidIssueModal ref="invalidIssueModal" :tasks="tasks" @repaired="startRepairPollingForResult" />');
    expect(invalidIssueModalSource).toContain('import MarketRepairResultTags from "@/components/research/MarketRepairResultTags.vue";');
    expect(invalidIssueModalSource).toContain('<MarketRepairResultTags :result="repairResult" :tasks="tasks" />');
  });

  it("shows CandleProvider invalid issue reasons in the chart status strip", () => {
    expect(source).toContain("const { t, te } = useI18n();");
    expect(source).toContain("candleIssueReason");
    expect(source).toContain("invalidIssueLabel(firstCandleIssue.value)");
    expect(source).toContain("research.candleIssueNoTime");
    expect(source).toContain("reason: candleIssueReason.value");
    expect(source).toContain("te(key) ? t(key) : issue.message");
    expect(source).not.toContain('t("research.candleIssue", {\n    time:');
    for (const messages of [zhResearchMessages, enResearchMessages]) {
      expect(messages).toContain('"research.candleIssueNoTime"');
      expect(messages).toContain('"research.invalidCandleIssue.invalid_native_series"');
      expect(messages).toContain('"research.invalidCandleIssue.invalid_aggregation_base_series"');
    }
  });

  it("lets users queue repair for the first CandleProvider invalid issue", () => {
    expect(source).toContain('import ChartInvalidIssueRepairAction from "@/components/research/ChartInvalidIssueRepairAction.vue";');
    expect(source).toContain("<ChartInvalidIssueRepairAction");
    expect(source).toContain(':interval="candleResult?.baseInterval || interval"');
    expect(source).toContain(':candle-result="candleResult"');
    expect(source).toContain(':issue="firstCandleIssue"');
    expect(source).toContain(':load-candles="loadCandles"');
    expect(source).toContain(':load-tasks="loadTasks"');
    expect(source).toContain('@repaired="(result) => startRepairPollingForResult(result, { immediate: false })"');
    expect(chartInvalidIssueRepairSource).toContain('import { repairChartInvalidIssue } from "@/composables/researchInvalidIssueRepairActions";');
    expect(chartInvalidIssueRepairSource).toContain("repairResult");
    expect(chartInvalidIssueRepairSource).toContain("repairLoading");
    expect(chartInvalidIssueRepairSource).toContain("canRepair");
    expect(chartInvalidIssueRepairSource).toContain('v-if="canRepair"');
    expect(chartInvalidIssueRepairSource).toContain('@click="repairInvalidIssue"');
    expect(chartInvalidIssueRepairSource).toContain("research.repairFirstInvalidIssue");
    expect(chartInvalidIssueRepairSource).toContain("normalizeSymbolInput(props.symbol)");
    expect(chartInvalidIssueRepairSource).toContain('<MarketRepairResultTags :candle-result="candleResult" :result="repairResult" :tasks="tasks" />');
    for (const messages of [zhResearchMessages, enResearchMessages]) {
      expect(messages).toContain('"research.repairFirstInvalidIssue"');
    }
  });

  it("starts bounded repair task status polling from every repair entrypoint", () => {
    expect(source).toContain('import { useResearchRepairTaskPolling } from "@/composables/useResearchRepairTaskPolling";');
    expect(source).toContain("const { startRepairTaskPolling } = useResearchRepairTaskPolling(loadTasks);");
    expect(source).toContain("function startRepairPollingForResult(result: DataSyncGapRepairResult");
    expect(source).toContain("repairTaskIds: result.createdTasks.map((task) => task.id)");
    expect(source).toContain("onSettled: loadCandles");
    expect(source).toContain("onExhausted: loadCandles");
    expect(source).toContain('<MarketCandleGapTag :exchange="exchange" :interval="interval" :symbol="symbol" :tasks="tasks" @repaired="startRepairPollingForResult" />');
    expect(source).toContain('@repair-gaps="repairTaskGapsAndPoll"');
    expect(source).toContain("async function repairTaskGapsAndPoll(task: DataSyncTask)");
    expect(source).toContain("startRepairPollingForResult(taskGapRepairResult.value, { immediate: false })");
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
