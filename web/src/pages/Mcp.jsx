import React, { useState } from "react";
import { Cable, Copy, Check, ArrowUpRight, KeyRound } from "lucide-react";
import { Button, Notice } from "../ui";
import { api } from "../client";
import { useData } from "../useData";
import { installPrompt } from "../mcp-config";
import "../mcp.css";

export default function Mcp() {
  const { data, error, reload } = useData("/console/mcp");
  const [busy, setBusy] = useState("");
  const [message, setMessage] = useState("");
  const [localError, setError] = useState("");
  const [fallback, setFallback] = useState("");
  async function copy(client) {
    setBusy(client);
    setError("");
    setMessage("");
    setFallback("");
    try {
      const connection = await api("/console/mcp/prepare", "POST", {
        renew: Boolean(data?.revoked),
      });
      const prompt = installPrompt(client, connection);
      try {
        await navigator.clipboard.writeText(prompt);
        setMessage(
          `Listo. Pégalo en ${client === "codex" ? "Codex" : "Claude Code"} para conectar tu cuenta.`,
        );
      } catch {
        setFallback(prompt);
        setMessage(
          "Selecciona y copia el texto de abajo. Incluye tu clave privada.",
        );
      }
      await reload();
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy("");
    }
  }
  return (
    <>
      <div className="page-title">
        <div>
          <span className="mcp-eyebrow">TU ASISTENTE, CON CONTEXTO</span>
          <h1>Conecta Vitruko a tu IA.</h1>
          <p>Copia una vez. Pega en tu asistente. Empieza a construir.</p>
        </div>
        <Cable size={32} aria-hidden="true" />
      </div>
      <Notice error>{error || localError}</Notice>
      <div className="mcp-status">
        <span className={`status-pill ${data?.ready ? "good" : "neutral"}`}>
          {!data
            ? "Cargando conexión…"
            : data.revoked
              ? "Conexión revocada"
              : data.connected
                ? "Conexión utilizada"
                : data.ready
                  ? "Lista para conectar"
                  : "Lista para preparar"}
        </span>
        <p>
          Usa tu cuenta actual y los límites de tu plan. No necesitas crear una
          API key.
        </p>
      </div>
      <div className="mcp-clients">
        {[
          [
            "codex",
            "Codex",
            "Pega las instrucciones en Codex en tu computadora.",
          ],
          [
            "claude",
            "Claude Code",
            "Pega las instrucciones en una sesión de Claude Code.",
          ],
        ].map(([id, title, text]) => (
          <section className="mcp-client" key={id}>
            <span className="mcp-eyebrow">CONECTAR CON</span>
            <h2>{title}</h2>
            <p>{text}</p>
            <Button
              disabled={!data || Boolean(busy)}
              busy={busy === id}
              onClick={() => copy(id)}
            >
              <Copy size={17} />
              {data?.revoked ? "Renovar y copiar" : `Copiar para ${title}`}
            </Button>
          </section>
        ))}
      </div>
      <Notice>
        {message && (
          <>
            <Check size={16} aria-hidden="true" /> {message}
          </>
        )}
      </Notice>
      {fallback && (
        <label className="mcp-fallback">
          Instrucciones privadas
          <textarea
            readOnly
            rows={10}
            value={fallback}
            onFocus={(e) => e.target.select()}
          />
          <Button variant="secondary" onClick={() => setFallback("")}>
            Ocultar instrucciones
          </Button>
        </label>
      )}
      <section className="mcp-capabilities">
        <h2>De la idea a tu integración.</h2>
        <p>
          Convierte enlaces públicos de productos en fichas con imágenes, marca,
          precio y disponibilidad cuando la tienda los publica. Tu asistente
          conoce el contrato, los campos faltantes y los límites de extracción.
        </p>
        <div className="mcp-example">
          <span>Prueba a decirle</span>
          <blockquote>
            Integra Vitruko en mi catálogo: al pegar un enlace de producto,
            completa la ficha y permite editar los campos que falten.
          </blockquote>
        </div>
        <ol>
          <li>Copia las instrucciones para tu asistente.</li>
          <li>Pégalas y deja que configure la conexión personal.</li>
          <li>
            Recarga el MCP si te lo pide y verifica con <code>vitruko_me</code>.
          </li>
        </ol>
      </section>
      <div className="mcp-security">
        <KeyRound size={20} />
        <div>
          <h2>Tu conexión sigue bajo tu control.</h2>
          <p>
            La clave ya va incluida en el texto copiado. Compártelo solo con tu
            asistente; no lo publiques ni lo guardes en Git. La misma conexión
            sirve para ambos asistentes. Puedes revocarla desde API keys.
          </p>
          <a className="text-link" href="/app/keys">
            Gestionar acceso <ArrowUpRight size={16} />
          </a>{" "}
          ·{" "}
          <a
            className="text-link"
            href="https://vitruko.lat/mcp"
            target="_blank"
            rel="noreferrer"
          >
            Guía del MCP
          </a>
        </div>
      </div>
    </>
  );
}
