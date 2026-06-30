import type { Time } from "lightweight-charts";

import type { ChartCandle } from "@/types/app";

export type ChartReadoutDirection = "up" | "down" | "flat";

export type ChartReadout = {
  change: string;
  changePct: string;
  close: string;
  direction: ChartReadoutDirection;
  high: string;
  low: string;
  open: string;
  timeLabel: string;
  volume: string;
};

export function chartReadoutFromCandle(candle: ChartCandle | null | undefined): ChartReadout | null {
  if (!candle) return null;
  const change = candle.close - candle.open;
  const changePct = candle.open === 0 ? 0 : (change / candle.open) * 100;
  return {
    change: signed(formatChartReadoutPrice(change)),
    changePct: `${signed(formatChartReadoutPercent(changePct))}%`,
    close: formatChartReadoutPrice(candle.close),
    direction: readoutDirection(change),
    high: formatChartReadoutPrice(candle.high),
    low: formatChartReadoutPrice(candle.low),
    open: formatChartReadoutPrice(candle.open),
    timeLabel: formatReadoutTime(candle.time),
    volume: formatChartReadoutVolume(candle.volume),
  };
}

export function chartCandleForTime(candles: ChartCandle[], time: Time | undefined): ChartCandle | null {
  if (typeof time !== "number") return null;
  return candles.find((candle) => candle.time === time) ?? null;
}

function readoutDirection(change: number): ChartReadoutDirection {
  if (change > 0) return "up";
  if (change < 0) return "down";
  return "flat";
}

function signed(value: string) {
  if (value === "0" || value.startsWith("-")) return value;
  return `+${value}`;
}

function formatReadoutTime(time: number) {
  const date = new Date(time * 1000);
  if (Number.isNaN(date.getTime())) return "-";
  return `${date.getUTCFullYear()}-${pad2(date.getUTCMonth() + 1)}-${pad2(date.getUTCDate())} ${pad2(date.getUTCHours())}:${pad2(date.getUTCMinutes())} UTC`;
}

function formatChartReadoutPrice(value: number) {
  if (!Number.isFinite(value)) return "-";
  const absolute = Math.abs(value);
  if (absolute === 0) return "0";
  if (absolute >= 1000) return trimTrailingZeros(value.toFixed(2));
  if (absolute >= 1) return trimTrailingZeros(value.toFixed(4));
  if (absolute >= 0.01) return trimTrailingZeros(value.toFixed(6));
  return trimTrailingZeros(value.toPrecision(4));
}

function formatChartReadoutPercent(value: number) {
  if (!Number.isFinite(value)) return "0";
  return trimTrailingZeros(value.toFixed(2));
}

function formatChartReadoutVolume(value: number) {
  if (!Number.isFinite(value)) return "-";
  return Intl.NumberFormat("en-US", { maximumFractionDigits: 4 }).format(value);
}

function trimTrailingZeros(value: string) {
  return value.replace(/(\.\d*?[1-9])0+$/, "$1").replace(/\.0+$/, "");
}

function pad2(value: number) {
  return value.toString().padStart(2, "0");
}
