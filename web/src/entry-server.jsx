import React from "react";
import { renderToString } from "react-dom/server";
import PublicApp from "./PublicApp";
export function render(path) {
  return renderToString(<PublicApp path={path} />);
}
