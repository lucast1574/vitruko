import React, { useState } from "react";
import { ArrowUpRight, Copy, Check, ScanLine } from "lucide-react";
import { Button, Notice } from "../ui";
export default function Inspector({ embedded = false }) {
  const [busy, setBusy] = useState(false),
    [result, setResult] = useState(null),
    [error, setError] = useState(""),
    [cache, setCache] = useState(""),
    [copied, setCopied] = useState(false);
  async function submit(e) {
    e.preventDefault();
    setBusy(true);
    setError("");
    setResult(null);
    setCopied(false);
    const fields = new FormData(e.currentTarget);
    try {
      const res = await fetch("/v1/extract", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-API-Key": fields.get("key"),
        },
        body: JSON.stringify({ url: fields.get("url") }),
      });
      const value = await res.json();
      if (!res.ok)
        throw new Error(value.error || "No se pudo extraer la ficha.");
      setResult(value);
      setCache(res.headers.get("X-Cache"));
    } catch (err) {
      setError(err.message);
    } finally {
      setBusy(false);
    }
  }
  async function copy() {
    try {
      await navigator.clipboard.writeText(JSON.stringify(result, null, 2));
      setCopied(true);
    } catch {
      setError("Selecciona el JSON para copiarlo.");
    }
  }
  return (
    <section className="extract-tool">
      {!embedded && (
        <div className="page-title">
          <div>
            <h1>De enlace a ficha</h1>
            <p>
              Pega el producto que quieres integrar. Cada ejecución consume una
              petición.
            </p>
          </div>
          <ScanLine size={24} />
        </div>
      )}
      <form className="extract-form" onSubmit={submit}>
        <label>
          Tu API key
          <input
            name="key"
            type="password"
            autoComplete="off"
            required
            placeholder="vt_live_…"
          />
        </label>
        <label>
          Enlace público del producto
          <input
            name="url"
            type="url"
            required
            placeholder="https://tienda.com/producto"
            maxLength={2048}
          />
        </label>
        <Button busy={busy} type="submit">
          {busy ? "Leyendo la ficha…" : "Extraer producto"}
          <ArrowUpRight size={17} />
        </Button>
      </form>
      <Notice error>{error}</Notice>
      {result ? (
        <div className="extraction-result">
          <article className="extracted-product">
            {result.imageUrl ? (
              <img
                src={result.imageUrl}
                alt={result.title}
                referrerPolicy="no-referrer"
                onError={(e) => {
                  e.currentTarget.style.display = "none";
                }}
              />
            ) : (
              <p>La tienda no publicó una imagen.</p>
            )}
            <div>
              <h2>{result.title}</h2>
              <p>
                {result.description || "La tienda no publicó una descripción."}
              </p>
              <a
                className="text-link"
                href={result.url}
                target="_blank"
                rel="noreferrer"
              >
                Abrir producto <ArrowUpRight size={16} />
              </a>
            </div>
          </article>
          <div className="extracted-json">
            <div>
              <h2>Respuesta JSON</h2>
              <button
                className="icon-button"
                onClick={copy}
                aria-label={copied ? "JSON copiado" : "Copiar JSON"}
              >
                {copied ? <Check size={18} /> : <Copy size={18} />}
              </button>
            </div>
            <pre>{JSON.stringify(result, null, 2)}</pre>
            <p>
              {cache === "HIT"
                ? "Respuesta desde caché de hasta cinco minutos."
                : "Respuesta consultada en la tienda."}{" "}
              Esta ejecución cuenta una petición.
            </p>
          </div>
        </div>
      ) : (
        <div className="extract-empty">
          <h2>Tu siguiente ficha empieza aquí.</h2>
          <p>
            Se mostrarán nombre, descripción, imagen y enlace. Tu clave se usa
            en esta petición y no se guarda en el navegador.
          </p>
          <p>
            Vitruko lee información pública. Si la tienda bloquea el acceso o no
            expone la ficha, recibirás el motivo.
          </p>
        </div>
      )}
    </section>
  );
}
