import type { CandleCoverage, CandleResult } from "@/types/app";

type CoverageTagType = "default" | "warning";
type Translate = (key: string, values?: Record<string, number>) => string;
type CandleCoverageWithBaseCounts = CandleCoverage & {
  requiredBaseCandles: number;
  returnedBaseCandles: number;
};

export function shouldShowCandleCoverage(result: CandleResult | null) {
  if (!result) return false;
  return result.coverage.limitedByBaseWindow || result.health !== "ok";
}

export function hasBaseCandleCoverage(coverage: CandleCoverage): coverage is CandleCoverageWithBaseCounts {
  return typeof coverage.requiredBaseCandles === "number" && typeof coverage.returnedBaseCandles === "number";
}

export function candleCoverageTagType(result: CandleResult | null): CoverageTagType {
  return result?.coverage.limitedByBaseWindow || result?.health !== "ok" ? "warning" : "default";
}

export function candleCoverageLabel(result: CandleResult | null, t: Translate) {
  const coverage = result?.coverage;
  if (!coverage) return "";
  const primary = t(coverage.limitedByBaseWindow ? "research.coverageLimited" : "research.coverageSummary", {
    requested: coverage.requestedLimit,
    returned: coverage.returnedCandles,
  });
  if (!hasBaseCandleCoverage(coverage)) return primary;
  return `${primary} / ${t("research.baseCoverage", {
    required: coverage.requiredBaseCandles,
    returned: coverage.returnedBaseCandles,
  })}`;
}
