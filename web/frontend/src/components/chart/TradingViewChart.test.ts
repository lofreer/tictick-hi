import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, it, vi } from "vitest";

import TradingViewChart from "@/components/chart/TradingViewChart.vue";

const resize = vi.fn();
const remove = vi.fn();
const setData = vi.fn();
const setMarkers = vi.fn();
const fitContent = vi.fn();

vi.mock("lightweight-charts", () => ({
  CandlestickSeries: "CandlestickSeries",
  createChart: vi.fn(() => ({
    addSeries: vi.fn(() => ({ setData })),
    applyOptions: vi.fn(),
    remove,
    resize,
    timeScale: vi.fn(() => ({ fitContent })),
  })),
  createSeriesMarkers: vi.fn(() => ({ setMarkers })),
}));

describe("TradingViewChart", () => {
  let observedTarget: Element | null = null;

  beforeEach(() => {
    setActivePinia(createPinia());
    observedTarget = null;
    resize.mockClear();
    remove.mockClear();
    setData.mockClear();
    setMarkers.mockClear();
    fitContent.mockClear();

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

  it("observes the stable chart root instead of the chart library canvas host", () => {
    const wrapper = mount(TradingViewChart, {
      attachTo: document.body,
      props: {
        data: [{ time: 1_788_220_800, open: 100, high: 110, low: 95, close: 104 }],
        emptyTitle: "No candles",
      },
    });

    const root = wrapper.get(".trading-chart").element;
    const canvasHost = wrapper.get(".trading-chart__canvas").element;

    expect(observedTarget).toBe(root);
    expect(observedTarget).not.toBe(canvasHost);

    wrapper.unmount();
  });
});
