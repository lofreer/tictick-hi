const minCanvasScaleTolerance = 0.25;
const canvasScaleToleranceRatio = 0.35;
const minCanvasScaleSkewTolerance = 0.2;
const canvasScaleSkewToleranceRatio = 0.12;
const dprTolerance = 0.01;

export function currentDevicePixelRatio() {
  return Number.isFinite(window.devicePixelRatio) && window.devicePixelRatio > 0 ? window.devicePixelRatio : 1;
}

export function devicePixelRatioChanged(previous: number, next = currentDevicePixelRatio()) {
  return Math.abs(previous - next) > dprTolerance;
}

export function hasDistortedChartCanvasGeometry(container: ParentNode | null, expectedDevicePixelRatio = currentDevicePixelRatio()) {
  if (!container) return false;
  const expectedScale = Math.max(expectedDevicePixelRatio, 0.01);
  return Array.from(container.querySelectorAll("canvas")).some((canvas) => {
    const bounds = canvas.getBoundingClientRect();
    if (bounds.width <= 0 || bounds.height <= 0 || canvas.width <= 0 || canvas.height <= 0) return false;
    const scaleX = canvas.width / bounds.width;
    const scaleY = canvas.height / bounds.height;
    return (
      isUnexpectedCanvasScale(scaleX, expectedScale) ||
      isUnexpectedCanvasScale(scaleY, expectedScale) ||
      Math.abs(scaleX - scaleY) > canvasScaleSkewTolerance(expectedScale)
    );
  });
}

function isUnexpectedCanvasScale(scale: number, expectedScale: number) {
  return Math.abs(scale - expectedScale) > canvasScaleTolerance(expectedScale);
}

function canvasScaleTolerance(expectedScale: number) {
  return Math.max(minCanvasScaleTolerance, expectedScale * canvasScaleToleranceRatio);
}

function canvasScaleSkewTolerance(expectedScale: number) {
  return Math.max(minCanvasScaleSkewTolerance, expectedScale * canvasScaleSkewToleranceRatio);
}
