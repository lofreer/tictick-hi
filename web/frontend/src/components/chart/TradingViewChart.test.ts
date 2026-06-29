import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import TradingViewChart from "@/components/chart/TradingViewChart.vue";
import { createChart, HistogramSeries } from "lightweight-charts";

const chartMocks = vi.hoisted(() => ({
  addSeries: vi.fn(),
  applyOptions: vi.fn(),
  applyPriceScaleOptions: vi.fn(),
  createChart: vi.fn(),
  fitContent: vi.fn(),
  priceScale: vi.fn(),
  remove: vi.fn(),
  resize: vi.fn(),
  setCandleData: vi.fn(),
  setVisibleLogicalRange: vi.fn(),
  setMarkers: vi.fn(),
  setVolumeData: vi.fn(),
}));

vi.mock("lightweight-charts", () => ({
  CandlestickSeries: "CandlestickSeries",
  createChart: chartMocks.createChart,
  createSeriesMarkers: vi.fn(() => ({ setMarkers: chartMocks.setMarkers })),
  HistogramSeries: "HistogramSeries",
  TickMarkType: { Year: 0, Month: 1, DayOfMonth: 2, Time: 3, TimeWithSeconds: 4 },
}));

const mockedCreateChart = vi.mocked(createChart);
const researchViewportSize = { width: 1180, height: 640 };
const researchRenderSize = { ...researchViewportSize };

function mockChartApi() {
  chartMocks.addSeries.mockImplementation((seriesType) => {
    if (seriesType === "HistogramSeries") {
      return { setData: chartMocks.setVolumeData };
    }
    return { setData: chartMocks.setCandleData };
  });
  chartMocks.createChart.mockReturnValue({
    addSeries: chartMocks.addSeries,
    applyOptions: chartMocks.applyOptions,
    priceScale: chartMocks.priceScale.mockReturnValue({
      applyOptions: chartMocks.applyPriceScaleOptions,
    }),
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

  it("reserves responsive right price scale width for visible price labels", () => {
    const host = createResearchHost();
    const wrapper = mountChart(host.body);

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        rightPriceScale: expect.objectContaining({
          minimumWidth: 64,
        }),
      }),
    );

    wrapper.unmount();
    host.panel.remove();
  });

  it("uses a compact right price scale on narrow chart viewports", () => {
    viewportSize = { width: 390, height: 500 };
    const host = createResearchHost();
    const wrapper = mountChart(host.body);

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        rightPriceScale: expect.objectContaining({
          minimumWidth: 56,
        }),
      }),
    );

    wrapper.unmount();
    host.panel.remove();
  });

  it("formats large price labels without trailing decimal noise", () => {
    const host = createResearchHost();
    const wrapper = mountChart(host.body);
    const options = mockedCreateChart.mock.calls[0]?.[1] as {
      localization: { priceFormatter: (price: number) => string };
    };

    expect(options.localization.priceFormatter(60_664.22)).toBe("60664");
    expect(options.localization.priceFormatter(248.5)).toBe("248.5");
    expect(options.localization.priceFormatter(99.123)).toBe("99.12");
    expect(options.localization.priceFormatter(0.012_3)).toBe("0.0123");

    wrapper.unmount();
    host.panel.remove();
  });

  it("uses compact UTC tick labels so the time axis does not clip edge text", () => {
    const host = createResearchHost();
    const wrapper = mountChart(host.body);
    const options = mockedCreateChart.mock.calls[0]?.[1] as {
      timeScale: {
        secondsVisible: boolean;
        tickMarkMaxCharacterLength: number;
        tickMarkFormatter: (time: number, tickMarkType: number, locale: string) => string | null;
      };
    };

    expect(options.timeScale).toMatchObject({ secondsVisible: false, tickMarkMaxCharacterLength: 8 });
    const time = Date.UTC(2026, 5, 27, 18, 58) / 1000;
    for (const [tickMarkType, label] of [[3, "18:58"], [2, "06-27"], [1, "26-06"], [0, "2026"]] as const) {
      expect(options.timeScale.tickMarkFormatter(time, tickMarkType, "en-US")).toBe(label);
    }
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
      volume: 1000 + index,
    })));

    expect(chartMocks.setVisibleLogicalRange).toHaveBeenLastCalledWith({ from: 629, to: 1010 });

    wrapper.unmount();
    host.panel.remove();
  });

  it("renders volume histogram on an overlay price scale", () => {
    const host = createResearchHost();
    const wrapper = mountChart(host.body, [
      { time: 1, open: 100, high: 110, low: 95, close: 104, volume: 1200 },
      { time: 2, open: 104, high: 108, low: 96, close: 99, volume: 1800 },
    ]);

    expect(chartMocks.addSeries).toHaveBeenCalledWith(
      HistogramSeries,
      expect.objectContaining({
        base: 0,
        lastValueVisible: false,
        priceFormat: { type: "volume" },
        priceLineVisible: false,
        priceScaleId: "",
      }),
    );
    expect(chartMocks.setVolumeData).toHaveBeenCalledWith([
      { time: 1, value: 1200, color: "rgba(14, 203, 129, 0.28)" },
      { time: 2, value: 1800, color: "rgba(246, 70, 93, 0.28)" },
    ]);
    expect(chartMocks.priceScale).toHaveBeenCalledWith("right");
    expect(chartMocks.priceScale).toHaveBeenCalledWith("");
    expect(chartMocks.applyPriceScaleOptions).toHaveBeenCalledWith({
      scaleMargins: {
        top: 0.08,
        bottom: 0.24,
      },
    });
    expect(chartMocks.applyPriceScaleOptions).toHaveBeenCalledWith({
      scaleMargins: {
        top: 0.8,
        bottom: 0,
      },
    });

    wrapper.unmount();
    host.panel.remove();
  });

  it("prefers fixed CSS height over polluted client height without snapshotting", () => {
    viewportSize = { width: 1180, height: 5000 };
    const host = createResearchHost();
    host.body.style.height = "603px";
    const wrapper = mountChart(host.body);

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        width: researchRenderSize.width,
        height: 603,
      }),
    );

    chartMocks.resize.mockClear();
    resizeCallback?.([resizeEntry(observedTarget!, { width: 1180, height: 9000 })], {} as ResizeObserver);

    expect(chartMocks.resize).not.toHaveBeenCalled();

    Object.defineProperty(window, "innerHeight", { configurable: true, value: 769 });
    host.body.style.height = "604px";
    window.dispatchEvent(new Event("resize"));

    expect(chartMocks.resize).toHaveBeenCalledTimes(1);
    expect(chartMocks.resize).toHaveBeenCalledWith(researchRenderSize.width, 604);

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
        height: 603,
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
    expect(chartMocks.resize).toHaveBeenCalledWith(researchRenderSize.width, 604);

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

  it("resizes when the external fixed viewport changes size", () => {
    const host = createResearchHost();
    const wrapper = mountChart(host.body);
    chartMocks.resize.mockClear();

    viewportSize = { width: 1180, height: 641 };
    host.body.style.height = "641px";
    resizeCallback?.([resizeEntry(observedTarget!, viewportSize)], {} as ResizeObserver);

    expect(chartMocks.resize).toHaveBeenCalledTimes(1);
    expect(chartMocks.resize).toHaveBeenCalledWith(1180, 641);

    wrapper.unmount();
    host.panel.remove();
  });

  it("resizes both width and height from the external fixed viewport", () => {
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
    expect(chartMocks.resize).toHaveBeenCalledWith(1190, 603);

    wrapper.unmount();
    host.panel.remove();
  });

  it("does not resize from polluted internal node dimensions on window resize", () => {
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

  it("uses declared fixed viewport size when client size is unavailable", () => {
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

  it("falls back when a fixed viewport exposes no positive size", () => {
    viewportSize = { width: 0, height: 0 };
    const host = createResearchHost({ declareViewportSize: false });
    const wrapper = mountChart(host.body);

    expect(mockedCreateChart).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        width: 1,
        height: 360,
      }),
    );

    wrapper.unmount();
    host.panel.remove();
  });

  it("does not write inline chart viewport locks", () => {
    const host = createResearchHost();
    const wrapper = mountChart(host.body);
    const root = wrapper.get<HTMLElement>(".trading-chart").element;
    const canvasHost = wrapper.get<HTMLElement>(".trading-chart__canvas").element;

    for (const element of [root, canvasHost]) {
      expect(element.getAttribute("style") ?? "").not.toContain("--tt-chart-render");
      expect(element.style.getPropertyPriority("width")).toBe("");
      expect(element.style.getPropertyPriority("height")).toBe("");
    }

    wrapper.unmount();
    host.panel.remove();
  });
});

function createResearchHost(options: { declareViewportSize?: boolean; markViewport?: boolean } = {}) {
  const panel = document.createElement("section");
  panel.className = "chart-panel";
  const body = document.createElement("div");
  body.className = "research-chart-body";
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

function mountChart(host: HTMLElement, data = [{ time: 1_788_220_800, open: 100, high: 110, low: 95, close: 104, volume: 1200 }]) {
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

function restorePrototypeProperty(name: "clientWidth" | "clientHeight", descriptor: PropertyDescriptor | undefined) {
  if (descriptor) {
    Object.defineProperty(HTMLElement.prototype, name, descriptor);
    return;
  }
  Reflect.deleteProperty(HTMLElement.prototype, name);
}
