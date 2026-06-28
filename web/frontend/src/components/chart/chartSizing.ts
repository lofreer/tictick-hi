export function readClientWidth(element: HTMLElement) {
  return element.clientWidth > 0 ? element.clientWidth : null;
}

export function readClientHeight(element: HTMLElement) {
  return element.clientHeight > 0 ? element.clientHeight : null;
}

export function readPixelSize(element: HTMLElement, property: "width" | "height" | "maxHeight") {
  const value = Number.parseFloat(window.getComputedStyle(element)[property]);
  if (!Number.isFinite(value) || value <= 0) return null;
  return value;
}

export function readChartGutter(style: CSSStyleDeclaration, property: string) {
  const value = Number.parseFloat(style.getPropertyValue(property));
  return Number.isFinite(value) && value > 0 ? value : 0;
}

export function positiveFloor(value: number) {
  const floored = Math.floor(value);
  return floored > 0 ? floored : null;
}
