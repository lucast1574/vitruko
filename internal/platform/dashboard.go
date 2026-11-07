package platform

import (
	"net/http"
	"strings"
	"time"
)

type Plan struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	MonthlyLimit  int64  `json:"monthlyLimit"`
	RPM           int    `json:"rpm"`
	PriceCents    int    `json:"priceCents"`
	Visible       bool   `json:"visible"`
	StripePriceID string `json:"-"`
}

func (p *Platform) plan(id string) (v Plan, err error) {
	err = p.db.QueryRow("SELECT id,name,monthly_limit,rpm,price_cents,visible,stripe_price_id FROM plans WHERE id=$1", id).Scan(&v.ID, &v.Name, &v.MonthlyLimit, &v.RPM, &v.PriceCents, &v.Visible, &v.StripePriceID)
	return
}
func (p *Platform) plans(w http.ResponseWriter, r *http.Request) {
	rows, err := p.db.QueryContext(r.Context(), "SELECT id,name,monthly_limit,rpm,price_cents FROM plans WHERE visible ORDER BY price_cents")
	if err != nil {
		problem(w, 503, "storage", "No se pudieron cargar los planes.")
		return
	}
	defer rows.Close()
	out := []Plan{}
	for rows.Next() {
		var v Plan
		if rows.Scan(&v.ID, &v.Name, &v.MonthlyLimit, &v.RPM, &v.PriceCents) != nil {
			problem(w, 503, "storage", "Intenta más tarde.")
			return
		}
		v.Visible = true
		out = append(out, v)
	}
	reply(w, 200, map[string]any{"plans": out, "currency": "USD", "billing": "monthly", "unit": "Cada petición autenticada admitida consume una unidad, incluso desde caché o si devuelve un error. Los rechazos por clave, permisos, cuota o velocidad no consumen unidades."})
}
func (p *Platform) usage(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	start, end := period(u, time.Now().UTC())
	plan, err := p.plan(u.PlanID)
	if err != nil {
		problem(w, 503, "storage", "Intenta más tarde.")
		return
	}
	var used int64
	_ = p.db.QueryRowContext(r.Context(), "SELECT used FROM usage_buckets WHERE user_id=$1 AND period_start=$2", u.ID, start).Scan(&used)
	rows, err := p.db.QueryContext(r.Context(), `SELECT day::date::text,count(e.id) FILTER(WHERE e.status<400),count(e.id) FILTER(WHERE NOT e.status<400) FROM generate_series(date_trunc('day',now())-interval '29 days',date_trunc('day',now()),interval '1 day') day LEFT JOIN usage_events e ON e.user_id=$1 AND e.created_at>=day AND e.created_at<day+interval '1 day' GROUP BY day ORDER BY day`, u.ID)
	if err != nil {
		problem(w, 503, "storage", "No se pudo cargar el consumo.")
		return
	}
	daily := []map[string]any{}
	for rows.Next() {
		var day string
		var success, failed int64
		if rows.Scan(&day, &success, &failed) != nil {
			rows.Close()
			problem(w, 503, "storage", "Intenta más tarde.")
			return
		}
		daily = append(daily, map[string]any{"day": day, "success": success, "failed": failed})
	}
	rows.Close()
	rows, err = p.db.QueryContext(r.Context(), "SELECT endpoint,count(*),count(*) FILTER(WHERE charged),COALESCE(round(avg(latency_ms)),0) FROM usage_events WHERE user_id=$1 AND created_at>=$2 GROUP BY endpoint ORDER BY count(*) DESC", u.ID, start)
	if err != nil {
		problem(w, 503, "storage", "Intenta más tarde.")
		return
	}
	endpoints := []map[string]any{}
	for rows.Next() {
		var endpoint string
		var total, charged, latency int64
		if rows.Scan(&endpoint, &total, &charged, &latency) != nil {
			continue
		}
		endpoints = append(endpoints, map[string]any{"endpoint": endpoint, "total": total, "charged": charged, "latencyMs": latency})
	}
	rows.Close()
	rows, err = p.db.QueryContext(r.Context(), "SELECT e.endpoint,e.status,e.charged,e.latency_ms,e.created_at,k.name FROM usage_events e JOIN api_keys k ON k.id=e.key_id WHERE e.user_id=$1 ORDER BY e.id DESC LIMIT 20", u.ID)
	if err != nil {
		problem(w, 503, "storage", "Intenta más tarde.")
		return
	}
	defer rows.Close()
	recent := []map[string]any{}
	for rows.Next() {
		var endpoint, key string
		var status int
		var charged bool
		var latency int64
		var at time.Time
		if rows.Scan(&endpoint, &status, &charged, &latency, &at, &key) != nil {
			continue
		}
		recent = append(recent, map[string]any{"endpoint": endpoint, "status": status, "charged": charged, "latencyMs": latency, "createdAt": at, "keyName": key})
	}
	reply(w, 200, map[string]any{"plan": plan, "used": used, "remaining": max(int64(0), plan.MonthlyLimit-used), "unlimited": plan.MonthlyLimit == 0, "periodStart": start, "periodEnd": end, "daily": daily, "endpoints": endpoints, "recent": recent, "subscriptionStatus": u.SubscriptionStatus, "hasSubscription": u.SubscriptionID.Valid})
}
func (p *Platform) adminUsers(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	rows, err := p.db.QueryContext(r.Context(), "SELECT "+userColumns+" FROM users WHERE email ILIKE $1 OR name ILIKE $1 ORDER BY created_at DESC LIMIT 200", "%"+q+"%")
	if err != nil {
		problem(w, 503, "storage", "No se pudieron cargar los usuarios.")
		return
	}
	defer rows.Close()
	users := []User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			problem(w, 503, "storage", "Intenta más tarde.")
			return
		}
		users = append(users, u)
	}
	reply(w, 200, map[string]any{"users": users})
}
func (p *Platform) adminUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Disabled *bool
		PlanID   *string
		Role     *string
	}
	if decode(w, r, &in) != nil {
		problem(w, 400, "invalid", "Datos inválidos.")
		return
	}
	tx, err := p.db.BeginTx(r.Context(), nil)
	if err != nil {
		problem(w, 503, "storage", "Intenta más tarde.")
		return
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(r.Context(), "SELECT pg_advisory_xact_lock(178304623)")
	if err != nil {
		problem(w, 503, "storage", "Intenta más tarde.")
		return
	}
	u, err := scanUser(tx.QueryRowContext(r.Context(), "SELECT "+userColumns+" FROM users WHERE id=$1 FOR UPDATE", r.PathValue("id")))
	if err != nil {
		problem(w, 404, "not_found", "Usuario no encontrado.")
		return
	}
	if in.Disabled != nil {
		u.Disabled = *in.Disabled
	}
	if in.Role != nil {
		if *in.Role != "admin" && *in.Role != "user" {
			problem(w, 400, "role", "Rol inválido.")
			return
		}
		u.Role = *in.Role
	}
	if u.ID == current(r).ID && (u.Disabled || u.Role != "admin") {
		problem(w, 400, "self", "No puedes suspender o quitar permisos a tu propia cuenta.")
		return
	}
	if in.PlanID != nil {
		if u.SubscriptionID.Valid && (u.SubscriptionStatus == "active" || u.SubscriptionStatus == "trialing" || u.SubscriptionStatus == "past_due") {
			problem(w, 409, "billing", "Cambia este plan desde Stripe para mantener la suscripción sincronizada.")
			return
		}
		var exists bool
		if err = tx.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM plans WHERE id=$1)", *in.PlanID).Scan(&exists); err != nil || !exists {
			problem(w, 400, "plan", "Plan inválido.")
			return
		}
		u.PlanID = *in.PlanID
	}
	var admins int
	_ = tx.QueryRowContext(r.Context(), "SELECT count(*) FROM users WHERE role='admin' AND NOT disabled AND id<>$1", u.ID).Scan(&admins)
	if admins == 0 && (u.Disabled || u.Role != "admin") {
		var oldRole string
		_ = tx.QueryRowContext(r.Context(), "SELECT role FROM users WHERE id=$1", u.ID).Scan(&oldRole)
		if oldRole == "admin" {
			problem(w, 400, "last_admin", "Debe existir al menos una cuenta administradora activa.")
			return
		}
	}
	_, err = tx.ExecContext(r.Context(), "UPDATE users SET disabled=$1,role=$2,plan_id=$3 WHERE id=$4", u.Disabled, u.Role, u.PlanID, u.ID)
	if err == nil {
		_, err = tx.ExecContext(r.Context(), "INSERT INTO audit_events(actor_id,action,target_id) VALUES($1,'user.update',$2)", current(r).ID, u.ID)
	}
	if err != nil || tx.Commit() != nil {
		problem(w, 503, "storage", "No se pudo actualizar el usuario.")
		return
	}
	reply(w, 200, u)
}
func (p *Platform) adminPlan(w http.ResponseWriter, r *http.Request) {
	var in struct {
		MonthlyLimit int64
		RPM          int
	}
	if decode(w, r, &in) != nil || in.MonthlyLimit < 0 || in.MonthlyLimit > 100000000 || in.RPM < 1 || in.RPM > 10000 {
		problem(w, 400, "invalid", "Cuota o límite por minuto inválidos.")
		return
	}
	result, err := p.db.ExecContext(r.Context(), "UPDATE plans SET monthly_limit=$1,rpm=$2 WHERE id=$3", in.MonthlyLimit, in.RPM, r.PathValue("id"))
	if err != nil {
		problem(w, 503, "storage", "No se pudo actualizar el plan.")
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		problem(w, 404, "not_found", "Plan no encontrado.")
		return
	}
	_, _ = p.db.ExecContext(r.Context(), "INSERT INTO audit_events(actor_id,action,target_id) VALUES($1,'plan.update',$2)", current(r).ID, r.PathValue("id"))
	reply(w, 200, map[string]bool{"ok": true})
}
func (p *Platform) adminOverview(w http.ResponseWriter, r *http.Request) {
	var users, keys, subscriptions, requests, errors int64
	_ = p.db.QueryRowContext(r.Context(), "SELECT count(*),count(*) FILTER(WHERE subscription_status='active') FROM users").Scan(&users, &subscriptions)
	_ = p.db.QueryRowContext(r.Context(), "SELECT count(*) FROM api_keys WHERE revoked_at IS NULL").Scan(&keys)
	_ = p.db.QueryRowContext(r.Context(), "SELECT count(*),count(*) FILTER(WHERE status>=400) FROM usage_events WHERE created_at>=date_trunc('month',now())").Scan(&requests, &errors)
	reply(w, 200, map[string]any{"users": users, "activeKeys": keys, "activeSubscriptions": subscriptions, "requestsThisMonth": requests, "failedRequests": errors})
}
