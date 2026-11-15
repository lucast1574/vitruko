// Only the two public paid plans may be carried from marketing to billing.
export function authDestination(mode, search) {
  const plan = new URLSearchParams(search).get("plan");
  if (["launch", "scale"].includes(plan)) return "/app/billing?plan=" + plan;
  const connect = new URLSearchParams(search).get("connect");
  return ["codex", "claude"].includes(connect) ? "/app?connect=" + connect : "/app";
}
