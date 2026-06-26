import { defineStore } from "pinia";
import { ref } from "vue";

import type { LocaleCode } from "@/types/app";

const storageKey = "tictick-hi.locale";

function readLocale(): LocaleCode {
  const stored = localStorage.getItem(storageKey);
  return stored === "en-US" || stored === "zh-CN" ? stored : "zh-CN";
}

export const useLocaleStore = defineStore("locale", () => {
  const locale = ref<LocaleCode>(readLocale());

  function setLocale(nextLocale: LocaleCode) {
    locale.value = nextLocale;
    localStorage.setItem(storageKey, nextLocale);
    document.documentElement.lang = nextLocale;
  }

  function toggleLocale() {
    setLocale(locale.value === "zh-CN" ? "en-US" : "zh-CN");
  }

  setLocale(locale.value);

  return { locale, setLocale, toggleLocale };
});

