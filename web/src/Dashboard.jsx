import React, { useState } from "react";
import {
  LayoutDashboard,
  KeyRound,
  ChartNoAxesCombined,
  CreditCard,
  BookOpen,
  UserRound,
  Users,
  Settings2,
  Menu,
  X,
  ArrowUpRight,
  ScanLine,
} from "lucide-react";
import { Logo, Notice } from "./ui";
import Avatar from "./Avatar";
import { useData } from "./useData";
import { Overview, Usage } from "./pages/Usage";
import Keys from "./pages/Keys";
import Billing from "./pages/Billing";
import Profile from "./pages/Profile";
import Docs from "./pages/Docs";
import Admin from "./pages/Admin";
import Inspector from "./pages/Inspector";
import { siteURL } from "./client";
const links = [
  ["", "Resumen", LayoutDashboard],
  ["keys", "API keys", KeyRound],
  ["inspector", "Extraer producto", ScanLine],
  ["usage", "Consumo", ChartNoAxesCombined],
  ["billing", "Plan y facturación", CreditCard],
  ["docs", "Documentación", BookOpen],
];
export default function Dashboard() {
  const { data: user, error } = useData("/console/session");
  const [menu, setMenu] = useState(false);
  const page = location.pathname.split("/")[2] || "";
  if (!user)
    return (
      <div className="page-loading">
        <Logo />
        <Notice error>{error}</Notice>
        <p>Cargando tu cuenta…</p>
      </div>
    );
  const items = [
    ...links,
    ...(user.role === "admin" ? [["admin", "Administración", Users]] : []),
  ];
  return (
    <div className="app-frame">
      <aside className={`sidebar ${menu ? "is-open" : ""}`}>
        <div className="sidebar-brand">
          <Logo />
          <button
            className="icon-button mobile-only"
            aria-label="Cerrar navegación"
            onClick={() => setMenu(false)}
          >
            <X size={20} />
          </button>
        </div>
        <div className="workspace">
          <Avatar name={user.name} src={user.avatarUrl} />
          <div>
            <strong>{user.name}</strong>
            <span>
              {user.role === "admin" ? "Administración" : "Cuenta personal"}
            </span>
          </div>
        </div>
        <nav aria-label="Panel">
          {items.map(([path, label, Icon]) => (
            <a
              href={`/app${path ? "/" + path : ""}`}
              className={page === path ? "active" : ""}
              key={path}
              aria-current={page === path ? "page" : undefined}
            >
              <Icon size={18} />
              {label}
            </a>
          ))}
        </nav>
        <div className="sidebar-bottom">
          <a href="/app/profile" className={page === "profile" ? "active" : ""}>
            <UserRound size={18} />
            Mi perfil
          </a>
          <a href={siteURL || "/"} target="_blank" rel="noreferrer">
            Sitio de Vitruko
            <ArrowUpRight size={15} />
          </a>
        </div>
      </aside>
      {menu && (
        <button
          className="nav-backdrop"
          aria-label="Cerrar menú"
          onClick={() => setMenu(false)}
        />
      )}
      <div className="app-body">
        <header className="app-topbar">
          <button
            className="icon-button mobile-only"
            onClick={() => setMenu(true)}
            aria-label="Abrir navegación"
          >
            <Menu size={20} />
          </button>
          <span>
            Vitruko <span className="slash">/</span>{" "}
            {items.find((v) => v[0] === page)?.[1] ||
              (page === "profile" ? "Mi perfil" : "Panel")}
          </span>
          <a href="/app/profile" className="account-shortcut">
            <span>{user.name}</span>
            <Avatar name={user.name} src={user.avatarUrl} small />
          </a>
        </header>
        <main className="app-sheet">
          {page === "keys" ? (
            <Keys />
          ) : page === "inspector" ? (
            <Inspector />
          ) : page === "usage" ? (
            <Usage />
          ) : page === "billing" ? (
            <Billing />
          ) : page === "profile" ? (
            <Profile user={user} />
          ) : page === "docs" ? (
            <Docs />
          ) : page === "admin" && user.role === "admin" ? (
            <Admin />
          ) : (
            <Overview user={user} />
          )}
        </main>
        <div className="app-footnote">Vitruko · Un enlace, listo para construir.</div>
      </div>
    </div>
  );
}
