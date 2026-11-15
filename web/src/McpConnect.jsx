import React, { useEffect, useId, useState } from "react";
import { ArrowRight, Check, Copy, Cable } from "lucide-react";
import { Button, Modal, Notice } from "./ui";
import { api } from "./client";
import { useData } from "./useData";
import { installPrompt } from "./mcp-config";
import { ClaudeIcon, OpenAIIcon } from "./assistant-icons";
import "./mcp.css";

const assistants = {
  codex: { name: "Codex", Icon: OpenAIIcon },
  claude: { name: "Claude Code", Icon: ClaudeIcon },
};
export default function McpConnect({ userId }) {
  const { data: connection, error, reload } = useData("/console/mcp");
  const [open, setOpen] = useState(false);
  const [client, setClient] = useState("codex");
  const [copied, setCopied] = useState(false);
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState("");
  const [fallback, setFallback] = useState("");
  const id = useId();
  useEffect(() => {
    if (!userId || !connection) return;
    const query = new URLSearchParams(location.search);
    const requested = query.get("connect");
    if (assistants[requested]) {
      setClient(requested);
      setOpen(true);
      query.delete("connect");
      history.replaceState(
        history.state,
        "",
        location.pathname + (query.size ? "?" + query : "") + location.hash,
      );
      return;
    }
    if (connection.connected) return;
    try {
      const key = `vitruko:mcp-intro:${userId}`;
      if (!localStorage.getItem(key)) {
        localStorage.setItem(key, "seen");
        setOpen(true);
      }
    } catch {
      /* The summary banner stays available without browser storage. */
    }
  }, [connection, userId]);
  useEffect(() => {
    if (!open || !copied || connection?.connected) return;
    const timer = setInterval(() => {
      reload().catch(() => {});
    }, 5000);
    return () => clearInterval(timer);
  }, [open, copied, connection?.connected, reload]);
  const { name: assistant, Icon } = assistants[client];
  function choose(value) {
    setClient(value);
    setCopied(false);
    setFailed("");
    setFallback("");
  }
  async function copy() {
    setBusy(true);
    setFailed("");
    setFallback("");
    try {
      const value = await api("/console/mcp/prepare", "POST", {
        renew: Boolean(connection?.revoked),
      });
      const prompt = installPrompt(client, value);
      try {
        await navigator.clipboard.writeText(prompt);
        setCopied(true);
      } catch {
        setFallback(prompt);
        setFailed(
          "No pudimos acceder al portapapeles. Selecciona y copia las instrucciones privadas de abajo.",
        );
      }
      await reload();
    } catch (e) {
      setFailed(e.message);
      if (e.code === "revoked") await reload().catch(() => {});
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <section className="mcp-banner" aria-label="Conecta tu asistente">
        <div className="mcp-marks" aria-hidden="true">
          <span>
            <ClaudeIcon />
          </span>
          <span>
            <OpenAIIcon />
          </span>
        </div>
        <div className="mcp-banner-copy">
          {connection?.connected && (
            <span className="mcp-connected">Asistente conectado</span>
          )}
          <h2>Conecta tu asistente. Aprovecha Vitruko al máximo.</h2>
          <p>
            Conecta Codex o Claude Code e integra fichas de productos con una
            conversación.
          </p>
        </div>
        <Button variant="secondary" onClick={() => setOpen(true)}>
          {connection?.connected
            ? "Conectar otro asistente"
            : "Conectar mi asistente"}
          <ArrowRight size={17} />
        </Button>
      </section>
      {open && (
        <Modal
          title="Conecta tu asistente"
          onClose={() => {
            setOpen(false);
            setFallback("");
          }}
        >
          <div className="mcp-connect-dialog">
            <div
              className="mcp-tabs"
              role="tablist"
              aria-label="Elige tu asistente"
            >
              {Object.entries(assistants).map(([value, item], index) => (
                <button
                  type="button"
                  role="tab"
                  key={value}
                  id={`${id}-${value}`}
                  aria-controls={`${id}-panel`}
                  aria-selected={client === value}
                  tabIndex={client === value ? 0 : -1}
                  onClick={() => choose(value)}
                  onKeyDown={(e) => {
                    const values = Object.keys(assistants);
                    let next;
                    if (e.key === "ArrowRight")
                      next = (index + 1) % values.length;
                    if (e.key === "ArrowLeft")
                      next = (index + values.length - 1) % values.length;
                    if (e.key === "Home") next = 0;
                    if (e.key === "End") next = values.length - 1;
                    if (next !== undefined) {
                      e.preventDefault();
                      choose(values[next]);
                      e.currentTarget.parentElement
                        .querySelectorAll('[role="tab"]')
                        [next].focus();
                    }
                  }}
                >
                  <item.Icon />
                  {item.name}
                </button>
              ))}
            </div>
            <div
              role="tabpanel"
              id={`${id}-panel`}
              aria-labelledby={`${id}-${client}`}
            >
              <div className="mcp-flow" aria-hidden="true">
                <span className="mcp-node">
                  <Cable size={26} />
                </span>
                <span className="mcp-track">
                  <span />
                </span>
                <span className="mcp-node">
                  <Icon />
                </span>
              </div>
              <ol className="mcp-steps">
                <li>
                  <span>1</span>
                  <div>
                    <strong>Copia tu conexión</strong>
                    <p>
                      El botón incluye la configuración y tu acceso personal,
                      sin mostrar la clave aquí.
                    </p>
                  </div>
                </li>
                <li>
                  <span>2</span>
                  <div>
                    <strong>Pégala en {assistant}</strong>
                    <p>
                      Abre una conversación en la app local y envía lo que
                      copiaste. Tu asistente hará la instalación.
                    </p>
                  </div>
                </li>
                <li>
                  <span>3</span>
                  <div>
                    <strong>Empieza a crear</strong>
                    <p>
                      Si te lo pide, vuelve a abrir el asistente. Después dile:
                      «Comprueba mi conexión con Vitruko». Aquí confirmaremos
                      cuando use tu conexión.
                    </p>
                  </div>
                </li>
              </ol>
              <div className="mcp-copy-action">
                {connection?.revoked && (
                  <p>
                    Revocaste esta conexión. Puedes crear una nueva para volver
                    a conectar tu asistente.
                  </p>
                )}
                <Button busy={busy} disabled={busy} onClick={copy}>
                  {copied ? <Check size={17} /> : <Copy size={17} />}{" "}
                  {connection?.revoked
                    ? "Crear nueva conexión y copiar"
                    : copied
                      ? "Copiar de nuevo"
                      : "Copiar conexión e instrucciones"}
                </Button>
                <Notice error>{error || failed}</Notice>
                {copied && (
                  <p role="status" className="mcp-connected">
                    {connection?.connected
                      ? "Conexión verificada. Tu asistente ya tiene acceso a Vitruko."
                      : `Ahora pega las instrucciones en ${assistant}. Esperando la primera conexión…`}
                  </p>
                )}
                {fallback && (
                  <label className="mcp-fallback">
                    Instrucciones privadas
                    <textarea
                      readOnly
                      rows={6}
                      value={fallback}
                      onFocus={(e) => e.target.select()}
                    />
                  </label>
                )}
              </div>
            </div>
          </div>
        </Modal>
      )}
    </>
  );
}
