import { createI18n } from "vue-i18n";

import { messages } from "@/i18n/messages";
import type { LocaleCode } from "@/types/app";

const fallbackLocale: LocaleCode = "zh-CN";

export const i18n = createI18n({
  legacy: false,
  locale: fallbackLocale,
  fallbackLocale,
  messages,
});

