#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLAN="$ROOT_DIR/docs/implementation-plan.md"
FAILED=0

require_text() {
  local text="$1"
  if ! rg -qF "$text" "$PLAN"; then
    echo "FAIL implementation plan is missing required boundary text: $text"
    FAILED=1
  fi
}

reject_text() {
  local text="$1"
  if rg -qF "$text" "$PLAN"; then
    echo "FAIL implementation plan still contains obsolete open question text: $text"
    FAILED=1
  fi
}

require_text "当前本地 demo 边界已收敛："
require_text "目标环境 / 生产阶段保留项："
require_text "生产通知启用："
require_text "生产密钥治理："
require_text "生产级回测撮合："

reject_text "第一版具体支持哪些 K 线周期"
reject_text "数据同步实时方式：WebSocket、轮询，还是交易所差异化"
reject_text "TradingView 开源图表具体采用哪个包"
reject_text '是否保留现有 `tictick-hi` Go + Vue 结构'
reject_text "风险默认值仍需随 live executor 阶段继续复核"

if [ "$FAILED" -ne 0 ]; then
  exit 1
fi

echo "plan boundary check passed"
