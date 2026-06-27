import { describe, expect, it } from "vitest";

import { defaultParamValues, isStrategyParamValueValid } from "@/composables/useStrategyTaskForm";
import type { StrategyParamSpec } from "@/types/app";

describe("strategy task form", () => {
  it("creates defaults from strategy parameter specs", () => {
    const params: StrategyParamSpec[] = [
      {
        key: "fastPeriod",
        label: "Fast period",
        type: "number",
        required: true,
        default: 12,
        options: [],
      },
      {
        key: "side",
        label: "Side",
        type: "select",
        required: true,
        options: [{ label: "Both", value: "both" }],
      },
      {
        key: "enabled",
        label: "Enabled",
        type: "boolean",
        required: false,
        options: [],
      },
    ];

    expect(defaultParamValues(params)).toEqual({
      enabled: false,
      fastPeriod: 12,
      side: "both",
    });
  });

  it("validates values against strategy parameter specs", () => {
    const numberParam: StrategyParamSpec = {
      key: "fastPeriod",
      label: "Fast period",
      type: "number",
      required: true,
      default: 12,
      min: 2,
      max: 200,
      options: [],
    };
    const selectParam: StrategyParamSpec = {
      key: "signalMode",
      label: "Signal mode",
      type: "select",
      required: true,
      options: [{ label: "Order", value: "order" }],
    };

    expect(isStrategyParamValueValid(numberParam, 12)).toBe(true);
    expect(isStrategyParamValueValid(numberParam, 1)).toBe(false);
    expect(isStrategyParamValueValid(selectParam, "order")).toBe(true);
    expect(isStrategyParamValueValid(selectParam, "webhook")).toBe(false);
  });
});
