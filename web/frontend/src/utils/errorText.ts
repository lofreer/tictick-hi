const externalURLPattern = /https?:\/\/[^\s"'<>]+/g;
const externalTransportErrorPattern = /\b(?:Get|Post|Put|Patch|Delete|Head|Options) "(https?:\/\/[^"]+)": ([^;]+)/g;

export function sanitizeExternalError(value?: string) {
  const normalized = value?.replace(/\s+/g, " ").trim() ?? "";
  if (!normalized) return undefined;
  return normalized
    .replace(externalTransportErrorPattern, (_raw, url, reason) => `${externalHost(url)}: ${String(reason).trim()}`)
    .replace(externalURLPattern, externalHost);
}

function externalHost(raw: string) {
  try {
    const url = new URL(raw);
    return url.host || "[external-url]";
  } catch {
    return "[external-url]";
  }
}
