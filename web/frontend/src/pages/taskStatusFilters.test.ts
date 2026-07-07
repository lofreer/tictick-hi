import { describe, expect, it } from "vitest";

import { taskMatchesStatusFilter, taskStatusFilterFromQuery, taskStatusQueryValue } from "@/pages/taskStatusFilters";

describe("task status filters", () => {
  it("normalizes task status query values", () => {
    expect(taskStatusFilterFromQuery("failed")).toBe("failed");
    expect(taskStatusFilterFromQuery("pending")).toBe("pending");
    expect(taskStatusFilterFromQuery("running")).toBe("running");
    expect(taskStatusFilterFromQuery("succeeded")).toBe("succeeded");
    expect(taskStatusFilterFromQuery("cancelled")).toBe("cancelled");
    expect(taskStatusFilterFromQuery("gap")).toBe("all");
    expect(taskStatusFilterFromQuery(["failed"])).toBe("all");
    expect(taskStatusQueryValue("all")).toBeUndefined();
    expect(taskStatusQueryValue("failed")).toBe("failed");
  });

  it("matches exact task statuses while keeping all as a pass-through", () => {
    expect(taskMatchesStatusFilter({ status: "failed" }, "failed")).toBe(true);
    expect(taskMatchesStatusFilter({ status: "running" }, "running")).toBe(true);
    expect(taskMatchesStatusFilter({ status: "succeeded" }, "failed")).toBe(false);
    expect(taskMatchesStatusFilter({ status: "cancelled" }, "all")).toBe(true);
  });
});
