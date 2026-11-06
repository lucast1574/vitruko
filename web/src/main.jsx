import React, { lazy, Suspense } from "react";
import { createRoot, hydrateRoot } from "react-dom/client";
import PublicApp from "./PublicApp";
import "./style.css";
const Dashboard = lazy(() => import("./Dashboard"));
const Auth = lazy(() => import("./Auth"));
const path = location.pathname.replace(/\/$/, "") || "/";
const root = document.getElementById("root");
let content;
if (path === "/app" || path.startsWith("/app/"))
  content = (
    <Suspense
      fallback={
        <div className="page-loading" role="status">
          Cargando tu cuenta…
        </div>
      }
    >
      <Dashboard />
    </Suspense>
  );
else if (["/login", "/signup", "/forgot", "/reset"].includes(path))
  content = (
    <Suspense
      fallback={
        <div className="page-loading" role="status">
          Cargando acceso…
        </div>
      }
    >
      <Auth />
    </Suspense>
  );
else content = <PublicApp path={path} />;
if (root.dataset.prerendered === "true") hydrateRoot(root, content);
else createRoot(root).render(content);
