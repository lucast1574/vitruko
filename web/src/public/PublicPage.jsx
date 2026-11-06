import React from "react";
import { ArrowRight, ArrowUpRight } from "lucide-react";
import { Logo } from "../ui";
import { appURL } from "../client";
import Docs from "../pages/Docs";
import { guides, pages } from "./content";
import "./public.css";

export function PublicHeader() {
  return (
    <header className="public-header">
      <Logo />
      <nav aria-label="Navegación de Vitruko">
        <a href="/">Producto</a>
        <a href="/documentacion">Documentación</a>
      </nav>
      <a className="public-account" href={`${appURL}/signup`}>
        Crea tu API key <ArrowUpRight size={17} />
      </a>
    </header>
  );
}
export function PublicFooter() {
  return (
    <footer className="public-footer">
      <Logo />
      <div>
        <a href="/documentacion">Documentación</a>
        <a href={`${appURL}/login`}>Mi cuenta</a>
        <a href="/legal">Condiciones y privacidad</a>
      </div>
      <p>Vitruko · Un enlace, listo para construir.</p>
    </footer>
  );
}
export function ResourceLinks({ current }) {
  return (
    <div className="resource-links">
      {guides
        .filter(([path]) => path !== current)
        .map(([path, p]) => (
          <a href={path} key={path}>
            <span>
              <strong>{p.navTitle}</strong>
              <span>{p.summary}</span>
            </span>
            <ArrowUpRight size={22} />
          </a>
        ))}
    </div>
  );
}
export default function PublicPage({ path }) {
  const page = pages[path];
  return (
    <div className="public-page">
      <PublicHeader />
      {path === "/documentacion" ? (
        <main className="public-documentation">
          <nav className="breadcrumbs" aria-label="Ruta de navegación">
            <a href="/">Vitruko</a>
            <span>/</span>
            <span>Documentación</span>
          </nav>
          <Docs heading={page.heading} />
          <section className="public-related">
            <h2>Guías de integración</h2>
            <ResourceLinks />
          </section>
        </main>
      ) : page ? (
        <main>
          <nav className="breadcrumbs" aria-label="Ruta de navegación">
            <a href="/">Vitruko</a>
            <span>/</span>
            <a href="/documentacion">Documentación</a>
            <span>/</span>
            <span>{page.navTitle}</span>
          </nav>
          <section className="guide-opening">
            <div>
              <h1>{page.heading}</h1>
              <p>{page.summary}</p>
              <a className="button" href={`${appURL}/signup`}>
                Crea tu API key <ArrowRight size={18} />
              </a>
            </div>
            <div className="guide-request">
              <span>Ejemplo de petición</span>
              <pre>{page.code}</pre>
              <p>
                Permiso de la clave: <code>{page.scope}</code>. Cada petición
                admitida consume una unidad del plan.
              </p>
            </div>
          </section>
          <div className="guide-body">
            {page.sections.map((s) => (
              <section key={s.title}>
                <h2>{s.title}</h2>
                <p>{s.text}</p>
              </section>
            ))}
            <section>
              <h2>Autenticación, cuotas y errores</h2>
              <p>
                Envía la clave en <code>X-API-Key</code> o{" "}
                <code>Authorization: Bearer</code>. HTTP 401 indica una clave
                ausente o inválida; 403, falta de permiso o cuenta suspendida;
                429, cuota o velocidad excedida. Consulta las cabeceras de cuota
                y respeta <code>Retry-After</code>. El panel muestra tu plan y
                consumo real.
              </p>
              <a className="text-link" href="/documentacion">
                Consultar parámetros y respuestas <ArrowRight size={18} />
              </a>
            </section>
          </div>
          <section className="public-related">
            <h2>Prepara tu siguiente integración.</h2>
            <ResourceLinks current={path} />
          </section>
        </main>
      ) : (
        <main className="public-missing">
          <h1>Esta página no existe.</h1>
          <p>
            La página que buscas no está disponible. Vuelve al producto o
            consulta cómo integrar la API.
          </p>
          <a className="button" href="/">
            Volver a Vitruko <ArrowRight size={18} />
          </a>
          <a href="/documentacion">Consultar documentación</a>
        </main>
      )}
      <PublicFooter />
    </div>
  );
}
