#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

RESEARCH_PAGE="$ROOT_DIR/web/frontend/src/pages/ResearchPage.vue"
RESEARCH_CSS="$ROOT_DIR/web/frontend/src/pages/ResearchPage.css"
KLINE_CHART_CSS="$ROOT_DIR/web/frontend/src/pages/klineChartLayout.css"
DETAIL_CHART_CSS="$ROOT_DIR/web/frontend/src/pages/detailChartLayout.css"
CHART_CSS="$ROOT_DIR/web/frontend/src/components/chart/TradingViewChart.css"
CHART_SMOKE="$ROOT_DIR/scripts/research-chart-height-smoke.mjs"

fail() {
  echo "research chart layout check failed: $1" >&2
  exit 1
}

require_contains() {
  local file="$1"
  local needle="$2"
  if ! grep -Fq -- "$needle" "$file"; then
    fail "$file must contain: $needle"
  fi
}

require_not_contains() {
  local file="$1"
  local needle="$2"
  if grep -Fq -- "$needle" "$file"; then
    fail "$file must not contain: $needle"
  fi
}

require_order() {
  local file="$1"
  local first="$2"
  local second="$3"
  local first_line
  local second_line

  first_line="$(grep -nF "$first" "$file" | head -n 1 | cut -d: -f1 || true)"
  second_line="$(grep -nF "$second" "$file" | head -n 1 | cut -d: -f1 || true)"
  if [ -z "$first_line" ] || [ -z "$second_line" ] || [ "$first_line" -ge "$second_line" ]; then
    fail "$file must place '$first' before '$second'"
  fi
}

css_block() {
  local file="$1"
  local selector="$2"
  awk -v selector="$selector" '
    index($0, selector " {") { found = 1 }
    found { print }
    found && index($0, "}") { exit }
  ' "$file"
}

require_block_contains() {
  local file="$1"
  local selector="$2"
  local needle="$3"
  if ! css_block "$file" "$selector" | grep -Fq -- "$needle"; then
    fail "$file block $selector must contain: $needle"
  fi
}

require_block_not_contains() {
  local file="$1"
  local selector="$2"
  local needle="$3"
  if css_block "$file" "$selector" | grep -Fq -- "$needle"; then
    fail "$file block $selector must not contain: $needle"
  fi
}

require_order "$RESEARCH_PAGE" 'class="surface research-tasks-panel"' 'class="surface research-chart-panel"'
require_contains "$RESEARCH_PAGE" 'import "./klineChartLayout.css";'
require_contains "$RESEARCH_PAGE" 'class="kline-chart-frame research-chart-body"'
require_contains "$RESEARCH_PAGE" 'class="kline-chart-frame__viewport research-chart-viewport" data-chart-viewport="fixed"'
require_contains "$RESEARCH_PAGE" ':show-sync-button="false"'
require_not_contains "$RESEARCH_PAGE" 'class="surface chart-panel research-chart-panel"'

require_contains "$RESEARCH_CSS" ".research-workspace"
require_contains "$RESEARCH_CSS" "overflow-x: clip;"
require_contains "$RESEARCH_CSS" ".research-tasks-panel"
require_contains "$RESEARCH_CSS" "width: 100%;"
require_contains "$RESEARCH_CSS" "max-width: 100%;"
require_contains "$RESEARCH_CSS" "min-width: 0;"
require_contains "$RESEARCH_CSS" "max-height: clamp(156px, 18vh, 188px);"
require_contains "$RESEARCH_CSS" "overflow: auto;"
require_contains "$RESEARCH_CSS" ".research-chart-panel"
require_contains "$RESEARCH_CSS" "display: flex;"
require_contains "$RESEARCH_CSS" "flex-direction: column;"
require_contains "$RESEARCH_CSS" "contain: layout paint;"
require_contains "$RESEARCH_CSS" ".research-toolbar-main"
require_contains "$RESEARCH_CSS" ".research-source-controls"
require_block_contains "$RESEARCH_CSS" ".research-source-controls" "display: grid;"
require_block_contains "$RESEARCH_CSS" ".research-source-controls" "grid-template-columns: 104px 112px 30px 58px max-content;"
require_contains "$RESEARCH_CSS" "width: max-content;"
require_contains "$RESEARCH_CSS" "overflow-x: auto;"
require_contains "$RESEARCH_CSS" ".research-toolbar-status"
require_contains "$RESEARCH_CSS" ".research-current-source"
require_block_contains "$RESEARCH_CSS" ".research-toolbar" "grid-template-columns: max-content minmax(0, 1fr);"
require_block_contains "$RESEARCH_CSS" ".research-toolbar" "padding: 8px 12px;"
require_block_contains "$RESEARCH_CSS" ".research-toolbar-status" "justify-content: flex-start;"
require_block_contains "$RESEARCH_CSS" ".research-toolbar-status" "flex-wrap: nowrap;"
require_block_contains "$RESEARCH_CSS" ".research-toolbar-status" "overflow-x: auto;"
require_not_contains "$RESEARCH_CSS" "flex: 1 1 620px;"
require_not_contains "$RESEARCH_CSS" "flex: 0 1 680px;"
require_not_contains "$RESEARCH_CSS" "width: clamp(180px, 22vw, 360px);"
require_not_contains "$RESEARCH_CSS" "width: clamp(180px, 15vw, 240px);"
require_not_contains "$RESEARCH_CSS" "grid-template-columns: 128px clamp(180px, 18vw, 300px) 84px auto;"
require_not_contains "$RESEARCH_CSS" "width: fit-content;"
require_not_contains "$RESEARCH_PAGE" 'class="toolbar-row"'
require_contains "$RESEARCH_CSS" ".research-chart-body"
require_contains "$RESEARCH_CSS" "flex: 0 0 var(--kline-chart-frame-height);"
require_block_not_contains "$RESEARCH_CSS" ".research-chart-body" "--kline-chart-plot-height:"
require_block_not_contains "$RESEARCH_CSS" ".research-chart-body" "--kline-chart-padding-left:"
require_block_not_contains "$RESEARCH_CSS" ".research-chart-body" "--kline-chart-padding-right:"
require_not_contains "$DETAIL_CHART_CSS" "--kline-chart-plot-height:"
require_not_contains "$DETAIL_CHART_CSS" "--kline-chart-padding-left:"
require_not_contains "$DETAIL_CHART_CSS" "--kline-chart-padding-right:"
require_contains "$KLINE_CHART_CSS" "--kline-chart-plot-height: clamp(680px, 72dvh, 820px);"
require_contains "$KLINE_CHART_CSS" "--kline-chart-frame-height:"
require_block_contains "$KLINE_CHART_CSS" ".kline-chart-frame" "height: var(--kline-chart-frame-height);"
require_block_contains "$KLINE_CHART_CSS" ".kline-chart-frame" "max-height: var(--kline-chart-frame-height);"
require_block_contains "$KLINE_CHART_CSS" ".kline-chart-frame" "var(--kline-chart-padding-right)"
require_block_contains "$KLINE_CHART_CSS" ".kline-chart-frame" "var(--kline-chart-padding-left)"
require_block_contains "$KLINE_CHART_CSS" ".kline-chart-frame" "overflow: hidden;"
require_block_contains "$KLINE_CHART_CSS" ".kline-chart-frame" "contain: layout paint;"
require_block_not_contains "$RESEARCH_CSS" ".research-chart-body" "contain: strict;"
require_contains "$RESEARCH_CSS" ".research-chart-viewport"
require_block_contains "$KLINE_CHART_CSS" ".kline-chart-frame__viewport" "height: var(--kline-chart-plot-height);"
require_block_contains "$KLINE_CHART_CSS" ".kline-chart-frame__viewport" "max-height: var(--kline-chart-plot-height);"
require_block_contains "$KLINE_CHART_CSS" ".kline-chart-frame__viewport" "overflow: hidden;"
require_block_contains "$KLINE_CHART_CSS" ".kline-chart-frame__viewport .trading-chart" "width: 100%;"
require_block_contains "$KLINE_CHART_CSS" ".kline-chart-frame__viewport .trading-chart" "height: 100%;"
require_not_contains "$RESEARCH_CSS" "--tt-chart-inline-end-gutter"
require_not_contains "$RESEARCH_CSS" "--tt-chart-block-end-gutter"
require_contains "$KLINE_CHART_CSS" "--kline-chart-plot-height: clamp(680px, 72dvh, 820px);"
require_contains "$KLINE_CHART_CSS" "--kline-chart-plot-height: 700px;"
require_contains "$KLINE_CHART_CSS" "--kline-chart-plot-height: 580px;"
require_contains "$KLINE_CHART_CSS" "--kline-chart-padding-left: 14px;"
require_contains "$KLINE_CHART_CSS" "--kline-chart-padding-left: 12px;"
require_contains "$KLINE_CHART_CSS" "--kline-chart-padding-left: 10px;"
require_contains "$KLINE_CHART_CSS" "--kline-chart-padding-right: 2px;"
require_contains "$RESEARCH_CSS" "width: 112px;"
require_contains "$RESEARCH_CSS" "max-width: min(220px, 24vw);"
require_not_contains "$RESEARCH_CSS" "--kline-chart-plot-height: clamp(720px, 78vh, 900px);"
require_not_contains "$RESEARCH_CSS" "grid-template-columns: 100px 132px 28px 56px max-content;"
require_not_contains "$RESEARCH_CSS" "width: 132px;"

require_block_contains "$CHART_CSS" ".trading-chart" "position: relative;"
require_block_contains "$CHART_CSS" ".trading-chart" "width: 100%;"
require_block_contains "$CHART_CSS" ".trading-chart" "height: 100%;"
require_block_contains "$CHART_CSS" ".trading-chart" "max-inline-size: 100%;"
require_block_contains "$CHART_CSS" ".trading-chart" "max-block-size: 100%;"
require_block_contains "$CHART_CSS" ".trading-chart" "overflow: hidden;"
require_block_contains "$CHART_CSS" ".trading-chart" "contain: layout style;"

require_block_contains "$CHART_CSS" ".trading-chart__canvas" "position: absolute;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas" "top: 0;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas" "left: 0;"
require_block_not_contains "$CHART_CSS" ".trading-chart__canvas" "inset: 0;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas" "width: 100%;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas" "height: 100%;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas" "overflow: hidden;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas" "contain: layout style;"
require_block_not_contains "$CHART_CSS" ".trading-chart__canvas" "!important"

require_block_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts" "inline-size: 100% !important;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts" "width: 100% !important;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts" "max-inline-size: 100% !important;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts" "max-width: 100% !important;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts" "block-size: 100% !important;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts" "height: 100% !important;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts" "max-block-size: 100% !important;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts" "max-height: 100% !important;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts" "overflow: hidden;"
require_block_not_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts" "contain:"
require_not_contains "$CHART_CSS" "--tt-chart-render-width"
require_not_contains "$CHART_CSS" "--tt-chart-render-height"

require_not_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts table"
require_not_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts tbody"
require_not_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts tr"
require_not_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts td"
require_block_not_contains "$CHART_CSS" ".trading-chart__canvas canvas" "--tt-chart-render-width"
require_block_not_contains "$CHART_CSS" ".trading-chart__canvas canvas" "--tt-chart-render-height"

require_contains "$CHART_SMOKE" "narrow-desktop-812x1320"
require_contains "$CHART_SMOKE" "desktop-2048x1152"
require_contains "$CHART_SMOKE" "polluteInternalChartHeights"
require_contains "$CHART_SMOKE" "'.tv-lightweight-charts tbody'"
require_contains "$CHART_SMOKE" "'.tv-lightweight-charts tr'"
require_contains "$CHART_SMOKE" "'.tv-lightweight-charts td'"
require_contains "$CHART_SMOKE" "missing bounded main pane canvas"
require_contains "$CHART_SMOKE" "chart main pane canvas is clipped by fixed body"
require_contains "$CHART_SMOKE" "main chart pane is detached from the right price-axis"
require_contains "$CHART_SMOKE" "main pane has no visible candle pixels"
require_contains "$CHART_SMOKE" "chart plot is too short for the viewport"
require_contains "$CHART_SMOKE" "research chart panel must not inherit the global chart-panel sizing contract"
require_contains "$CHART_SMOKE" "chart left side"
require_contains "$CHART_SMOKE" "page overflowed horizontally and can clip the chart viewport"
require_contains "$CHART_SMOKE" "time-axis label touches fixed body edge"
require_contains "$CHART_SMOKE" "does not match configured fixed body inset"
require_contains "$CHART_SMOKE" "maxRetries: 5"
require_contains "$ROOT_DIR/scripts/stage8-visual-smoke.mjs" "SMOKE_MAX_CHART_EDGE_GAP"
require_contains "$ROOT_DIR/scripts/stage8-visual-smoke.mjs" "maxChartEdgeGap"
require_contains "$ROOT_DIR/scripts/stage8-visual-smoke.mjs" "right price-axis does not sit on the chart viewport edge"

echo "research chart layout check passed"
