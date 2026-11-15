import React from "react";
import { ArrowRight, Cable } from "lucide-react";
import { appURL } from "../client";
import { PublicHeader, PublicFooter } from "./PublicPage";
import "../mcp.css";
export function McpPromo() {
  return (
    <section className="mcp-promo" id="mcp">
      <div>
        <span className="mcp-eyebrow">MCP · CLAUDE CODE Y CODEX</span>
        <h2>De un enlace a tu catálogo, con tu IA.</h2>
        <p>
          Convierte enlaces públicos de productos en fichas con imágenes, marca,
          precio y disponibilidad cuando la tienda los publica. Tu asistente
          conoce el contrato, los campos faltantes y los límites de extracción.
        </p>
        <a className="button" href={`${appURL}/app/mcp`}>
          Conectar mi asistente <ArrowRight size={18} />
        </a>
        <a className="text-link" href="/mcp">
          Cómo funciona
        </a>
      </div>
      <div className="mcp-example">
        <Cable size={28} aria-hidden="true" />
        <blockquote>
          Integra Vitruko en mi catálogo: al pegar un enlace de producto,
          completa la ficha y permite editar los campos que falten.
        </blockquote>
        <p>Una copia desde tu panel. Tu cuenta ya va conectada.</p>
      </div>
    </section>
  );
}
export default function McpPage() {
  return (
    <div className="public-page">
      <PublicHeader />
      <main className="mcp-public">
        <nav className="breadcrumbs" aria-label="Ruta de navegación">
          <a href="/">Vitruko</a>
          <span>/</span>
          <span>MCP</span>
        </nav>
        <span className="mcp-eyebrow">MODEL CONTEXT PROTOCOL</span>
        <h1>Conecta Vitruko a Claude Code y Codex.</h1>
        <p>
          Convierte enlaces públicos de productos en fichas con imágenes, marca,
          precio y disponibilidad cuando la tienda los publica. Tu asistente
          conoce el contrato, los campos faltantes y los límites de extracción.
        </p>
        <a className="button" href={`${appURL}/app/mcp`}>
          Conectar desde mi panel <ArrowRight size={18} />
        </a>
        <section>
          <h2>Tu cuenta, lista para copiar.</h2>
          <ol>
            <li>
              Inicia sesión en{" "}
              <a href="https://app.vitruko.lat">app.vitruko.lat</a> y abre
              Conectar IA.
            </li>
            <li>
              Elige tu asistente y copia las instrucciones. Tu API key privada
              ya viene incluida.
            </li>
            <li>
              Pega el texto en Codex o Claude Code en tu computadora y deja que
              configure el MCP.
            </li>
            <li>
              Recarga la conexión cuando el asistente lo indique. Verifica tu
              cuenta con <code>vitruko_me</code>.
            </li>
          </ol>
          <p>
            No necesitas crear claves manualmente ni instalar un servidor. La
            conexión utiliza MCP por HTTPS.
          </p>
        </section>
        <section>
          <h2>Un asistente que conoce la API.</h2>
          <p>
            El MCP incluye herramientas para consultar tu cuenta y consumo, leer
            el contrato OpenAPI completo y ejecutar las operaciones de Vitruko.
            Consulta <code>vitruko_documentation</code> antes de generar una
            integración.
          </p>
          <p>
            Convierte enlaces públicos de productos en fichas con imágenes,
            marca, precio y disponibilidad cuando la tienda los publica. Tu
            asistente conoce el contrato, los campos faltantes y los límites de
            extracción.
          </p>
          <p>
            Prueba: “Integra Vitruko en mi catálogo: al pegar un enlace de
            producto, completa la ficha y permite editar los campos que falten.”
          </p>
        </section>
        <section>
          <h2>Dominios de producción y documentación.</h2>
          <p>
            Sitio: <a href="https://vitruko.lat">vitruko.lat</a>. Panel:{" "}
            <a href="https://app.vitruko.lat">app.vitruko.lat</a>. API:{" "}
            <code>https://api.vitruko.lat</code>.
          </p>
          <p>
            Endpoint MCP: <code>https://api.vitruko.lat/mcp</code>. Requiere{" "}
            <code>Authorization: Bearer TU_API_KEY</code>; la clave se envía en
            una cabecera, nunca en la URL.
          </p>
          <a className="text-link" href="/documentacion">
            Referencia y ejemplos de la API
          </a>{" "}
          · <a href="https://api.vitruko.lat/openapi.json">Contrato OpenAPI</a>
        </section>
        <section>
          <h2>Los mismos permisos y límites de tu plan.</h2>
          <p>
            Conectar el asistente no cambia tu suscripción. Consultar tu cuenta
            y la documentación no consume cuota. Las operaciones de datos
            consumen lo mismo que una petición directa a la API, incluidas las
            respuestas desde caché y los errores de proveedor admitidos. Respeta
            Retry-After cuando la API solicite esperar.
          </p>
          <p>
            La conexión se comparte entre tus asistentes y puedes revocarla
            desde API keys. El texto copiado contiene una credencial privada:
            úsalo solo en tu asistente y conserva sus confirmaciones.
          </p>
        </section>
      </main>
      <PublicFooter />
    </div>
  );
}
