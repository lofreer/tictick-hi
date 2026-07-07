/// <reference types="node" />
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

import source from "./TradingViewChart.vue?raw";

const appPackage = JSON.parse(readFileSync("package.json", "utf8")) as {
  dependencies?: Record<string, string>;
};
const lightweightChartsPackage = JSON.parse(readFileSync("node_modules/lightweight-charts/package.json", "utf8")) as {
  license?: string;
  name?: string;
};

describe("TradingViewChart package contract", () => {
  it("uses the Apache-2.0 lightweight-charts package for the first-version TradingView chart", () => {
    expect(appPackage.dependencies?.["lightweight-charts"]).toBeDefined();
    expect(lightweightChartsPackage).toMatchObject({
      license: "Apache-2.0",
      name: "lightweight-charts",
    });
    expect(source).toContain('from "lightweight-charts"');
  });

  it("does not embed proprietary TradingView widget assets", () => {
    expect(source).not.toContain("s3.tradingview.com");
    expect(source).not.toContain("charting_library");
    expect(source).not.toContain("new TradingView.widget");
    expect(source).not.toContain("<iframe");
  });
});
