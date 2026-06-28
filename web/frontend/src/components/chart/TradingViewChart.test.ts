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
  setVisibleLogicalRange: vi.fn(),
  setData: vi.fn(),
  setMarkers: vi.fn(),
}));

vi.mock("lightweight-charts", () => ({
  CandlestickSeries: "CandlestickSeries",
  createChart: chartMocks.createChart,
  createSeriesMarkers: vi.fn(() => ({ setMarkers: chartMocks.setMarkers })),
}));

const mockedCreateChart = vi.mocked(createChart);
const researchViewportSize = { width: 1180, height: 640 };
const researchViewportGutter = { inlineEnd: 12, blockEnd: 12 };
const researchRenderSize = {
  width: researchViewportSize.width - researchViewportGutter.inlineEnd,
  height: researchViewportSize.height - researchViewportGutter.blockEnd,
};

function mockChartApi() {
  chartMocks.createChart.mockReturnValue({
    addSeries: vi.fn(() => ({ setData: chartMocks.setData })),
    applyOptions: vi.fn(),
    remove: chartMocks.remove,
    resize: chartMocks.resize,
    timeScale: vi.fn(() => ({
      fitContent: chartMocks.fitContent,
      setVisibleLogicalRange: chartMocks.setVisibleLogicalRange,
    })),
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
        autoSize: false,
        width: researchRenderSize.width,
        height: researchRenderSize.height,
      }),
    );

    wrapper.unmount();
    host.panel.remove();
  });

  it("reserves enough right price scale width for visible price labels", () => {
    const host = createResearchHost();
    const wrapper = mountChart(host.body);

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        rightPriceScale: expect.objectContaining({
          minimumWidth: 132,
        }),
      }),
    );

    wrapper.unmount();
    host.panel.remove();
  });

  it("pads the visible logical range so edge time labels are not clipped", () => {
    const host = createResearchHost();
    const wrapper = mountChart(host.body);

    expect(chartMocks.fitContent).toHaveBeenCalled();
    expect(chartMocks.setVisibleLogicalRange).toHaveBeenCalledWith({ from: -6, to: 6 });

    wrapper.unmount();
    host.panel.remove();
  });

  it("scales the logical edge padding with dense candle windows", () => {
    const host = createResearchHost();
    const wrapper = mountChart(host.body, Array.from({ length: 1000 }, (_, index) => ({
      time: index + 1,
      open: 100,
      high: 110,
      low: 95,
      close: 104,
    })));

    expect(chartMocks.setVisibleLogicalRange).toHaveBeenLastCalledWith({ from: -55, to: 1054 });

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
        width: researchRenderSize.width,
        height: 591,
      }),
    );

    chartMocks.resize.mockClear();
    resizeCallback?.([resizeEntry(observedTarget!, { width: 1180, height: 9000 })], {} as ResizeObserver);

    expect(chartMocks.resize).not.toHaveBeenCalled();

    Object.defineProperty(window, "innerHeight", { configurable: true, value: 769 });
    host.body.style.height = "604px";
    window.dispatchEvent(new Event("resize"));

    expect(chartMocks.resize).toHaveBeenCalledTimes(1);
    expect(chartMocks.resize).toHaveBeenCalledWith(researchRenderSize.width, 592);

    wrapper.unmount();
    host.panel.remove();
  });

  it("prefers fixed max-height over polluted fixed viewport height", () => {
    viewportSize = { width: 1180, height: 9000 };
    const host = createResearchHost({ declareViewportSize: false });
    host.body.style.width = "1180px";
    host.body.style.height = "9000px";
    host.body.style.maxHeight = "603px";
    const wrapper = mountChart(host.body);

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        width: researchRenderSize.width,
        height: 591,
      }),
    );

    chartMocks.resize.mockClear();
    host.body.style.height = "9500px";
    resizeCallback?.([resizeEntry(observedTarget!, { width: 1180, height: 9500 })], {} as ResizeObserver);

    expect(chartMocks.resize).not.toHaveBeenCalled();

    Object.defineProperty(window, "innerHeight", { configurable: true, value: 769 });
    host.body.style.maxHeight = "604px";
    window.dispatchEvent(new Event("resize"));

    expect(chartMocks.resize).toHaveBeenCalledTimes(1);
    expect(chartMocks.resize).toHaveBeenCalledWith(researchRenderSize.width, 592);

    wrapper.unmount();
    host.panel.remove();
  });

  it("uses the declared fixed viewport height before a looser max-height", () => {
    const host = createFixedPanelHost();
    const wrapper = mountChart(host.panel);

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        width: 1000,
        height: 720,
      }),
    );

    wrapper.unmount();
    host.panel.remove();
  });

  it("blocks fixed viewport height changes while the window is unchanged", () => {
    const host = createResearchHost();
    const wrapper = mountChart(host.body);
    chartMocks.resize.mockClear();

    viewportSize = { width: 1180, height: 641 };
    host.body.style.height = "641px";
    resizeCallback?.([resizeEntry(observedTarget!, viewportSize)], {} as ResizeObserver);

    expect(chartMocks.resize).not.toHaveBeenCalled();

    wrapper.unmount();
    host.panel.remove();
  });

  it("does not accept fixed viewport height changes when only the width changes", () => {
    const host = createResearchHost();
    const wrapper = mountChart(host.body);
    chartMocks.resize.mockClear();

    resizeCallback?.([resizeEntry(observedTarget!, { width: 1180, height: 640 })], {} as ResizeObserver);
    expect(chartMocks.resize).not.toHaveBeenCalled();

    viewportSize = { width: 1190, height: 603 };
    host.body.style.width = "1190px";
    host.body.style.height = "603px";
    resizeCallback?.([resizeEntry(observedTarget!, viewportSize)], {} as ResizeObserver);

    expect(chartMocks.resize).toHaveBeenCalledTimes(1);
    expect(chartMocks.resize).toHaveBeenCalledWith(1178, researchRenderSize.height);

    wrapper.unmount();
    host.panel.remove();
  });

  it("keeps the fixed viewport snapshot when a window resize sees polluted height", () => {
    const host = createResearchHost();
    const wrapper = mountChart(host.body);
    chartMocks.resize.mockClear();

    Object.defineProperty(window, "innerHeight", { configurable: true, value: 769 });
    viewportSize = { width: 1190, height: 9000 };
    host.body.style.width = "1190px";
    host.body.style.height = "9000px";
    window.dispatchEvent(new Event("resize"));

    expect(chartMocks.resize).toHaveBeenCalledTimes(1);
    expect(chartMocks.resize).toHaveBeenCalledWith(1178, researchRenderSize.height);

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
        width: 988,
        height: 568,
      }),
    );

    chartMocks.resize.mockClear();
    resizeCallback?.([resizeEntry(observedTarget!, { width: 1000, height: 9000 })], {} as ResizeObserver);

    expect(chartMocks.resize).not.toHaveBeenCalled();

    host.body.style.height = "560px";
    resizeCallback?.([resizeEntry(observedTarget!, { width: 1000, height: 9000 })], {} as ResizeObserver);

    expect(chartMocks.resize).not.toHaveBeenCalled();

    Object.defineProperty(window, "innerHeight", { configurable: true, value: 769 });
    window.dispatchEvent(new Event("resize"));

    expect(chartMocks.resize).toHaveBeenCalledTimes(1);
    expect(chartMocks.resize).toHaveBeenCalledWith(988, 548);

    wrapper.unmount();
    host.panel.remove();
  });

  it("falls back when a fixed viewport exposes only an inflated direct height", () => {
    viewportSize = { width: 1180, height: 5000 };
    const host = createResearchHost({ declareViewportSize: false });
    const wrapper = mountChart(host.body);

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        width: researchRenderSize.width,
        height: 348,
      }),
    );

    wrapper.unmount();
    host.panel.remove();
  });

  it("locks root and canvas elements to the measured viewport size", () => {
    const host = createResearchHost();
    const wrapper = mountChart(host.body);
    const root = wrapper.get<HTMLElement>(".trading-chart").element;
    const canvasHost = wrapper.get<HTMLElement>(".trading-chart__canvas").element;

    expectLockedSize(root, researchRenderSize);
    expectLockedSize(canvasHost, researchRenderSize);

    wrapper.unmount();
    host.panel.remove();
  });

  it("restores locked chart dimensions after runtime node height pollution", () => {
    const host = createResearchHost();
    const wrapper = mountChart(host.body);
    const root = wrapper.get<HTMLElement>(".trading-chart").element;
    const canvasHost = wrapper.get<HTMLElement>(".trading-chart__canvas").element;
    chartMocks.resize.mockClear();

    for (const element of [root, canvasHost]) {
      element.style.height = "9000px";
      element.style.maxHeight = "9000px";
      element.style.blockSize = "9000px";
      element.style.maxBlockSize = "9000px";
    }

    window.dispatchEvent(new Event("resize"));

    expect(chartMocks.resize).not.toHaveBeenCalled();
    expectLockedSize(root, researchRenderSize);
    expectLockedSize(canvasHost, researchRenderSize);

    wrapper.unmount();
    host.panel.remove();
  });
});

function createResearchHost(options: { declareViewportSize?: boolean; markViewport?: boolean } = {}) {
  const panel = document.createElement("section");
  panel.className = "chart-panel";
  const body = document.createElement("div");
  body.className = "research-chart-body";
  body.style.setProperty("--tt-chart-fixed-inline-end-gutter", `${researchViewportGutter.inlineEnd}px`);
  body.style.setProperty("--tt-chart-fixed-block-end-gutter", `${researchViewportGutter.blockEnd}px`);
  if (options.declareViewportSize !== false) {
    body.style.width = `${researchViewportSize.width}px`;
    body.style.height = `${researchViewportSize.height}px`;
  }
  if (options.markViewport !== false) {
    body.setAttribute("data-chart-viewport", "fixed");
  }
  panel.append(body);
  document.body.append(panel);
  return { panel, body };
}

function createFixedPanelHost() {
  const panel = document.createElement("section");
  panel.className = "chart-panel";
  panel.setAttribute("data-chart-viewport", "fixed");
  panel.style.width = "1000px";
  panel.style.height = "720px";
  panel.style.maxHeight = "820px";
  document.body.append(panel);
  return { panel };
}

function mountChart(host: HTMLElement, data = [{ time: 1_788_220_800, open: 100, high: 110, low: 95, close: 104 }]) {
  const mountPoint = document.createElement("div");
  host.append(mountPoint);
  return mount(TradingViewChart, {
    attachTo: mountPoint,
    props: {
      data,
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

function expectLockedSize(element: HTMLElement, size: { width: number; height: number }) {
  expect(element.style.getPropertyValue("--tt-chart-render-width")).toBe(`${size.width}px`);
  expect(element.style.getPropertyValue("--tt-chart-render-height")).toBe(`${size.height}px`);
  for (const [property, value] of [
    ["width", `${size.width}px`],
    ["height", `${size.height}px`],
    ["max-width", `${size.width}px`],
    ["max-height", `${size.height}px`],
    ["inline-size", `${size.width}px`],
    ["block-size", `${size.height}px`],
    ["max-inline-size", `${size.width}px`],
    ["max-block-size", `${size.height}px`],
  ]) {
    expect(element.style.getPropertyValue(property)).toBe(value);
    if (property === "width" || property === "height") {
      expect(element.style.getPropertyPriority(property)).toBe("important");
    }
  }
}

function restorePrototypeProperty(name: "clientWidth" | "clientHeight", descriptor: PropertyDescriptor | undefined) {
  if (descriptor) {
    Object.defineProperty(HTMLElement.prototype, name, descriptor);
    return;
  }
  Reflect.deleteProperty(HTMLElement.prototype, name);
}
