import { appColors } from "@/theme/tokens";
import type { CandleGap, CandleResult, ChartCandle, ChartMarker } from "@/types/app";

type Translate = (key: string, named?: Record<string, unknown>) => string;

export function researchChartGapMarkers(candles: ChartCandle[], result: CandleResult | null, t: Translate) {
  return candleGapMarkers(candles, result?.gaps ?? [], (missingCandles) =>
    t("research.chartGapMarker", { missing: missingCandles }),
  );
}

export function candleGapMarkers(
  candles: ChartCandle[],
  gaps: CandleGap[],
  formatMarkerText: (missingCandles: number) => string,
): ChartMarker[] {
  if (candles.length === 0 || gaps.length === 0) {
    return [];
  }

  const orderedCandles = [...candles].sort((left, right) => left.time - right.time);
  return gaps.flatMap((gap, index) => {
    const anchor = gapMarkerAnchor(orderedCandles, gap);
    if (!anchor) {
      return [];
    }

    return [{
      id: `candle-gap-${gap.from}-${gap.to}-${index}`,
      time: anchor.time,
      position: "aboveBar" as const,
      shape: "square" as const,
      color: appColors.warning,
      text: formatMarkerText(gap.missingCandles),
      size: 1.1,
    }];
  });
}

function gapMarkerAnchor(candles: ChartCandle[], gap: CandleGap) {
  const gapFrom = utcSeconds(gap.from);
  const gapTo = utcSeconds(gap.to);
  if (gapFrom === null || gapTo === null) {
    return null;
  }

  const firstAfterGap = candles.find((candle) => candle.time >= gapTo);
  if (firstAfterGap) {
    return firstAfterGap;
  }

  for (let index = candles.length - 1; index >= 0; index -= 1) {
    if (candles[index].time < gapFrom) {
      return candles[index];
    }
  }

  return candles[0] ?? null;
}

function utcSeconds(value: string) {
  const timestamp = Date.parse(value);
  if (!Number.isFinite(timestamp)) {
    return null;
  }
  return Math.floor(timestamp / 1000);
}
