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

  beforeEach(() => {
    setActivePinia(createPinia());
    observedTarget = null;
    resizeCallback = null;
    originalGetBoundingClientRect = Element.prototype.getBoundingClientRect;
    vi.clearAllMocks();
    chartMocks.createChart.mockReset();
    mockChartApi();

    window.requestAnimationFrame = ((callback: FrameRequestCallback) => {
      callback(0);
      return 0;
    }) as typeof window.requestAnimationFrame;
    window.cancelAnimationFrame = vi.fn() as typeof window.cancelAnimationFrame;

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
  });

  it("observes the chart viewport instead of the component or chart library node", () => {
    const host = document.createElement("div");
    host.className = "research-chart-body";
    document.body.append(host);

    const wrapper = mountChart(host);
    const root = wrapper.get(".trading-chart").element;
    const canvasHost = wrapper.get(".trading-chart__canvas").element;

    expect(observedTarget).toBe(host);
    expect(observedTarget).not.toBe(root);
    expect(observedTarget).not.toBe(canvasHost);

    wrapper.unmount();
    host.remove();
  });

  it("observes the chart viewport inside a fixed panel", () => {
    const panel = document.createElement("section");
    panel.className = "chart-panel";
    const host = document.createElement("div");
    host.className = "research-chart-body";
    panel.append(host);
    document.body.append(panel);

    const wrapper = mountChart(host);
    const root = wrapper.get(".trading-chart").element;
    const canvasHost = wrapper.get(".trading-chart__canvas").element;

    expect(observedTarget).toBe(host);
    expect(observedTarget).not.toBe(panel);
    expect(observedTarget).not.toBe(root);
    expect(observedTarget).not.toBe(canvasHost);

    wrapper.unmount();
    panel.remove();
  });

  it("uses the parent chart viewport size without reading inflated chart children", () => {
    const panel = document.createElement("section");
    panel.className = "chart-panel";
    const host = document.createElement("div");
    host.className = "research-chart-body";
    panel.append(host);
    document.body.append(panel);

    Element.prototype.getBoundingClientRect = function getBoundingClientRect() {
      if (this === panel) {
        return rect({ top: 100, width: 1200, height: 720 });
      }
      if (this === host) {
        return rect({ top: 180, width: 1180, height: 640 });
      }
      if (this instanceof Element && this.classList.contains("trading-chart")) {
        return rect({ top: 180, width: 1180, height: 3200 });
      }
      if (this instanceof Element && this.classList.contains("trading-chart__canvas")) {
        return rect({ top: 180, width: 1180, height: 3200 });
      }
      if (this instanceof Element && this.classList.contains("tv-lightweight-charts")) {
        return rect({ top: 180, width: 1180, height: 3200 });
      }
      return originalGetBoundingClientRect.call(this);
    };

    const wrapper = mountChart(host);

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        width: 1180,
        height: 640,
      }),
    );

    wrapper.unmount();
    panel.remove();
  });

  it("caps inflated viewport height to the chart panel boundary", () => {
    const panel = document.createElement("section");
    panel.className = "chart-panel";
    const host = document.createElement("div");
    host.className = "research-chart-body";
    panel.append(host);
    document.body.append(panel);

    Element.prototype.getBoundingClientRect = function getBoundingClientRect() {
      if (this === panel) {
        return rect({ top: 100, width: 1200, height: 760 });
      }
      if (this === host) {
        return rect({ top: 180, width: 1180, height: 3200 });
      }
      return originalGetBoundingClientRect.call(this);
    };

    const wrapper = mountChart(host);

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        width: 1180,
        height: 680,
      }),
    );

    wrapper.unmount();
    panel.remove();
  });

  it("prefers the chart viewport client size over inflated layout bounds", () => {
    const panel = document.createElement("section");
    panel.className = "chart-panel";
    const host = document.createElement("div");
    host.className = "research-chart-body";
    panel.append(host);
    document.body.append(panel);
    setClientSize(panel, { width: 1200, height: 760 });
    setClientSize(host, { width: 1180, height: 640 });

    Element.prototype.getBoundingClientRect = function getBoundingClientRect() {
      if (this === panel) {
        return rect({ top: 100, width: 1200, height: 760 });
      }
      if (this === host) {
        return rect({ top: 180, width: 1180, height: 3200 });
      }
      if (this instanceof Element && this.classList.contains("tv-lightweight-charts")) {
        return rect({ top: 180, width: 1180, height: 3200 });
      }
      return originalGetBoundingClientRect.call(this);
    };

    const wrapper = mountChart(host);

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        width: 1180,
        height: 640,
      }),
    );
    chartMocks.resize.mockClear();

    setClientSize(host, { width: 1180, height: 5000 });
    resizeCallback?.([resizeEntry(host, { width: 1180, height: 5000 })], {} as ResizeObserver);

    expect(chartMocks.resize).toHaveBeenCalledTimes(1);
    expect(chartMocks.resize).toHaveBeenCalledWith(1180, 680);

    wrapper.unmount();
    panel.remove();
  });

  it("uses the chart panel size when the component is mounted directly in a panel", () => {
    const panel = document.createElement("section");
    panel.className = "chart-panel";
    document.body.append(panel);
    setClientSize(panel, { width: 900, height: 560 });

    Element.prototype.getBoundingClientRect = function getBoundingClientRect() {
      if (this === panel) {
        return rect({ top: 100, width: 900, height: 560 });
      }
      return originalGetBoundingClientRect.call(this);
    };

    const wrapper = mountChart(panel);

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        width: 900,
        height: 560,
      }),
    );

    wrapper.unmount();
    panel.remove();
  });

  it("ignores inflated direct panel client height when css height is fixed", () => {
    const panel = document.createElement("section");
    panel.className = "chart-panel";
    panel.style.height = "760px";
    document.body.append(panel);
    setClientSize(panel, { width: 900, height: 5000 });

    const wrapper = mountChart(panel);

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        width: 900,
        height: 760,
      }),
    );

    wrapper.unmount();
    panel.remove();
  });

  it("resizes only to changed viewport dimensions", () => {
    const panel = document.createElement("section");
    panel.className = "chart-panel";
    const host = document.createElement("div");
    host.className = "research-chart-body";
    panel.append(host);
    document.body.append(panel);
    setClientSize(panel, { width: 1000, height: 700 });
    setClientSize(host, { width: 1000, height: 620 });

    let size = { width: 1000, height: 620 };
    Element.prototype.getBoundingClientRect = function getBoundingClientRect() {
      if (this === panel) {
        return rect({ top: 80, width: 1000, height: 700 });
      }
      if (this === host) {
        return rect({ top: 100, width: size.width, height: size.height });
      }
      return originalGetBoundingClientRect.call(this);
    };

    const wrapper = mountChart(host);
    chartMocks.resize.mockClear();

    resizeCallback?.([], {} as ResizeObserver);
    expect(chartMocks.resize).not.toHaveBeenCalled();

    size = { width: 1000, height: 580 };
    setClientSize(host, size);
    resizeCallback?.([], {} as ResizeObserver);

    expect(chartMocks.resize).toHaveBeenCalledTimes(1);
    expect(chartMocks.resize).toHaveBeenCalledWith(1000, 580);
    expect(wrapper.get<HTMLElement>(".trading-chart").element.style.height).toBe("");
    expect(wrapper.get<HTMLElement>(".trading-chart__canvas").element.style.height).toBe("");

    wrapper.unmount();
    panel.remove();
  });

  it("does not chase chart-driven host height growth beyond the panel boundary", () => {
    const panel = document.createElement("section");
    panel.className = "chart-panel";
    const host = document.createElement("div");
    host.className = "research-chart-body";
    panel.append(host);
    document.body.append(panel);

    let hostHeight = 680;
    Element.prototype.getBoundingClientRect = function getBoundingClientRect() {
      if (this === panel) {
        return rect({ top: 100, width: 1200, height: 760 });
      }
      if (this === host) {
        return rect({ top: 180, width: 1180, height: hostHeight });
      }
      return originalGetBoundingClientRect.call(this);
    };

    const wrapper = mountChart(host);
    chartMocks.resize.mockClear();

    hostHeight = 3200;
    resizeCallback?.([], {} as ResizeObserver);

    expect(chartMocks.resize).not.toHaveBeenCalled();

    wrapper.unmount();
    panel.remove();
  });

  it("ignores observed chart viewport height when chart internals inflate layout bounds", () => {
    const panel = document.createElement("section");
    panel.className = "chart-panel";
    const host = document.createElement("div");
    host.className = "research-chart-body";
    panel.append(host);
    document.body.append(panel);
    setClientSize(panel, { width: 1200, height: 760 });
    setClientSize(host, { width: 1180, height: 680 });

    let inflated = false;
    Element.prototype.getBoundingClientRect = function getBoundingClientRect() {
      if (this === panel) {
        return rect({ top: 100, width: 1200, height: inflated ? 5200 : 760 });
      }
      if (this === host) {
        return rect({ top: 180, width: 1180, height: inflated ? 5000 : 680 });
      }
      return originalGetBoundingClientRect.call(this);
    };

    const wrapper = mountChart(host);
    chartMocks.resize.mockClear();

    inflated = true;
    resizeCallback?.([resizeEntry(host, { width: 1180, height: 680 })], {} as ResizeObserver);

    expect(chartMocks.resize).not.toHaveBeenCalled();

    wrapper.unmount();
    panel.remove();
  });

  it("ignores inflated observer height when the chart panel has a fixed css height", () => {
    const panel = document.createElement("section");
    panel.className = "chart-panel";
    panel.style.height = "760px";
    const host = document.createElement("div");
    host.className = "research-chart-body";
    panel.append(host);
    document.body.append(panel);

    let inflated = false;
    Element.prototype.getBoundingClientRect = function getBoundingClientRect() {
      if (this === panel) {
        return rect({ top: 100, width: 1200, height: inflated ? 5200 : 760 });
      }
      if (this === host) {
        return rect({ top: 180, width: 1180, height: inflated ? 5000 : 680 });
      }
      return originalGetBoundingClientRect.call(this);
    };

    const wrapper = mountChart(host);
    chartMocks.resize.mockClear();

    inflated = true;
    resizeCallback?.([resizeEntry(host, { width: 1180, height: 5200 })], {} as ResizeObserver);

    expect(chartMocks.resize).not.toHaveBeenCalled();
    expect(wrapper.get<HTMLElement>(".trading-chart").element.style.height).toBe("");
    expect(wrapper.get<HTMLElement>(".trading-chart__canvas").element.style.height).toBe("");

    wrapper.unmount();
    panel.remove();
  });

  it("does not chase chart-driven host growth without a trusted height boundary", () => {
    const host = document.createElement("div");
    host.className = "research-chart-body";
    document.body.append(host);

    Object.defineProperty(window, "innerHeight", { configurable: true, value: 768 });

    let hostHeight = 620;
    Element.prototype.getBoundingClientRect = function getBoundingClientRect() {
      if (this === host) {
        return rect({ top: 100, width: 1000, height: hostHeight });
      }
      return originalGetBoundingClientRect.call(this);
    };

    const wrapper = mountChart(host);
    chartMocks.resize.mockClear();

    hostHeight = 5000;
    resizeCallback?.([], {} as ResizeObserver);

    expect(chartMocks.resize).not.toHaveBeenCalled();

    hostHeight = 8000;
    resizeCallback?.([], {} as ResizeObserver);

    expect(chartMocks.resize).not.toHaveBeenCalled();

    wrapper.unmount();
    host.remove();
  });

  it("leaves root and canvas sizing to the fixed viewport css", () => {
    const panel = document.createElement("section");
    panel.className = "chart-panel";
    const host = document.createElement("div");
    host.className = "research-chart-body";
    panel.append(host);
    document.body.append(panel);
    setClientSize(panel, { width: 1000, height: 700 });
    setClientSize(host, { width: 1000, height: 620 });

    const wrapper = mountChart(host);
    const root = wrapper.get<HTMLElement>(".trading-chart").element;
    const canvasHost = wrapper.get<HTMLElement>(".trading-chart__canvas").element;

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        width: 1000,
        height: 620,
      }),
    );
    expect(root.style.width).toBe("");
    expect(root.style.height).toBe("");
    expect(canvasHost.style.width).toBe("");
    expect(canvasHost.style.height).toBe("");

    wrapper.unmount();
    panel.remove();
  });
});

function mountChart(host: HTMLElement) {
  return mount(TradingViewChart, {
    attachTo: host,
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

function setClientSize(element: HTMLElement, size: { width: number; height: number }) {
  Object.defineProperty(element, "clientWidth", { configurable: true, value: size.width });
  Object.defineProperty(element, "clientHeight", { configurable: true, value: size.height });
}

function resizeEntry(target: Element, size: { width: number; height: number }) {
  return {
    target,
    contentRect: rect({ top: 0, width: size.width, height: size.height }),
    contentBoxSize: [{ inlineSize: size.width, blockSize: size.height }],
  } as unknown as ResizeObserverEntry;
}
