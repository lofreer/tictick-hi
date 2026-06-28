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
  let resizeCallback: ResizeObserverCallback | null = null;
  let originalGetBoundingClientRect: typeof Element.prototype.getBoundingClientRect;
  let originalClientWidth: PropertyDescriptor | undefined;
  let originalClientHeight: PropertyDescriptor | undefined;
  let viewportSize: { width: number; height: number };

  beforeEach(() => {
    setActivePinia(createPinia());
    observedTarget = null;
    resizeCallback = null;
    viewportSize = { width: 1180, height: 640 };
    originalGetBoundingClientRect = Element.prototype.getBoundingClientRect;
    originalClientWidth = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "clientWidth");
    originalClientHeight = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "clientHeight");
    vi.clearAllMocks();
    chartMocks.createChart.mockReset();
    mockChartApi();

    window.requestAnimationFrame = ((callback: FrameRequestCallback) => {
      callback(0);
      return 0;
    }) as typeof window.requestAnimationFrame;
    window.cancelAnimationFrame = vi.fn() as typeof window.cancelAnimationFrame;
    Object.defineProperty(window, "innerWidth", { configurable: true, value: 1440 });
    Object.defineProperty(window, "innerHeight", { configurable: true, value: 768 });

    Object.defineProperty(HTMLElement.prototype, "clientWidth", {
      configurable: true,
      get() {
        return this instanceof Element && this.classList.contains("research-chart-body") ? viewportSize.width : 0;
      },
    });
    Object.defineProperty(HTMLElement.prototype, "clientHeight", {
      configurable: true,
      get() {
        return this instanceof Element && this.classList.contains("research-chart-body") ? viewportSize.height : 0;
      },
    });

    Element.prototype.getBoundingClientRect = function getBoundingClientRect() {
      if (this instanceof Element && this.classList.contains("research-chart-body")) {
        return rect({ top: 180, width: viewportSize.width, height: viewportSize.height });
      }
      if (this instanceof Element && this.classList.contains("trading-chart")) {
        return rect({ top: 180, width: viewportSize.width, height: viewportSize.height + 2200 });
      }
      if (this instanceof Element && this.classList.contains("trading-chart__canvas")) {
        return rect({ top: 180, width: viewportSize.width, height: viewportSize.height + 3200 });
      }
      if (this instanceof Element && this.classList.contains("chart-panel")) {
        return rect({ top: 100, width: viewportSize.width + 20, height: 5200 });
      }
      if (this instanceof Element && this.classList.contains("tv-lightweight-charts")) {
        return rect({ top: 180, width: viewportSize.width, height: 8000 });
      }
      return originalGetBoundingClientRect.call(this);
    };

    class ResizeObserverTestDouble {
      constructor(callback: ResizeObserverCallback) {
        resizeCallback = callback;
      }

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
    restorePrototypeProperty("clientWidth", originalClientWidth);
    restorePrototypeProperty("clientHeight", originalClientHeight);
  });

  it("observes only the external fixed chart slot", () => {
    const host = createResearchHost();
    const wrapper = mountChart(host.body);
    const root = wrapper.get(".trading-chart").element;
    const canvasHost = wrapper.get(".trading-chart__canvas").element;

    expect(observedTarget).toBe(host.body);
    expect(observedTarget).not.toBe(root);
    expect(observedTarget).not.toBe(canvasHost);
    expect(observedTarget).not.toBe(host.panel);

    wrapper.unmount();
    host.panel.remove();
  });

  it("does not observe a generic chart panel without an explicit viewport marker", () => {
    const host = createResearchHost({ markViewport: false });
    const wrapper = mountChart(host.body);

    expect(observedTarget).not.toBe(host.panel);
    expect(observedTarget).not.toBe(host.body);

    wrapper.unmount();
    host.panel.remove();
  });

  it("initializes from the fixed chart slot without reading inflated internal nodes", () => {
    const host = createResearchHost();
    const wrapper = mountChart(host.body);

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        width: 1180,
        height: 640,
      }),
    );

    wrapper.unmount();
    host.panel.remove();
  });

  it("prefers fixed CSS height over polluted client height", () => {
    viewportSize = { width: 1180, height: 5000 };
    const host = createResearchHost();
    host.body.style.height = "603px";
    const wrapper = mountChart(host.body);

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        width: 1180,
        height: 603,
      }),
    );

    chartMocks.resize.mockClear();
    resizeCallback?.([resizeEntry(observedTarget!, { width: 1180, height: 9000 })], {} as ResizeObserver);

    expect(chartMocks.resize).not.toHaveBeenCalled();

    Object.defineProperty(window, "innerHeight", { configurable: true, value: 769 });
    host.body.style.height = "604px";
    resizeCallback?.([resizeEntry(observedTarget!, { width: 1180, height: 9000 })], {} as ResizeObserver);

    expect(chartMocks.resize).toHaveBeenCalledTimes(1);
    expect(chartMocks.resize).toHaveBeenCalledWith(1180, 604);

    wrapper.unmount();
    host.panel.remove();
  });

  it("blocks fixed viewport height growth when the window and width are unchanged", () => {
    const host = createResearchHost();
    const wrapper = mountChart(host.body);
    chartMocks.resize.mockClear();

    viewportSize = { width: 1180, height: 641 };
    resizeCallback?.([resizeEntry(observedTarget!, viewportSize)], {} as ResizeObserver);

    expect(chartMocks.resize).not.toHaveBeenCalled();

    wrapper.unmount();
    host.panel.remove();
  });

  it("resizes only when the fixed chart slot changes", () => {
    const host = createResearchHost();
    const wrapper = mountChart(host.body);
    chartMocks.resize.mockClear();

    resizeCallback?.([resizeEntry(observedTarget!, { width: 1180, height: 640 })], {} as ResizeObserver);
    expect(chartMocks.resize).not.toHaveBeenCalled();

    viewportSize = { width: 1180, height: 603 };
    resizeCallback?.([resizeEntry(observedTarget!, viewportSize)], {} as ResizeObserver);

    expect(chartMocks.resize).toHaveBeenCalledTimes(1);
    expect(chartMocks.resize).toHaveBeenCalledWith(1180, 603);

    wrapper.unmount();
    host.panel.remove();
  });

  it("ignores resize entries from internal chart elements", () => {
    const host = createResearchHost();
    const wrapper = mountChart(host.body);
    const root = wrapper.get(".trading-chart").element;
    const canvasHost = wrapper.get(".trading-chart__canvas").element;
    chartMocks.resize.mockClear();

    resizeCallback?.(
      [resizeEntry(root, { width: 1180, height: 7000 }), resizeEntry(canvasHost, { width: 1180, height: 9000 })],
      {} as ResizeObserver,
    );

    expect(chartMocks.resize).not.toHaveBeenCalled();

    wrapper.unmount();
    host.panel.remove();
  });

  it("ignores observed fixed viewport height when client size is unavailable", () => {
    viewportSize = { width: 0, height: 0 };
    const host = createResearchHost();
    host.body.style.width = "1000px";
    host.body.style.height = "580px";
    const wrapper = mountChart(host.body);

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        width: 1000,
        height: 580,
      }),
    );

    chartMocks.resize.mockClear();
    resizeCallback?.([resizeEntry(observedTarget!, { width: 1000, height: 9000 })], {} as ResizeObserver);

    expect(chartMocks.resize).not.toHaveBeenCalled();

    host.body.style.height = "560px";
    resizeCallback?.([resizeEntry(observedTarget!, { width: 1000, height: 9000 })], {} as ResizeObserver);

    expect(chartMocks.resize).toHaveBeenCalledTimes(1);
    expect(chartMocks.resize).toHaveBeenCalledWith(1000, 560);

    wrapper.unmount();
    host.panel.remove();
  });

  it("caps an anomalous initial direct viewport height to the safe render maximum", () => {
    viewportSize = { width: 1180, height: 5000 };
    const host = createResearchHost();
    const wrapper = mountChart(host.body);

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        width: 1180,
        height: 768,
      }),
    );

    wrapper.unmount();
    host.panel.remove();
  });

  it("does not write inline size back to root or canvas elements", () => {
    const host = createResearchHost();
    const wrapper = mountChart(host.body);
    const root = wrapper.get<HTMLElement>(".trading-chart").element;
    const canvasHost = wrapper.get<HTMLElement>(".trading-chart__canvas").element;

    expect(root.style.width).toBe("");
    expect(root.style.height).toBe("");
    expect(canvasHost.style.width).toBe("");
    expect(canvasHost.style.height).toBe("");

    wrapper.unmount();
    host.panel.remove();
  });
});

function createResearchHost(options: { markViewport?: boolean } = {}) {
  const panel = document.createElement("section");
  panel.className = "chart-panel";
  const body = document.createElement("div");
  body.className = "research-chart-body";
  if (options.markViewport !== false) {
    body.setAttribute("data-chart-viewport", "fixed");
  }
  panel.append(body);
  document.body.append(panel);
  return { panel, body };
}

function mountChart(host: HTMLElement) {
  const mountPoint = document.createElement("div");
  host.append(mountPoint);
  return mount(TradingViewChart, {
    attachTo: mountPoint,
    props: {
      data: [{ time: 1_788_220_800, open: 100, high: 110, low: 95, close: 104 }],
      emptyTitle: "No candles",
    },
  });
}

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

function resizeEntry(target: Element, size: { width: number; height: number }) {
  return {
    target,
    contentRect: rect({ top: 0, width: size.width, height: size.height }),
    contentBoxSize: [{ inlineSize: size.width, blockSize: size.height }],
  } as unknown as ResizeObserverEntry;
}

function restorePrototypeProperty(name: "clientWidth" | "clientHeight", descriptor: PropertyDescriptor | undefined) {
  if (descriptor) {
    Object.defineProperty(HTMLElement.prototype, name, descriptor);
    return;
  }
  Reflect.deleteProperty(HTMLElement.prototype, name);
}
