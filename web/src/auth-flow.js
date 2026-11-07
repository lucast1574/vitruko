// Only the two public paid plans may be carried from marketing to billing.
export function authDestination(mode, search) {
  const plan = new URLSearchParams(search).get("plan");
  if (["launch", "scale"].includes(plan)) return "/app/billing?plan=" + plan;
  return mode === "signup" ? "/app/keys" : "/app";
}
