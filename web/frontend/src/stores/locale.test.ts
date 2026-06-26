import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, it } from "vitest";

import { useLocaleStore } from "@/stores/locale";

describe("locale store", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.lang = "";
    setActivePinia(createPinia());
  });

  it("toggles and persists console locale", () => {
    const store = useLocaleStore();

    store.toggleLocale();

    expect(store.locale).toBe("en-US");
    expect(localStorage.getItem("tictick-hi.locale")).toBe("en-US");
    expect(document.documentElement.lang).toBe("en-US");
  });
});

