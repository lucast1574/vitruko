import React, { useState } from "react";
import { Plus, Copy, Check, KeyRound, Trash2 } from "lucide-react";
import { useData } from "../useData";
import { api, date } from "../client";
import { Button, Modal, Notice, Empty } from "../ui";
export default function Keys() {
  const { data, error, reload } = useData("/console/keys");
  const [modal, setModal] = useState(""),
    [secret, setSecret] = useState(null),
    [target, setTarget] = useState(null),
    [busy, setBusy] = useState(false),
    [localError, setError] = useState(""),
    [copied, setCopied] = useState(false);
  async function create(e) {
    e.preventDefault();
    setBusy(true);
    setError("");
    const f = new FormData(e.currentTarget);
    try {
      const v = await api("/console/keys", "POST", {
        name: f.get("name"),
        scopes: f.getAll("scopes"),
      });
      setSecret(v);
      setModal("secret");
      await reload();
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  }
  async function revoke() {
    setBusy(true);
    setError("");
    try {
      await api("/console/keys/" + target.id, "DELETE");
      setModal("");
      setTarget(null);
      await reload();
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  }
  const active = data?.keys.filter((k) => !k.revokedAt) || [];
  return (
    <>
      <div className="page-title">
        <div>
          <h1>Las llaves de tus proyectos</h1>
          <p>Permisos específicos. Consumo compartido con tu cuenta.</p>
        </div>
        <Button
          onClick={() => {
            setError("");
            setModal("create");
          }}
        >
          <Plus size={17} />
          Crear API key
        </Button>
      </div>
      <Notice error>{error || localError}</Notice>
      <div className="key-summary">
        <KeyRound size={20} />
        <span>{active.length} de 10 claves activas</span>
        <p>
          La clave completa solo se muestra al crearla. Nunca la publiques en un
          repositorio.
        </p>
      </div>
      {data?.keys.length ? (
        <div className="table-scroll">
          <table className="keys-table">
            <thead>
              <tr>
                <th>Proyecto</th>
                <th>Clave</th>
                <th>Permisos</th>
                <th>Último uso</th>
                <th>Estado</th>
                <th aria-label="Acciones" />
              </tr>
            </thead>
            <tbody>
              {data.keys.map((k) => (
                <tr key={k.id}>
                  <td>
                    <strong>{k.name}</strong>
                    <small>Creada el {date(k.createdAt)}</small>
                  </td>
                  <td>
                    <code>{k.prefix}</code>
                  </td>
                  <td>
                    <span className="scope-list">{k.scopes.join(" · ")}</span>
                  </td>
                  <td>{k.lastUsedAt ? date(k.lastUsedAt) : "Sin uso"}</td>
                  <td>
                    <span
                      className={`status-pill ${k.revokedAt ? "neutral" : "good"}`}
                    >
                      {k.revokedAt ? "Revocada" : "Activa"}
                    </span>
                  </td>
                  <td>
                    {!k.revokedAt && (
                      <button
                        className="icon-button"
                        aria-label={`Revocar ${k.name}`}
                        onClick={() => {
                          setTarget(k);
                          setModal("revoke");
                        }}
                      >
                        <Trash2 size={17} />
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <Empty
          title="Tu primera integración empieza con una clave"
          action={
            <Button onClick={() => setModal("create")}>
              <Plus size={16} />
              Crear mi primera clave
            </Button>
          }
        >
          Ponle el nombre de tu proyecto y elige qué operaciones podrá usar.
        </Empty>
      )}
      <div className="code-guide">
        <h2>Así se conecta</h2>
        <pre>{`curl "https://api-vitruko.nexode.app/v1/extract" \\\n  -H "X-API-Key: TU_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d '{"url":"https://tienda.com/products/articulo"}'`}</pre>
        <p>
          También puedes enviar <code>Authorization: Bearer TU_API_KEY</code>.
          Las claves revocadas dejan de aceptar nuevas peticiones.
        </p>
      </div>
      {modal === "create" && (
        <Modal title="Nueva API key" onClose={() => setModal("")}>
          <form onSubmit={create}>
            <label>
              Nombre del proyecto
              <input
                name="name"
                placeholder="Ej. Mi aplicación de entregas"
                required
                minLength={2}
                maxLength={80}
                autoFocus
              />
            </label>
            <fieldset>
              <legend>Permisos</legend>
              {[
                ["extract", "Extraer fichas de producto"],
              ].map(([v, label]) => (
                <label className="check-row" key={v}>
                  <input
                    type="checkbox"
                    name="scopes"
                    value={v}
                    defaultChecked
                  />
                  {label}
                </label>
              ))}
            </fieldset>
            <Notice error>{localError}</Notice>
            <Button busy={busy}>Crear clave</Button>
          </form>
        </Modal>
      )}
      {modal === "secret" && secret && (
        <Modal
          title="Guarda tu API key"
          onClose={() => {
            setSecret(null);
            setModal("");
            setCopied(false);
          }}
        >
          <p>
            Esta es la única vez que verás la clave completa. Cópiala y guárdala
            en un lugar seguro.
          </p>
          <textarea
            className="secret-value"
            readOnly
            value={secret.key}
            aria-label="Nueva API key"
            onFocus={(e) => e.target.select()}
          />
          <Button
            onClick={async () => {
              try {
                await navigator.clipboard.writeText(secret.key);
                setCopied(true);
              } catch {
                setError("Selecciona la clave y cópiala manualmente.");
              }
            }}
          >
            {copied ? <Check size={16} /> : <Copy size={16} />}{" "}
            {copied ? "Clave copiada" : "Copiar clave"}
          </Button>
          <Notice error>{localError}</Notice>
        </Modal>
      )}
      {modal === "revoke" && target && (
        <Modal title="Revocar API key" onClose={() => setModal("")}>
          <p>
            Las nuevas peticiones con <strong>{target.name}</strong> serán
            rechazadas. Esta acción no se puede deshacer.
          </p>
          <Notice error>{localError}</Notice>
          <div className="modal-actions">
            <Button variant="secondary" onClick={() => setModal("")}>
              Conservar clave
            </Button>
            <Button variant="danger" busy={busy} onClick={revoke}>
              Revocar clave
            </Button>
          </div>
        </Modal>
      )}
    </>
  );
}
