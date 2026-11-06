import React from "react";
import Landing from "./Landing";
import Legal from "./pages/Legal";
import PublicPage from "./public/PublicPage";
export default function PublicApp({ path }) {
  if (path === "/") return <Landing />;
  if (path === "/legal") return <Legal />;
  return <PublicPage path={path} />;
}
