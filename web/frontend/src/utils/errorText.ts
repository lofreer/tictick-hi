const externalURLPattern = /https?:\/\/[^\s"'<>]+/g;

export function sanitizeExternalError(value?: string) {
  const normalized = value?.replace(/\s+/g, " ").trim() ?? "";
  if (!normalized) return undefined;
  return normalized.replace(externalURLPattern, (raw) => {
    try {
      const url = new URL(raw);
      return url.host || "[external-url]";
    } catch {
      return "[external-url]";
    }
  });
}
