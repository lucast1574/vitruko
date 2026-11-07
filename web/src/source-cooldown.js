export function sourceRetryDeadline(header, now = Date.now()) {
  if (/^\d+$/.test(header || "")) {
    const seconds = Number(header);
    if (Number.isSafeInteger(seconds)) return now + Math.min(86400, Math.max(1, seconds)) * 1000;
  }
  const date = Date.parse(header);
  return Number.isFinite(date) && date > now ? Math.min(date, now + 86400000) : now + 60000;
}
export function sourceHost(raw) {
  try { return new URL(raw).hostname; } catch { return ""; }
}
