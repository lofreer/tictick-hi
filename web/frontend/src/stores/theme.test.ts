import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, it } from "vitest";

import { useThemeStore } from "@/stores/theme";

describe("theme store", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute("data-theme");
    setActivePinia(createPinia());
  });

  it("persists and applies the selected theme", () => {
    const store = useThemeStore();

    store.setTheme("dark");

    expect(store.mode).toBe("dark");
    expect(localStorage.getItem("tictick-hi.theme")).toBe("dark");
    expect(document.documentElement.dataset.theme).toBe("dark");
  });
});

