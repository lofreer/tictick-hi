import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import TradingViewChart from "@/components/chart/TradingViewChart.vue";
import { hasDistortedChartCanvasGeometry } from "@/components/chart/chartCanvasGeometry";

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
  setMarkers: vi.fn(),
  setVisibleLogicalRange: vi.fn(),
  setVolumeData: vi.fn(),
  subscribeCrosshairMove: vi.fn(),
  unsubscribeCrosshairMove: vi.fn(),
}));

vi.mock("lightweight-charts", () => ({
  CandlestickSeries: "CandlestickSeries",
  createChart: chartMocks.createChart,
  createSeriesMarkers: vi.fn(() => ({ setMarkers: chartMocks.setMarkers })),
  HistogramSeries: "HistogramSeries",
  TickMarkType: { Year: 0, Month: 1, DayOfMonth: 2, Time: 3, TimeWithSeconds: 4 },
}));

describe("TradingViewChart canvas geometry recovery", () => {
  let originalGetBoundingClientRect: typeof Element.prototype.getBoundingClientRect;
  let originalClientWidth: PropertyDescriptor | undefined;
  let originalClientHeight: PropertyDescriptor | undefined;

  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
    chartMocks.addSeries.mockImplementation((seriesType) =>
      seriesType === "HistogramSeries" ? { setData: chartMocks.setVolumeData } : { setData: chartMocks.setCandleData },
    );
    chartMocks.createChart.mockReturnValue({
      addSeries: chartMocks.addSeries,
      applyOptions: chartMocks.applyOptions,
      priceScale: chartMocks.priceScale.mockReturnValue({ applyOptions: chartMocks.applyPriceScaleOptions }),
      remove: chartMocks.remove,
      resize: chartMocks.resize,
      subscribeCrosshairMove: chartMocks.subscribeCrosshairMove,
      timeScale: vi.fn(() => ({ fitContent: chartMocks.fitContent, setVisibleLogicalRange: chartMocks.setVisibleLogicalRange })),
      unsubscribeCrosshairMove: chartMocks.unsubscribeCrosshairMove,
    });
    originalGetBoundingClientRect = Element.prototype.getBoundingClientRect;
    originalClientWidth = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "clientWidth");
    originalClientHeight = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "clientHeight");
    window.requestAnimationFrame = ((callback: FrameRequestCallback) => {
      callback(0);
      return 0;
    }) as typeof window.requestAnimationFrame;
    window.cancelAnimationFrame = vi.fn() as typeof window.cancelAnimationFrame;
    Object.defineProperty(HTMLElement.prototype, "clientWidth", {
      configurable: true,
      get() {
        return this instanceof Element && this.classList.contains("research-chart-body") ? 1180 : 0;
      },
    });
    Object.defineProperty(HTMLElement.prototype, "clientHeight", {
      configurable: true,
      get() {
        return this instanceof Element && this.classList.contains("research-chart-body") ? 700 : 0;
      },
    });
    Element.prototype.getBoundingClientRect = function getBoundingClientRect() {
      if (this instanceof Element && this.classList.contains("research-chart-body")) {
        return rect({ width: 1180, height: 700 });
      }
      return originalGetBoundingClientRect.call(this);
    };
    globalThis.ResizeObserver = class ResizeObserverTestDouble {
      observe() {}
      unobserve() {}
      disconnect() {}
    } as typeof ResizeObserver;
  });

  afterEach(() => {
    Element.prototype.getBoundingClientRect = originalGetBoundingClientRect;
    restorePrototypeProperty("clientWidth", originalClientWidth);
    restorePrototypeProperty("clientHeight", originalClientHeight);
  });

  it("forces a repaint when an internal chart canvas is CSS-scaled without a host resize", () => {
    const host = document.createElement("div");
    host.className = "research-chart-body";
    host.setAttribute("data-chart-viewport", "fixed");
    document.body.append(host);
    const wrapper = mount(TradingViewChart, {
      attachTo: host,
      props: {
        data: [{ time: 1_788_220_800, open: 100, high: 110, low: 95, close: 104, volume: 1200 }],
        emptyTitle: "No candles",
      },
    });
    const canvas = document.createElement("canvas");
    canvas.width = 64;
    canvas.height = 64;
    canvas.getBoundingClientRect = () => rect({ width: 64, height: 640 });
    wrapper.get(".trading-chart__canvas").element.append(canvas);
    chartMocks.resize.mockClear();

    window.dispatchEvent(new Event("resize"));

    expect(chartMocks.resize).toHaveBeenNthCalledWith(1, 1180, 699, true);
    expect(chartMocks.resize).toHaveBeenNthCalledWith(2, 1180, 700, true);

    wrapper.unmount();
    host.remove();
  });

  it("accepts a normal high-DPR chart canvas", () => {
    const container = document.createElement("div");
    const canvas = document.createElement("canvas");
    canvas.width = 192;
    canvas.height = 192;
    canvas.getBoundingClientRect = () => rect({ width: 64, height: 64 });
    container.append(canvas);

    expect(hasDistortedChartCanvasGeometry(container, 3)).toBe(false);
  });

  it("detects a chart canvas stretched on only one axis", () => {
    const container = document.createElement("div");
    const canvas = document.createElement("canvas");
    canvas.width = 192;
    canvas.height = 192;
    canvas.getBoundingClientRect = () => rect({ width: 64, height: 640 });
    container.append(canvas);

    expect(hasDistortedChartCanvasGeometry(container, 3)).toBe(true);
  });
});

function rect({ width, height }: { width: number; height: number }) {
  return {
    x: 0,
    y: 0,
    top: 0,
    left: 0,
    right: width,
    bottom: height,
    width,
    height,
    toJSON: () => ({}),
  } as DOMRect;
}

function restorePrototypeProperty(name: "clientWidth" | "clientHeight", descriptor: PropertyDescriptor | undefined) {
  if (descriptor) Object.defineProperty(HTMLElement.prototype, name, descriptor);
}
