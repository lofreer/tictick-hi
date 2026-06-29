#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

RESEARCH_PAGE="$ROOT_DIR/web/frontend/src/pages/ResearchPage.vue"
RESEARCH_CSS="$ROOT_DIR/web/frontend/src/pages/ResearchPage.css"
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
require_contains "$RESEARCH_PAGE" 'class="research-chart-body" data-chart-viewport="fixed"'
require_not_contains "$RESEARCH_PAGE" 'class="surface chart-panel research-chart-panel"'

require_contains "$RESEARCH_CSS" ".research-workspace"
require_contains "$RESEARCH_CSS" "overflow-x: clip;"
require_contains "$RESEARCH_CSS" ".research-tasks-panel"
require_contains "$RESEARCH_CSS" "width: 100%;"
require_contains "$RESEARCH_CSS" "max-width: 100%;"
require_contains "$RESEARCH_CSS" "min-width: 0;"
require_contains "$RESEARCH_CSS" "max-height: min(260px, 28vh);"
require_contains "$RESEARCH_CSS" "max-height: min(260px, 28dvh);"
require_contains "$RESEARCH_CSS" "overflow: auto;"
require_contains "$RESEARCH_CSS" ".research-chart-panel"
require_contains "$RESEARCH_CSS" "--research-chart-viewport-height:"
require_contains "$RESEARCH_CSS" "display: flex;"
require_contains "$RESEARCH_CSS" "flex-direction: column;"
require_contains "$RESEARCH_CSS" "contain: layout paint;"
require_contains "$RESEARCH_CSS" "flex: 1 1 620px;"
require_contains "$RESEARCH_CSS" "flex: 0 1 680px;"
require_contains "$RESEARCH_CSS" "width: clamp(180px, 22vw, 360px);"
require_contains "$RESEARCH_CSS" ".research-chart-body"
require_contains "$RESEARCH_CSS" "flex: 0 0 var(--research-chart-viewport-height);"
require_contains "$RESEARCH_CSS" "height: var(--research-chart-viewport-height) !important;"
require_contains "$RESEARCH_CSS" "max-height: var(--research-chart-viewport-height) !important;"
require_block_contains "$RESEARCH_CSS" ".research-chart-body" "overflow: hidden;"
require_block_contains "$RESEARCH_CSS" ".research-chart-body" "contain: layout paint;"
require_block_not_contains "$RESEARCH_CSS" ".research-chart-body" "contain: strict;"
require_block_contains "$RESEARCH_CSS" ".research-chart-body" "--tt-chart-inline-end-gutter: 28px;"
require_block_contains "$RESEARCH_CSS" ".research-chart-body" "--tt-chart-block-end-gutter: 12px;"
require_block_contains "$RESEARCH_CSS" ".research-chart-body .trading-chart" "width: calc(100% - var(--tt-chart-inline-end-gutter, 0px)) !important;"
require_block_contains "$RESEARCH_CSS" ".research-chart-body .trading-chart" "height: calc(100% - var(--tt-chart-block-end-gutter, 0px)) !important;"
require_contains "$RESEARCH_CSS" "--tt-chart-inline-end-gutter: 18px;"
require_contains "$RESEARCH_CSS" "--tt-chart-block-end-gutter: 10px;"
require_contains "$RESEARCH_CSS" "--research-chart-viewport-height: clamp(280px, calc(100vh - 620px), 560px);"
require_contains "$RESEARCH_CSS" "--research-chart-viewport-height: clamp(280px, calc(100dvh - 620px), 560px);"
require_contains "$RESEARCH_CSS" "--research-chart-viewport-height: clamp(240px, calc(100vh - 680px), 480px);"
require_contains "$RESEARCH_CSS" "--research-chart-viewport-height: clamp(240px, calc(100dvh - 680px), 480px);"

require_block_contains "$CHART_CSS" ".trading-chart" "position: relative;"
require_block_contains "$CHART_CSS" ".trading-chart" "width: calc(100% - var(--tt-chart-inline-end-gutter, 0px));"
require_block_contains "$CHART_CSS" ".trading-chart" "height: calc(100% - var(--tt-chart-block-end-gutter, 0px));"
require_block_contains "$CHART_CSS" ".trading-chart" "max-inline-size: 100%;"
require_block_contains "$CHART_CSS" ".trading-chart" "max-block-size: 100%;"
require_block_contains "$CHART_CSS" ".trading-chart" "overflow: hidden;"
require_block_contains "$CHART_CSS" ".trading-chart" "contain: layout style;"

require_block_contains "$CHART_CSS" ".trading-chart__canvas" "position: absolute;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas" "inset: 0;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas" "width: 100%;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas" "height: 100%;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas" "overflow: hidden;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas" "contain: layout style;"
require_block_not_contains "$CHART_CSS" ".trading-chart__canvas" "!important"

require_block_not_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts" "max-width"
require_block_not_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts" "max-inline-size"
require_block_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts" "block-size: 100% !important;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts" "height: 100% !important;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts" "max-block-size: 100% !important;"
require_block_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts" "max-height: 100% !important;"
require_block_not_contains "$CHART_CSS" ".trading-chart__canvas > .tv-lightweight-charts" "overflow:"
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
require_contains "$CHART_SMOKE" "requireInitialChartFit: true"
require_contains "$CHART_SMOKE" "polluteInternalChartHeights"
require_contains "$CHART_SMOKE" "'.tv-lightweight-charts tbody'"
require_contains "$CHART_SMOKE" "'.tv-lightweight-charts tr'"
require_contains "$CHART_SMOKE" "'.tv-lightweight-charts td'"
require_contains "$CHART_SMOKE" "missing bounded main pane canvas"
require_contains "$CHART_SMOKE" "chart main pane canvas is clipped by fixed body"
require_contains "$CHART_SMOKE" "main pane has no visible candle pixels"
require_contains "$CHART_SMOKE" "research chart panel must not inherit the global chart-panel sizing contract"
require_contains "$CHART_SMOKE" "chart bottom axis is clipped from the initial viewport"
require_contains "$CHART_SMOKE" "page overflowed horizontally and can clip the chart viewport"
require_contains "$CHART_SMOKE" "time-axis label touches fixed body edge"
require_contains "$CHART_SMOKE" "does not match configured fixed body inset"
require_contains "$CHART_SMOKE" "maxRetries: 5"

echo "research chart layout check passed"
