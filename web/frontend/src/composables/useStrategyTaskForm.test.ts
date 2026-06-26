import { describe, expect, it } from "vitest";

import { defaultParamValues } from "@/composables/useStrategyTaskForm";
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
});
