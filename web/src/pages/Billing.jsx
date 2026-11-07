import React, { useState, useEffect } from "react";
import { ArrowUpRight, Check, RefreshCw } from "lucide-react";
import { useData } from "../useData";
import { api, number, date } from "../client";
import { Button, Notice } from "../ui";
export default function Billing() {
  const { data: usage, error, reload } = useData("/console/usage"),
    { data: catalog, error: catalogError } = useData("/console/plans");
  const [busy, setBusy] = useState(""),
    [message, setMessage] = useState(""),
    [localError, setError] = useState("");
  async function sync() {
    setBusy("sync");
    setError("");
    setMessage("");
    try {
      await api("/console/billing/sync", "POST", {});
      await reload();
      setMessage("Facturación sincronizada con Stripe.");
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy("");
    }
  }
  useEffect(() => {
    if (new URLSearchParams(location.search).get("checkout") === "success")
      sync();
  }, []);
  async function checkout(id) {
    setBusy(id);
    setError("");
    setMessage("");
    try {
      const existing =
        sessionStorage.getItem("vitruko-checkout-" + id) || crypto.randomUUID();
      sessionStorage.setItem("vitruko-checkout-" + id, existing);
      const v = await api(
        "/console/billing/checkout",
        "POST",
        { planId: id },
        { "Idempotency-Key": existing },
      );
      if (new URL(v.url).hostname !== "checkout.stripe.com")
        throw new Error("Destino de pago inválido.");
      location.assign(v.url);
    } catch (e) {
      setError(e.message);
      setBusy("");
    }
  }
  async function portal() {
    setBusy("portal");
    setError("");
    setMessage("");
    try {
      const v = await api("/console/billing/portal", "POST", {});
      if (new URL(v.url).hostname !== "billing.stripe.com")
        throw new Error("Destino de facturación inválido.");
      location.assign(v.url);
    } catch (e) {
      setError(e.message);
      setBusy("");
    }
  }
  const requestedPlan = catalog?.plans.find(
    (plan) => plan.id === new URLSearchParams(location.search).get("plan") && plan.priceCents > 0,
  );
  return (
    <>
      <div className="page-title">
        <div>
          <h1>Tu plan, a tu ritmo</h1>
          <p>Suscripciones mensuales en USD. Sin cargos por excedentes.</p>
        </div>
        <Button variant="secondary" busy={busy === "sync"} disabled={!!busy} onClick={sync}>
          <RefreshCw size={16} />
          Sincronizar
        </Button>
      </div>
      <Notice error>{error || localError || catalogError}</Notice>
      <Notice>{message}</Notice>
      {requestedPlan && (
        <Notice>Elegiste {requestedPlan.name}. Revisa sus límites y confirma el plan para continuar en Stripe.</Notice>
      )}
      {usage && (
        <div className="billing-current">
          <div>
            <span>Plan actual</span>
            <h2>{usage.plan.name}</h2>
            <p>
              {usage.unlimited
                ? "Peticiones sin cuota mensual"
                : `${number(usage.used)} de ${number(usage.plan.monthlyLimit)} peticiones utilizadas`}
            </p>
          </div>
          <div>
            <span>Fin del período</span>
            <strong>{date(usage.periodEnd)}</strong>
            <small>
              Estado:{" "}
              {{
                none: "sin suscripción",
                active: "activa",
                past_due: "pago pendiente",
                canceled: "cancelada",
                trialing: "en prueba",
              }[usage.subscriptionStatus] || usage.subscriptionStatus}
            </small>
          </div>
          {usage.hasSubscription && (
            <Button onClick={portal} busy={busy === "portal"} disabled={!!busy}>
              Gestionar facturación
              <ArrowUpRight size={16} />
            </Button>
          )}
        </div>
      )}
      <div className="billing-plans">
        {catalog?.plans.map((p) => (
          <article
            key={p.id}
            className={usage?.plan.id === p.id ? "current-plan" : ""}
          >
            <span>
              {usage?.plan.id === p.id ? "Tu plan actual" : "Plan mensual"}
            </span>
            <h2>{p.name}</h2>
            <strong className="billing-price">
              {p.priceCents === 0 ? "$0" : `$${p.priceCents / 100}`}
              <small>USD / mes</small>
            </strong>
            <p>
              {p.monthlyLimit === 0
                ? "Peticiones ilimitadas"
                : `${number(p.monthlyLimit)} peticiones / mes`}
            </p>
            <ul>
              <li>
                <Check size={16} />
                {p.rpm} peticiones / minuto
              </li>
              <li>
                <Check size={16} />
                Extracción de fichas de producto
              </li>
              <li>
                <Check size={16} />
                Hasta 10 API keys activas
              </li>
            </ul>
            {usage?.plan.id === p.id ? (
              <Button disabled variant="secondary">
                Plan actual
              </Button>
            ) : p.id === "free" ? (
              <p className="muted">
                Para volver a Free, cancela tu suscripción desde Gestionar
                facturación.
              </p>
            ) : (
              <Button
                busy={busy === p.id}
                disabled={!!busy || !usage}
                onClick={() =>
                  usage?.hasSubscription &&
                  ["active", "trialing", "past_due"].includes(
                    usage.subscriptionStatus,
                  )
                    ? portal()
                    : checkout(p.id)
                }
              >
                {usage?.hasSubscription &&
                ["active", "trialing", "past_due"].includes(
                  usage.subscriptionStatus,
                )
                  ? "Cambiar en Stripe"
                  : `Elegir ${p.name}`}
                <ArrowUpRight size={16} />
              </Button>
            )}
          </article>
        ))}
      </div>
      <div className="billing-terms">
        <h2>Cómo contamos las peticiones</h2>
        <p>
          Cada petición autenticada a búsqueda, resolución o consulta inversa
          consume una unidad, aunque utilice caché o devuelva un error. Las
          visitas al panel, la documentación y los rechazos por autenticación,
          permisos o cuota no consumen unidades.
        </p>
        <p>
          El período de pago sigue el ciclo de Stripe; Free se renueva cada mes
          calendario en UTC. Scale elimina la cuota mensual, manteniendo su
          límite por minuto. La cancelación se gestiona en Stripe y se hace
          efectiva al finalizar el período pagado.
        </p>
      </div>
    </>
  );
}
