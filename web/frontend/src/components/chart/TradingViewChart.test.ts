import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import TradingViewChart from "@/components/chart/TradingViewChart.vue";
import { createChart } from "lightweight-charts";

const chartMocks = vi.hoisted(() => ({
  createChart: vi.fn(),
  fitContent: vi.fn(),
  remove: vi.fn(),
  resize: vi.fn(),
  setData: vi.fn(),
  setMarkers: vi.fn(),
}));

vi.mock("lightweight-charts", () => ({
  CandlestickSeries: "CandlestickSeries",
  createChart: chartMocks.createChart,
  createSeriesMarkers: vi.fn(() => ({ setMarkers: chartMocks.setMarkers })),
}));

const mockedCreateChart = vi.mocked(createChart);

function mockChartApi() {
  chartMocks.createChart.mockReturnValue({
    addSeries: vi.fn(() => ({ setData: chartMocks.setData })),
    applyOptions: vi.fn(),
    remove: chartMocks.remove,
    resize: chartMocks.resize,
    timeScale: vi.fn(() => ({ fitContent: chartMocks.fitContent })),
  });
}

describe("TradingViewChart", () => {
  let observedTarget: Element | null = null;
  let originalGetBoundingClientRect: typeof Element.prototype.getBoundingClientRect;

  beforeEach(() => {
    setActivePinia(createPinia());
    observedTarget = null;
    originalGetBoundingClientRect = Element.prototype.getBoundingClientRect;
    vi.clearAllMocks();
    chartMocks.createChart.mockReset();
    mockChartApi();

    window.requestAnimationFrame = ((callback: FrameRequestCallback) => {
      callback(0);
      return 1;
    }) as typeof window.requestAnimationFrame;
    window.cancelAnimationFrame = vi.fn() as typeof window.cancelAnimationFrame;

    class ResizeObserverTestDouble {
      constructor(_callback: ResizeObserverCallback) {}

      observe(target: Element) {
        observedTarget = target;
      }

      unobserve() {}

      disconnect() {}
    }

    globalThis.ResizeObserver = ResizeObserverTestDouble as typeof ResizeObserver;
  });

  afterEach(() => {
    Element.prototype.getBoundingClientRect = originalGetBoundingClientRect;
  });

  it("observes the stable chart host instead of the chart component or library mount node", () => {
    const host = document.createElement("div");
    host.className = "chart-panel";
    document.body.append(host);

    const wrapper = mount(TradingViewChart, {
      attachTo: host,
      props: {
        data: [{ time: 1_788_220_800, open: 100, high: 110, low: 95, close: 104 }],
        emptyTitle: "No candles",
      },
    });

    const root = wrapper.get(".trading-chart").element;
    const canvasHost = wrapper.get(".trading-chart__canvas").element;

    expect(observedTarget).toBe(host);
    expect(observedTarget).not.toBe(root);
    expect(observedTarget).not.toBe(canvasHost);

    wrapper.unmount();
    host.remove();
  });

  it("observes the chart panel instead of the research chart body", () => {
    const panel = document.createElement("section");
    panel.className = "chart-panel";
    const body = document.createElement("div");
    body.className = "research-chart-body";
    panel.append(body);
    document.body.append(panel);

    const wrapper = mount(TradingViewChart, {
      attachTo: body,
      props: {
        data: [{ time: 1_788_220_800, open: 100, high: 110, low: 95, close: 104 }],
        emptyTitle: "No candles",
      },
    });

    expect(observedTarget).toBe(panel);
    expect(observedTarget).not.toBe(body);

    wrapper.unmount();
    panel.remove();
  });

  it("uses the stable chart host size instead of inflated chart library heights", () => {
    const panel = document.createElement("section");
    panel.className = "chart-panel";
    const host = document.createElement("div");
    host.className = "research-chart-body";
    panel.append(host);
    document.body.append(panel);

    Object.defineProperty(panel, "clientHeight", { configurable: true, value: 760 });

    Element.prototype.getBoundingClientRect = function getBoundingClientRect() {
      if (this === panel) {
        return rect({ top: 100, width: 1200, height: 760 });
      }
      if (this === host) {
        return rect({ top: 180, width: 1200, height: 680 });
      }
      if (this instanceof Element && this.classList.contains("trading-chart")) {
        return rect({ top: 180, width: 1200, height: 3200 });
      }
      if (this instanceof Element && this.classList.contains("trading-chart__canvas")) {
        return rect({ top: 180, width: 1200, height: 3200 });
      }
      return originalGetBoundingClientRect.call(this);
    };

    const wrapper = mount(TradingViewChart, {
      attachTo: host,
      props: {
        data: [{ time: 1_788_220_800, open: 100, high: 110, low: 95, close: 104 }],
        emptyTitle: "No candles",
      },
    });

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        width: 1200,
        height: 680,
      }),
    );

    wrapper.unmount();
    panel.remove();
  });

  it("clamps oversized chart host height to the remaining chart panel space", () => {
    const panel = document.createElement("section");
    panel.className = "chart-panel";
    const host = document.createElement("div");
    host.className = "research-chart-body";
    panel.append(host);
    document.body.append(panel);

    Object.defineProperty(panel, "clientHeight", { configurable: true, value: 800 });

    Element.prototype.getBoundingClientRect = function getBoundingClientRect() {
      if (this === panel) {
        return rect({ top: 100, width: 1200, height: 800 });
      }
      if (this === host) {
        return rect({ top: 180, width: 1200, height: 1200 });
      }
      if (this instanceof Element && this.classList.contains("trading-chart")) {
        return rect({ top: 180, width: 1200, height: 1200 });
      }
      if (this instanceof Element && this.classList.contains("trading-chart__canvas")) {
        return rect({ top: 180, width: 1200, height: 1200 });
      }
      return originalGetBoundingClientRect.call(this);
    };

    const wrapper = mount(TradingViewChart, {
      attachTo: host,
      props: {
        data: [{ time: 1_788_220_800, open: 100, high: 110, low: 95, close: 104 }],
        emptyTitle: "No candles",
      },
    });

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        width: 1200,
        height: 720,
      }),
    );

    wrapper.unmount();
    panel.remove();
  });

  it("uses remaining chart panel space when the chart body reports no height", () => {
    const panel = document.createElement("section");
    panel.className = "chart-panel";
    const host = document.createElement("div");
    host.className = "research-chart-body";
    panel.append(host);
    document.body.append(panel);

    Object.defineProperty(panel, "clientHeight", { configurable: true, value: 760 });

    Element.prototype.getBoundingClientRect = function getBoundingClientRect() {
      if (this === panel) {
        return rect({ top: 100, width: 1200, height: 760 });
      }
      if (this === host) {
        return rect({ top: 220, width: 1200, height: 0 });
      }
      return originalGetBoundingClientRect.call(this);
    };

    const wrapper = mount(TradingViewChart, {
      attachTo: host,
      props: {
        data: [{ time: 1_788_220_800, open: 100, high: 110, low: 95, close: 104 }],
        emptyTitle: "No candles",
      },
    });

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        width: 1200,
        height: 640,
      }),
    );

    wrapper.unmount();
    panel.remove();
  });
});

function rect({ top, width, height }: { top: number; width: number; height: number }) {
  return {
    x: 0,
    y: top,
    top,
    left: 0,
    right: width,
    bottom: top + height,
    width,
    height,
    toJSON: () => ({}),
  } as DOMRect;
}
