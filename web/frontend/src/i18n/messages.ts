import type { LocaleCode } from "@/types/app";

import { enUSMessages } from "@/i18n/messages.en";
import { enUSResearchMessages } from "@/i18n/messages.research.en";
import { zhCNResearchMessages } from "@/i18n/messages.research.zh";
import { zhCNMessages } from "@/i18n/messages.zh";

export const messages: Record<LocaleCode, Record<string, string>> = {
  "zh-CN": { ...zhCNMessages, ...zhCNResearchMessages },
  "en-US": { ...enUSMessages, ...enUSResearchMessages },
};
