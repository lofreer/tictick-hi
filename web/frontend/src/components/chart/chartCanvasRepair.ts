import { positiveFloor } from "./chartSizing";

const healthyCanvasSizes = new WeakMap<HTMLCanvasElement, { width: number; height: number }>();

export function repairDistortedChartCanvases(
  host: HTMLElement,
  fixedSize: { width: number; height: number },
  devicePixelRatio: number,
) {
  const hostBounds = host.getBoundingClientRect();
  const hostWidth = boundedHostSize(positiveFloor(hostBounds.width), fixedSize.width);
  const hostHeight = boundedHostSize(positiveFloor(hostBounds.height), fixedSize.height);
  if (hostWidth <= 0 || hostHeight <= 0) return;

  for (const canvas of host.querySelectorAll("canvas")) {
    repairCanvasElement(canvas, hostWidth, hostHeight, devicePixelRatio);
  }
}

function repairCanvasElement(
  canvas: HTMLCanvasElement,
  hostWidth: number,
  hostHeight: number,
  devicePixelRatio: number,
) {
  const bounds = canvas.getBoundingClientRect();
  const scaleX = canvas.width / Math.max(1, bounds.width);
  const scaleY = canvas.height / Math.max(1, bounds.height);
  const distorted =
    bounds.width > hostWidth + 1 ||
    bounds.height > hostHeight + 1 ||
    scaleX < 0.75 ||
    scaleX > 2.25 ||
    scaleY < 0.75 ||
    scaleY > 2.25 ||
    Math.abs(scaleX - scaleY) > 0.2;
  if (!distorted) {
    const width = positiveFloor(bounds.width);
    const height = positiveFloor(bounds.height);
    if (width !== null && height !== null) healthyCanvasSizes.set(canvas, { width, height });
    return;
  }

  const pixelRatio = Math.max(1, Math.round(devicePixelRatio || 1));
  const healthySize = healthyCanvasSizes.get(canvas);
  const width = healthySize?.width ?? canvasCssSize(canvas.width, pixelRatio, hostWidth);
  const height = healthySize?.height ?? canvasCssSize(canvas.height, pixelRatio, hostHeight);
  canvas.style.width = `${width}px`;
  canvas.style.height = `${height}px`;
  canvas.style.maxWidth = `${hostWidth}px`;
  canvas.style.maxHeight = `${hostHeight}px`;
}

function canvasCssSize(bitmapSize: number, pixelRatio: number, hostSize: number) {
  const size = Math.round(bitmapSize / pixelRatio);
  if (!Number.isFinite(size) || size <= 0) return Math.max(1, hostSize);
  return Math.min(Math.max(1, size), hostSize);
}

function boundedHostSize(measuredSize: number | null, fixedSize: number) {
  if (fixedSize > 0 && measuredSize !== null) return Math.min(fixedSize, measuredSize);
  if (fixedSize > 0) return fixedSize;
  return measuredSize ?? 0;
}
