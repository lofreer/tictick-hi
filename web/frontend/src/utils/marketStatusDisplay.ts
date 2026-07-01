export type Translate = (key: string, named?: Record<string, string | number>) => string;

export type MarketCatalogStatus = "active" | "inactive" | "missing" | "unknown";

const detailKeyByValue = new Map<string, string>([
  ["active", "research.marketStatusDetail.active"],
  ["inactive", "research.marketStatusDetail.inactive"],
  ["missing", "research.marketStatusDetail.missing"],
  ["trading", "research.marketStatusDetail.trading"],
  ["live", "research.marketStatusDetail.trading"],
  ["break", "research.marketStatusDetail.paused"],
  ["suspend", "research.marketStatusDetail.paused"],
  ["suspended", "research.marketStatusDetail.paused"],
  ["not_returned", "research.marketStatusDetail.notReturned"],
]);

export function marketStatusLabel(
  t: Translate,
  status: MarketCatalogStatus,
  detail?: string,
) {
  const base = marketStatusBaseLabel(t, status);
  const normalizedDetail = normalizeMarketStatusDetail(detail);
  if (!shouldShowMarketStatusDetail(status, normalizedDetail)) {
    return base;
  }
  return `${base} · ${marketStatusDetailLabel(t, normalizedDetail)}`;
}

export function marketStatusBaseLabel(t: Translate, status: MarketCatalogStatus) {
  if (status === "unknown") return t("research.marketStatus.unknown");
  return t(`research.marketStatus.${status}`);
}

export function marketStatusExchangeDetail(t: Translate, detail?: string) {
  const normalizedDetail = normalizeMarketStatusDetail(detail);
  if (!normalizedDetail) return "";
  return t("strategy.marketCatalogExchangeStatus", {
    status: marketStatusDetailLabel(t, normalizedDetail),
  });
}

export function marketStatusDetailLabel(t: Translate, detail?: string) {
  const normalizedDetail = normalizeMarketStatusDetail(detail);
  if (!normalizedDetail) return "";
  const key = detailKeyByValue.get(normalizedDetail.toLowerCase());
  return key ? t(key) : normalizedDetail;
}

function shouldShowMarketStatusDetail(status: MarketCatalogStatus, detail: string) {
  if (!detail) return false;
  const normalized = detail.toLowerCase();
  if (normalized === status) return false;
  if (status === "unknown") return false;
  return true;
}

function normalizeMarketStatusDetail(detail?: string) {
  return (detail ?? "").trim();
}
