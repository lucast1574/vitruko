package platform

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func apiEndpoint(path string) string {
	if path == "/v1/extract" {
		return "extract"
	}
	return ""
}

type allowance struct {
	UserID, KeyID      string
	Start, End, Minute time.Time
	Limit, Remaining   int64
	RPM                int
}
type quotaError struct {
	Code  string
	Reset time.Time
}

func (e quotaError) Error() string { return e.Code }
func period(u User, now time.Time) (time.Time, time.Time) {
	if u.PlanID != "free" && u.BillingStart.Valid && u.BillingEnd.Valid && now.Before(u.BillingEnd.Time) && !now.Before(u.BillingStart.Time) {
		return u.BillingStart.Time, u.BillingEnd.Time
	}
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return start, start.AddDate(0, 1, 0)
}
func (p *Platform) reserve(ctx context.Context, key, endpoint string) (a allowance, err error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return a, err
	}
	defer tx.Rollback()
	var scopes []string
	err = tx.QueryRowContext(ctx, "SELECT id,user_id,scopes FROM api_keys WHERE token_hash=$1 AND revoked_at IS NULL", digest(key)).Scan(&a.KeyID, &a.UserID, &scopes)
	if err == sql.ErrNoRows {
		return a, quotaError{Code: "invalid_key"}
	}
	if err != nil {
		return a, err
	}
	allowed := false
	for _, s := range scopes {
		if s == endpoint {
			allowed = true
		}
	}
	if !allowed {
		return a, quotaError{Code: "scope_denied"}
	}
	u, err := scanUser(tx.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE id=$1 FOR UPDATE", a.UserID))
	if err != nil {
		return a, err
	}
	if u.Disabled {
		return a, quotaError{Code: "account_disabled"}
	}
	if err = tx.QueryRowContext(ctx, "SELECT monthly_limit,rpm FROM plans WHERE id=$1", u.PlanID).Scan(&a.Limit, &a.RPM); err != nil {
		return a, err
	}
	now := time.Now().UTC()
	if u.PlanID != "free" && u.PlanID != "internal" && u.SubscriptionID.Valid && u.BillingEnd.Valid && !now.Before(u.BillingEnd.Time) {
		return a, quotaError{Code: "billing_pending"}
	}
	a.Start, a.End = period(u, now)
	a.Minute = now.Truncate(time.Minute)
	_, err = tx.ExecContext(ctx, "INSERT INTO usage_buckets(user_id,period_start) VALUES($1,$2) ON CONFLICT DO NOTHING", a.UserID, a.Start)
	if err != nil {
		return a, err
	}
	var used int64
	err = tx.QueryRowContext(ctx, "SELECT used FROM usage_buckets WHERE user_id=$1 AND period_start=$2", a.UserID, a.Start).Scan(&used)
	if err != nil {
		return a, err
	}
	if a.Limit > 0 && used >= a.Limit {
		return a, quotaError{Code: "quota_exceeded", Reset: a.End}
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO rate_buckets(user_id,minute) VALUES($1,$2) ON CONFLICT DO NOTHING", a.UserID, a.Minute)
	if err != nil {
		return a, err
	}
	var minuteUsed int
	err = tx.QueryRowContext(ctx, "SELECT used FROM rate_buckets WHERE user_id=$1 AND minute=$2", a.UserID, a.Minute).Scan(&minuteUsed)
	if err != nil {
		return a, err
	}
	if minuteUsed >= a.RPM {
		return a, quotaError{Code: "rate_limited", Reset: a.Minute.Add(time.Minute)}
	}
	_, err = tx.ExecContext(ctx, "UPDATE usage_buckets SET used=used+1 WHERE user_id=$1 AND period_start=$2", a.UserID, a.Start)
	if err == nil {
		_, err = tx.ExecContext(ctx, "UPDATE rate_buckets SET used=used+1 WHERE user_id=$1 AND minute=$2", a.UserID, a.Minute)
	}
	if err != nil {
		return a, err
	}
	a.Remaining = a.Limit - used - 1
	return a, tx.Commit()
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
		w.ResponseWriter.WriteHeader(code)
	}
}
func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	return w.ResponseWriter.Write(b)
}
func (p *Platform) finish(a allowance, endpoint string, status int, elapsed time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	charged := true
	if err == nil {
		_, err = tx.ExecContext(ctx, "INSERT INTO usage_events(user_id,key_id,endpoint,status,charged,latency_ms) VALUES($1,$2,$3,$4,$5,$6)", a.UserID, a.KeyID, endpoint, status, charged, elapsed.Milliseconds())
	}
	if err == nil {
		_, err = tx.ExecContext(ctx, "UPDATE api_keys SET last_used_at=now() WHERE id=$1", a.KeyID)
	}
	if err == nil {
		_ = tx.Commit()
	}
}
func (p *Platform) protectAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
	w.Header().Set("Access-Control-Expose-Headers", "X-Cache, X-Quota-Limit, X-Quota-Remaining, X-Quota-Reset, Retry-After")
	if r.Method == "OPTIONS" {
		w.WriteHeader(204)
		return
	}
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST, OPTIONS")
		problem(w, 405, "method", "Usa POST para extraer una ficha de producto.")
		return
	}
	raw := r.Header.Get("X-API-Key")
	if raw == "" {
		raw = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	if raw == "" {
		problem(w, 401, "api_key_required", "Envía tu API key en X-API-Key o Authorization: Bearer.")
		return
	}
	endpoint := apiEndpoint(r.URL.Path)
	a, err := p.reserve(r.Context(), raw, endpoint)
	if err != nil {
		var q quotaError
		if errors.As(err, &q) {
			code := 401
			if q.Code == "billing_pending" {
				code = 503
			}
			if q.Code == "scope_denied" || q.Code == "account_disabled" {
				code = 403
			}
			if q.Code == "quota_exceeded" || q.Code == "rate_limited" {
				code = 429
				w.Header().Set("Retry-After", strconv.Itoa(max(1, int(time.Until(q.Reset).Seconds()))))
			}
			problem(w, code, q.Code, map[string]string{"billing_pending": "La renovación está pendiente de sincronización. Sincroniza tu facturación o reintenta en unos minutos.", "invalid_key": "API key inválida o revocada.", "scope_denied": "La clave no tiene permiso para esta operación.", "account_disabled": "La cuenta está suspendida.", "quota_exceeded": "Agotaste la cuota del período. Cambia de plan para continuar.", "rate_limited": "Superaste el límite por minuto. Reintenta después."}[q.Code])
		} else {
			problem(w, 503, "quota_unavailable", "No se pudo comprobar la cuota. Reintenta.")
		}
		return
	}
	w.Header().Set("X-Quota-Limit", strconv.FormatInt(a.Limit, 10))
	w.Header().Set("X-Quota-Remaining", strconv.FormatInt(a.Remaining, 10))
	if a.Limit == 0 {
		w.Header().Set("X-Quota-Limit", "unlimited")
		w.Header().Set("X-Quota-Remaining", "unlimited")
	}
	w.Header().Set("X-Quota-Reset", a.End.Format(time.RFC3339))
	start := time.Now()
	writer := &statusWriter{ResponseWriter: w}
	p.geo.ServeHTTP(writer, r)
	p.finish(a, endpoint, writer.status, time.Since(start))
}
