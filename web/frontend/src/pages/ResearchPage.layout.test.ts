import { describe, expect, it } from "vitest";

import source from "./ResearchPage.vue?raw";

describe("ResearchPage chart layout contract", () => {
  it("keeps the chart in a fixed flex viewport below the sync list", () => {
    const panelStyle = cssBlock(source, ".research-chart-panel");
    const bodyStyle = cssBlock(source, ".research-chart-body");

    expect(panelStyle).toContain("display: flex;");
    expect(panelStyle).toContain("flex-direction: column;");
    expect(panelStyle).not.toContain("display: grid;");
    expect(panelStyle).not.toContain("grid-template-rows");
    expect(bodyStyle).toContain("flex: 1 1 0;");
    expect(bodyStyle).toContain("height: 0;");
    expect(cssDeclarations(bodyStyle)).not.toContain("height: 100%");
    expect(bodyStyle).toContain("contain: strict;");
  });
});

function cssBlock(source: string, selector: string) {
  const start = source.indexOf(`${selector} {`);
  if (start < 0) {
    throw new Error(`missing ${selector} style block`);
  }
  const bodyStart = source.indexOf("{", start) + 1;
  const bodyEnd = source.indexOf("}", bodyStart);
  if (bodyEnd < 0) {
    throw new Error(`unterminated ${selector} style block`);
  }
  return source.slice(bodyStart, bodyEnd);
}

function cssDeclarations(block: string) {
  return block
    .split(";")
    .map((line) => line.trim())
    .filter(Boolean);
}
