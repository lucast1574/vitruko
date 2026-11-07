import React, { useState } from "react";
import { Search, Settings2, Users, Save } from "lucide-react";
import { useData } from "../useData";
import { api, number, date } from "../client";
import { Button, Notice, Modal, Select } from "../ui";
export default function Admin() {
  const [query, setQuery] = useState(""),
    [filter, setFilter] = useState(""),
    [target, setTarget] = useState(null),
    [planTarget, setPlanTarget] = useState(null),
    [error, setError] = useState(""),
    [success, setSuccess] = useState(""),
    [busy, setBusy] = useState(false);
  const { data: overview } = useData("/console/admin/overview"),
    { data: users, reload } = useData(
      "/console/admin/users?q=" + encodeURIComponent(filter),
    ),
    { data: plans, reload: reloadPlans } = useData("/console/plans");
  async function save(e) {
    e.preventDefault();
    setBusy(true);
    setError("");
    setSuccess("");
    try {
      await api("/console/admin/users/" + target.id, "PATCH", {
        disabled: target.disabled,
        role: target.role,
        ...(target.planId !== target.originalPlanId
          ? { planId: target.planId }
          : {}),
      });
      setTarget(null);
      await reload();
      setSuccess("Cambios de la cuenta guardados.");
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  }
  async function savePlan(e) {
    e.preventDefault();
    setBusy(true);
    setError("");
    setSuccess("");
    try {
      await api("/console/admin/plans/" + planTarget.id, "PATCH", {
        monthlyLimit: Number(planTarget.monthlyLimit),
        rpm: Number(planTarget.rpm),
      });
      setPlanTarget(null);
      await reloadPlans();
      setSuccess("Límites del plan guardados.");
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <div className="page-title">
        <div>
          <h1>Administración de Vitruko</h1>
          <p>Usuarios, acceso y límites del servicio.</p>
        </div>
        <span className="admin-marker">
          <Users size={16} />
          Administrador
        </span>
      </div>
      <Notice error>{error}</Notice>
      <Notice>{success}</Notice>
      {overview && (
        <div className="admin-metrics">
          {[
            ["Usuarios", overview.users],
            ["API keys activas", overview.activeKeys],
            ["Suscripciones activas", overview.activeSubscriptions],
            ["Peticiones este mes", overview.requestsThisMonth],
          ].map(([label, value]) => (
            <div key={label}>
              <span>{label}</span>
              <strong>{number(value)}</strong>
            </div>
          ))}
        </div>
      )}
      <section className="admin-users">
        <div className="section-heading">
          <h2>Cuentas</h2>
          <form
            className="search-form"
            onSubmit={(e) => {
              e.preventDefault();
              setFilter(query);
            }}
          >
            <input
              type="search"
              aria-label="Buscar usuarios"
              placeholder="Nombre o correo"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
            <Button variant="secondary">
              <Search size={16} />
              Buscar
            </Button>
          </form>
        </div>
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Usuario</th>
                <th>Plan</th>
                <th>Rol</th>
                <th>Estado</th>
                <th>Creado</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {users?.users.map((u) => (
                <tr key={u.id}>
                  <td>
                    <strong>{u.name}</strong>
                    <small>{u.email}</small>
                  </td>
                  <td>{u.planId}</td>
                  <td>{u.role === "admin" ? "Administrador" : "Usuario"}</td>
                  <td>
                    <span
                      className={`status-pill ${u.disabled ? "bad" : "good"}`}
                    >
                      {u.disabled ? "Suspendido" : "Activo"}
                    </span>
                  </td>
                  <td>{date(u.createdAt)}</td>
                  <td>
                    <button
                      className="icon-button"
                      aria-label={`Editar ${u.email}`}
                      onClick={() => {
                        setError("");
                        setSuccess("");
                        setTarget({ ...u, originalPlanId: u.planId });
                      }}
                    >
                      <Settings2 size={17} />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
      <section className="admin-plans">
        <div className="section-heading">
          <h2>Límites de los planes</h2>
          <span>Los precios se gestionan en Stripe.</span>
        </div>
        {plans?.plans.map((p) => (
          <div className="admin-plan-row" key={p.id}>
            <strong>{p.name}</strong>
            <span>
              {p.monthlyLimit === 0
                ? "Sin cuota mensual"
                : `${number(p.monthlyLimit)} peticiones / mes`}
            </span>
            <span>{p.rpm} / minuto</span>
            <Button
              variant="secondary"
              onClick={() => {
                setError("");
                setSuccess("");
                setPlanTarget({ ...p });
              }}
            >
              Editar límites
            </Button>
          </div>
        ))}
      </section>
      {target && (
        <Modal
          title={`Cuenta de ${target.name}`}
          onClose={() => setTarget(null)}
        >
          <form onSubmit={save}>
            <p>{target.email}</p>
            <label>
              Rol
              <Select
                label="Rol del usuario"
                value={target.role}
                onChange={(v) => setTarget({ ...target, role: v })}
                options={[
                  { value: "user", label: "Usuario" },
                  { value: "admin", label: "Administrador" },
                ]}
              />
            </label>
            {["active", "trialing", "past_due"].includes(target.subscriptionStatus) ? (
              <Notice>
                El plan está vinculado a Stripe. Debe cambiarse desde la
                facturación de la suscripción.
              </Notice>
            ) : (
              <label>
                Plan
                <Select
                  label="Plan del usuario"
                  value={target.planId}
                  onChange={(v) => setTarget({ ...target, planId: v })}
                  options={[
                    ...(plans?.plans || []).map((p) => ({
                      value: p.id,
                      label: p.name,
                    })),
                    { value: "internal", label: "Interno" },
                  ]}
                />
              </label>
            )}
            <label className="check-row">
              <input
                type="checkbox"
                checked={target.disabled}
                onChange={(e) =>
                  setTarget({ ...target, disabled: e.target.checked })
                }
              />
              Suspender acceso a la cuenta y sus claves
            </label>
            <Notice error>{error}</Notice>
            <Button busy={busy}>
              <Save size={16} />
              Guardar cambios
            </Button>
          </form>
        </Modal>
      )}
      {planTarget && (
        <Modal
          title={`Límites de ${planTarget.name}`}
          onClose={() => setPlanTarget(null)}
        >
          <form onSubmit={savePlan}>
            <label>
              Cuota mensual de peticiones
              <input
                type="number"
                min="0"
                max="100000000"
                value={planTarget.monthlyLimit}
                onChange={(e) =>
                  setPlanTarget({ ...planTarget, monthlyLimit: e.target.value })
                }
                required
              />
              <small>0 significa sin cuota mensual.</small>
            </label>
            <label>
              Peticiones por minuto
              <input
                type="number"
                min="1"
                max="10000"
                value={planTarget.rpm}
                onChange={(e) =>
                  setPlanTarget({ ...planTarget, rpm: e.target.value })
                }
                required
              />
            </label>
            <p>
              El cambio se aplica a las cuentas de este plan. El consumo ya
              realizado se conserva.
            </p>
            <Notice error>{error}</Notice>
            <Button busy={busy}>Guardar límites</Button>
          </form>
        </Modal>
      )}
    </>
  );
}
