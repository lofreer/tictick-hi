import { describe, expect, it } from "vitest";

import { repairDistortedChartCanvases } from "./chartCanvasRepair";

describe("repairDistortedChartCanvases", () => {
  it("repairs polluted canvas dimensions using fixed viewport bounds", () => {
    const { canvas, host } = chartFixture({
      canvasBitmap: { width: 60, height: 652 },
      canvasRect: { width: 600, height: 9000 },
      hostRect: { width: 1180, height: 3841 },
    });

    repairDistortedChartCanvases(host, { width: 1180, height: 641 }, 1);

    expect(canvas.style.width).toBe("60px");
    expect(canvas.style.height).toBe("641px");
    expect(canvas.style.maxWidth).toBe("1180px");
    expect(canvas.style.maxHeight).toBe("641px");
  });

  it("returns a polluted canvas to its last healthy size", () => {
    const fixture = chartFixture({
      canvasBitmap: { width: 60, height: 672 },
      canvasRect: { width: 52, height: 672 },
      hostRect: { width: 778, height: 700 },
    });

    repairDistortedChartCanvases(fixture.host, { width: 778, height: 700 }, 1);
    fixture.setCanvasRect({ width: 9000, height: 9000 });
    repairDistortedChartCanvases(fixture.host, { width: 778, height: 700 }, 1);

    expect(fixture.canvas.style.width).toBe("52px");
    expect(fixture.canvas.style.height).toBe("672px");
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
