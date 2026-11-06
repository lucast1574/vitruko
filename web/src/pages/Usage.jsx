import React from "react";
import { ArrowUpRight, KeyRound, ArrowRight } from "lucide-react";
import { useData } from "../useData";
import { Notice, Empty } from "../ui";
import { number, date } from "../client";
function Chart({ daily }) {
  const max = Math.max(1, ...daily.map((d) => d.success + d.failed));
  const total = daily.reduce((n, d) => n + d.success + d.failed, 0);
  return (
    <div className="usage-chart">
      <div className="chart-heading">
        <h2>Peticiones de los últimos 30 días</h2>
        <span>{number(total)} en total</span>
      </div>
      {total === 0 ? (
        <Empty title="Tu primera petición empieza aquí">
          Crea una API key e integra una operación. El consumo real aparecerá en
          este gráfico.
        </Empty>
      ) : (
        <>
          <svg
            viewBox="0 0 900 190"
            role="img"
            aria-label={`${total} peticiones durante los últimos 30 días`}
            preserveAspectRatio="none"
          >
            <line x1="0" x2="900" y1="170" y2="170" stroke="var(--border)" />
            {daily.map((d, i) => {
              const h = ((d.success + d.failed) / max) * 145;
              return (
                <rect
                  key={d.day}
                  x={i * 30 + 5}
                  y={170 - h}
                  width="19"
                  height={Math.max(h, 1)}
                  rx="3"
                  fill="var(--accent)"
                >
                  <title>
                    {d.day}: {d.success + d.failed} peticiones
                  </title>
                </rect>
              );
            })}
          </svg>
          <div className="chart-labels">
            <span>{daily[0]?.day}</span>
            <span>Máximo diario: {number(max)}</span>
            <span>{daily.at(-1)?.day}</span>
          </div>
        </>
      )}
    </div>
  );
}
function Metrics({ data }) {
  return (
    <div className="metrics-strip">
      <div>
        <span>Peticiones del período</span>
        <strong>{number(data.used)}</strong>
        <small>Desde el {date(data.periodStart)}</small>
      </div>
      <div>
        <span>
          {data.unlimited ? "Cuota mensual" : "Peticiones disponibles"}
        </span>
        <strong>
          {data.unlimited ? "Sin límite" : number(data.remaining)}
        </strong>
        <small>
          {data.unlimited
            ? "Sin cuota mensual"
            : `De ${number(data.plan.monthlyLimit)} incluidas`}
        </small>
      </div>
      <div>
        <span>Plan actual</span>
        <strong className="plan-name">{data.plan.name}</strong>
        <small>{data.plan.rpm} peticiones / minuto</small>
      </div>
    </div>
  );
}
function Recent({ recent }) {
  return (
    <div className="data-section">
      <div className="section-heading">
        <h2>Actividad reciente</h2>
        <span>Últimas 20 peticiones</span>
      </div>
      {recent.length ? (
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Operación</th>
                <th>API key</th>
                <th>Estado</th>
                <th className="numeric">Tiempo</th>
                <th className="numeric">Fecha</th>
              </tr>
            </thead>
            <tbody>
              {recent.map((r, i) => (
                <tr key={i}>
                  <td>
                    <code>/v1/{r.endpoint}</code>
                  </td>
                  <td>{r.keyName}</td>
                  <td>
                    <span
                      className={`status-pill ${r.status >= 400 ? "bad" : "good"}`}
                    >
                      {r.status}
                    </span>
                  </td>
                  <td className="numeric">{number(r.latencyMs)} ms</td>
                  <td className="numeric">
                    {new Date(r.createdAt).toLocaleString("es-PE", {
                      month: "short",
                      day: "numeric",
                      hour: "2-digit",
                      minute: "2-digit",
                    })}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <Empty title="Aún no hay peticiones">
          No contamos las visitas al panel ni las claves rechazadas. Aquí
          aparecerán tus peticiones autenticadas a la API.
        </Empty>
      )}
    </div>
  );
}
export function Overview({ user }) {
  const { data, error } = useData("/console/usage");
  return (
    <>
      <div className="page-title">
        <div>
          <h1>Tu consumo, al detalle</h1>
          <p>Todo lo que tu aplicación está conectando.</p>
        </div>
        <a className="button" href="/app/keys">
          <KeyRound size={16} />
          Gestionar API keys
        </a>
      </div>
      <Notice error>{error}</Notice>
      {data ? (
        <>
          <Metrics data={data} />
          <div className="overview-grid">
            <Chart daily={data.daily} />
            <aside className="integration-aside">
              <h2>Tu siguiente conexión</h2>
              <p>
                Envía tu clave en cada petición. Tu cuenta reúne el consumo de
                todos tus proyectos.
              </p>
              <pre>{`X-API-Key: vt_live_…`}</pre>
              <a href="/app/docs">
                Ver guía de integración
                <ArrowUpRight size={16} />
              </a>
              <div className="period-note">
                <span>Próximo período</span>
                <strong>{date(data.periodEnd)}</strong>
                <p>
                  Una petición autenticada equivale a una unidad, también si
                  responde desde caché o devuelve un error.
                </p>
              </div>
            </aside>
          </div>
          <Recent recent={data.recent} />
        </>
      ) : (
        <p className="loading-text">Cargando el consumo real de tu cuenta…</p>
      )}
    </>
  );
}
export function Usage() {
  const { data, error } = useData("/console/usage");
  return (
    <>
      <div className="page-title">
        <div>
          <h1>Consumo por petición</h1>
          <p>Todas tus claves, un solo período de uso.</p>
        </div>
        <a className="text-link" href="/app/billing">
          Ver mi plan
          <ArrowUpRight size={17} />
        </a>
      </div>
      <Notice error>{error}</Notice>
      {data ? (
        <>
          <Metrics data={data} />
          <Chart daily={data.daily} />
          <div className="endpoint-breakdown">
            <h2>Por operación</h2>
            {data.endpoints.length ? (
              data.endpoints.map((e) => (
                <div className="endpoint-row" key={e.endpoint}>
                  <code>/v1/{e.endpoint}</code>
                  <span>{number(e.total)} peticiones</span>
                  <span>{number(e.latencyMs)} ms de media</span>
                </div>
              ))
            ) : (
              <p className="muted">
                El desglose aparecerá con tu primera petición.
              </p>
            )}
          </div>
          <Recent recent={data.recent} />
        </>
      ) : (
        <p>Cargando consumo…</p>
      )}
    </>
  );
}
