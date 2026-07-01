import { describe, expect, it } from "vitest";

import { repairDistortedChartCanvases } from "./chartCanvasRepair";

describe("repairDistortedChartCanvases", () => {
  it("clears polluted canvas dimensions instead of sizing from a distorted bitmap", () => {
    const { canvas, host } = chartFixture({
      canvasBitmap: { width: 9000, height: 9000 },
      canvasRect: { width: 600, height: 9000 },
      hostRect: { width: 1180, height: 3841 },
    });
    polluteInlineSize(canvas);

    expect(repairDistortedChartCanvases(host, { width: 1180, height: 641 }, 1)).toBe(true);

    expect(canvas.style.width).toBe("");
    expect(canvas.style.height).toBe("");
    expect(canvas.style.maxWidth).toBe("");
    expect(canvas.style.maxHeight).toBe("");
  });

  it("leaves library-managed healthy canvas dimensions intact", () => {
    const fixture = chartFixture({
      canvasBitmap: { width: 104, height: 1282 },
      canvasRect: { width: 52, height: 672 },
      hostRect: { width: 778, height: 700 },
    });
    fixture.canvas.style.width = "52px";
    fixture.canvas.style.height = "641px";

    expect(repairDistortedChartCanvases(fixture.host, { width: 778, height: 700 }, 2)).toBe(false);

    expect(fixture.canvas.style.width).toBe("52px");
    expect(fixture.canvas.style.height).toBe("641px");
  });

  it("clears polluted lightweight-charts table geometry", () => {
    const fixture = chartFixture({
      canvasBitmap: { width: 104, height: 1282 },
      canvasRect: { width: 52, height: 641 },
      hostRect: { width: 778, height: 700 },
    });
    const table = document.createElement("table");
    table.style.height = "9000px";
    table.style.maxBlockSize = "9000px";
    const chartRoot = document.createElement("div");
    chartRoot.className = "tv-lightweight-charts";
    chartRoot.append(table, fixture.canvas);
    fixture.host.replaceChildren(chartRoot);

    expect(repairDistortedChartCanvases(fixture.host, { width: 778, height: 700 }, 2)).toBe(true);

    expect(table.style.height).toBe("");
    expect(table.style.maxBlockSize).toBe("");
  });
});

function chartFixture(options: {
  canvasBitmap: { width: number; height: number };
  canvasRect: { width: number; height: number };
  hostRect: { width: number; height: number };
}) {
  const host = document.createElement("div");
  const canvas = document.createElement("canvas");
  canvas.width = options.canvasBitmap.width;
  canvas.height = options.canvasBitmap.height;
  host.append(canvas);
  Object.assign(host, { getBoundingClientRect: () => rect(options.hostRect) });
  let canvasRect = options.canvasRect;
  Object.assign(canvas, { getBoundingClientRect: () => rect(canvasRect) });
  return {
    canvas,
    host,
    setCanvasRect(next: { width: number; height: number }) {
      canvasRect = next;
    },
  };
}

function polluteInlineSize(element: HTMLElement) {
  element.style.width = "9000px";
  element.style.height = "9000px";
  element.style.maxWidth = "9000px";
  element.style.maxHeight = "9000px";
}

function rect(size: { width: number; height: number }) {
  return {
    x: 0,
    y: 0,
    top: 0,
    left: 0,
    right: size.width,
    bottom: size.height,
    width: size.width,
    height: size.height,
    toJSON: () => ({}),
  } as DOMRect;
}
