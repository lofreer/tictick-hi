import { positiveFloor } from "./chartSizing";

const dimensionProperties = [
  "width",
  "height",
  "max-width",
  "max-height",
  "inline-size",
  "block-size",
  "max-inline-size",
  "max-block-size",
] as const;

export function repairDistortedChartCanvases(
  host: HTMLElement,
  fixedSize: { width: number; height: number },
  devicePixelRatio: number,
) {
  const hostBounds = host.getBoundingClientRect();
  const hostWidth = boundedHostSize(positiveFloor(hostBounds.width), fixedSize.width);
  const hostHeight = boundedHostSize(positiveFloor(hostBounds.height), fixedSize.height);
  if (hostWidth <= 0 || hostHeight <= 0) return false;

  let repaired = false;
  for (const element of chartInternalElements(host)) {
    const canvasDistorted =
      element instanceof HTMLCanvasElement && isCanvasDistorted(element, hostWidth, hostHeight, devicePixelRatio);
    if (!canvasDistorted && !hasPollutedInlineSize(element, hostWidth, hostHeight)) continue;
    clearInlineSizeLocks(element);
    repaired = true;
  }
  return repaired;
}

function chartInternalElements(host: HTMLElement) {
  return host.querySelectorAll<HTMLElement | HTMLCanvasElement>(
    ".tv-lightweight-charts, .tv-lightweight-charts table, .tv-lightweight-charts tbody, .tv-lightweight-charts tr, .tv-lightweight-charts td, canvas",
  );
}

function isCanvasDistorted(
  canvas: HTMLCanvasElement,
  hostWidth: number,
  hostHeight: number,
  devicePixelRatio: number,
) {
  const maximumScale = Math.max(2.25, Math.min(4, Math.ceil(devicePixelRatio || 1) + 0.25));
  const bounds = canvas.getBoundingClientRect();
  const scaleX = canvas.width / Math.max(1, bounds.width);
  const scaleY = canvas.height / Math.max(1, bounds.height);
  return (
    bounds.width > hostWidth + 1 ||
    bounds.height > hostHeight + 1 ||
    scaleX < 0.75 ||
    scaleX > maximumScale ||
    scaleY < 0.75 ||
    scaleY > maximumScale ||
    Math.abs(scaleX - scaleY) > 0.2
  );
}

function hasPollutedInlineSize(element: HTMLElement, hostWidth: number, hostHeight: number) {
  for (const property of dimensionProperties) {
    const limit = property.includes("width") || property.includes("inline") ? hostWidth : hostHeight;
    const size = readInlinePixelValue(element.style.getPropertyValue(property));
    if (size !== null && size > limit + 1) return true;
  }
  return false;
}

function clearInlineSizeLocks(element: HTMLElement) {
  for (const property of dimensionProperties) {
    element.style.removeProperty(property);
  }
}

function readInlinePixelValue(value: string) {
  const size = Number.parseFloat(value);
  if (!Number.isFinite(size) || size <= 0) return null;
  return size;
}

function boundedHostSize(measuredSize: number | null, fixedSize: number) {
  if (fixedSize > 0 && measuredSize !== null) return Math.min(fixedSize, measuredSize);
  if (fixedSize > 0) return fixedSize;
  return measuredSize ?? 0;
}
