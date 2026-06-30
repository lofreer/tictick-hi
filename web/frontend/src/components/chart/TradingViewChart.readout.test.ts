import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { nextTick } from "vue";

import TradingViewChart from "@/components/chart/TradingViewChart.vue";
import type { ChartCandle } from "@/types/app";

const chartMocks = vi.hoisted(() => ({
  addSeries: vi.fn(),
  createChart: vi.fn(),
  fitContent: vi.fn(),
  priceScale: vi.fn(),
  remove: vi.fn(),
  resize: vi.fn(),
  setData: vi.fn(),
  setMarkers: vi.fn(),
  setVisibleLogicalRange: vi.fn(),
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

describe("TradingViewChart readout", () => {
  let crosshairHandler: ((payload: { time?: number }) => void) | null = null;
  let originalResizeObserver: typeof ResizeObserver | undefined;

  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
    crosshairHandler = null;
    chartMocks.addSeries.mockReturnValue({ setData: chartMocks.setData });
    chartMocks.priceScale.mockReturnValue({ applyOptions: vi.fn() });
    chartMocks.subscribeCrosshairMove.mockImplementation((handler) => {
      crosshairHandler = handler;
    });
    chartMocks.createChart.mockReturnValue({
      addSeries: chartMocks.addSeries,
      applyOptions: vi.fn(),
      priceScale: chartMocks.priceScale,
      remove: chartMocks.remove,
      resize: chartMocks.resize,
      subscribeCrosshairMove: chartMocks.subscribeCrosshairMove,
      timeScale: vi.fn(() => ({
        fitContent: chartMocks.fitContent,
        setVisibleLogicalRange: chartMocks.setVisibleLogicalRange,
      })),
      unsubscribeCrosshairMove: chartMocks.unsubscribeCrosshairMove,
    });
    window.requestAnimationFrame = ((callback: FrameRequestCallback) => {
      callback(0);
      return 0;
    }) as typeof window.requestAnimationFrame;
    window.cancelAnimationFrame = vi.fn() as typeof window.cancelAnimationFrame;
    originalResizeObserver = globalThis.ResizeObserver;
    globalThis.ResizeObserver = class ResizeObserverTestDouble {
      observe() {}
      unobserve() {}
      disconnect() {}
    } as typeof ResizeObserver;
  });

  afterEach(() => {
    globalThis.ResizeObserver = originalResizeObserver as typeof ResizeObserver;
  });

  it("shows the latest candle OHLCV by default", async () => {
    const wrapper = mountChart();
    await nextTick();

    expect(wrapper.get(".trading-chart__readout").text()).toContain("2026-06-28 00:01 UTC");
    expect(wrapper.get(".trading-chart__readout").text()).toContain("O 104");
    expect(wrapper.get(".trading-chart__readout").text()).toContain("H 110");
    expect(wrapper.get(".trading-chart__readout").text()).toContain("L 101");
    expect(wrapper.get(".trading-chart__readout").text()).toContain("C 99");
    expect(wrapper.get(".trading-chart__readout").text()).toContain("V 2,500.5");
    expect(wrapper.get(".trading-chart__readout").text()).toContain("-5 / -4.81%");
    expect(wrapper.get(".trading-chart__readout").classes()).toContain("trading-chart__readout--down");

    wrapper.unmount();
  });

  it("switches to the crosshair candle and resets to latest when no candle is hit", async () => {
    const wrapper = mountChart();

    crosshairHandler?.({ time: utcSeconds("2026-06-28T00:00:00Z") });
    await nextTick();

    expect(wrapper.get(".trading-chart__readout").text()).toContain("2026-06-28 00:00 UTC");
    expect(wrapper.get(".trading-chart__readout").text()).toContain("O 100");
    expect(wrapper.get(".trading-chart__readout").text()).toContain("C 104");
    expect(wrapper.get(".trading-chart__readout").text()).toContain("+4 / +4%");
    expect(wrapper.get(".trading-chart__readout").classes()).toContain("trading-chart__readout--up");

    crosshairHandler?.({});
    await nextTick();

    expect(wrapper.get(".trading-chart__readout").text()).toContain("2026-06-28 00:01 UTC");
    expect(wrapper.get(".trading-chart__readout").text()).toContain("C 99");

    wrapper.unmount();
    expect(chartMocks.unsubscribeCrosshairMove).toHaveBeenCalled();
  });
});

function mountChart(data = candles()) {
  return mount(TradingViewChart, {
    props: {
      data,
      emptyTitle: "No data",
    },
    attachTo: document.body,
  });
}

function candles(): ChartCandle[] {
  return [
    { time: utcSeconds("2026-06-28T00:00:00Z"), open: 100, high: 108, low: 96, close: 104, volume: 1200 },
    { time: utcSeconds("2026-06-28T00:01:00Z"), open: 104, high: 110, low: 101, close: 99, volume: 2500.5 },
  ];
}

function utcSeconds(value: string) {
  return Math.floor(Date.parse(value) / 1000);
}
