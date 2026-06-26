import { defineStore } from "pinia";
import { ref } from "vue";

import { applyThemeAttribute } from "@/theme/tokens";
import type { ThemeMode } from "@/types/app";

const storageKey = "tictick-hi.theme";

function readTheme(): ThemeMode {
  const stored = localStorage.getItem(storageKey);
  return stored === "dark" || stored === "light" ? stored : "light";
}

export const useThemeStore = defineStore("theme", () => {
  const mode = ref<ThemeMode>(readTheme());

  function setTheme(nextMode: ThemeMode) {
    mode.value = nextMode;
    localStorage.setItem(storageKey, nextMode);
    applyThemeAttribute(nextMode);
  }

  function toggleTheme() {
    setTheme(mode.value === "dark" ? "light" : "dark");
  }

  setTheme(mode.value);

  return { mode, setTheme, toggleTheme };
});

