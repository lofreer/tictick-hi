import type { LocaleCode } from "@/types/app";

import { enUSMessages } from "@/i18n/messages.en";
import { zhCNMessages } from "@/i18n/messages.zh";

export const messages: Record<LocaleCode, Record<string, string>> = {
  "zh-CN": zhCNMessages,
  "en-US": enUSMessages,
};
