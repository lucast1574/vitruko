package platform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func (p *Platform) stripe(ctx context.Context, method, path string, form url.Values, idempotency string) (map[string]any, error) {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, "https://api.stripe.com/v1"+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.config.StripeKey)
	req.Header.Set("Stripe-Version", "2025-06-30.basil")
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if idempotency != "" {
		req.Header.Set("Idempotency-Key", idempotency)
	}
	res, err := p.stripeHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("stripe status %d", res.StatusCode)
	}
	var result map[string]any
	err = json.NewDecoder(io.LimitReader(res.Body, 2<<20)).Decode(&result)
	return result, err
}
func text(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if m, ok := v.(map[string]any); ok {
		s, _ := m["id"].(string)
		return s
	}
	return ""
}
func object(v any) map[string]any {
	m, _ := v.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}
func array(v any) []any  { a, _ := v.([]any); return a }
func number(v any) int64 { f, _ := v.(float64); return int64(f) }
func (p *Platform) customer(ctx context.Context, u User) (string, error) {
	if u.CustomerID.Valid {
		return u.CustomerID.String, nil
	}
	value, err := p.stripe(ctx, "POST", "/customers", url.Values{"email": {u.Email}, "name": {u.Name}, "metadata[app]": {"vitruko"}, "metadata[user_id]": {u.ID}}, "vitruko-customer-"+u.ID)
	if err != nil {
		return "", err
	}
	id := text(value["id"])
	if id == "" {
		return "", errors.New("missing customer")
	}
	_, err = p.db.ExecContext(ctx, "UPDATE users SET customer_id=$1 WHERE id=$2 AND customer_id IS NULL", id, u.ID)
	return id, err
}

func (p *Platform) checkout(w http.ResponseWriter, r *http.Request) {
	var in struct{ PlanID string }
	if decode(w, r, &in) != nil {
		problem(w, 400, "invalid", "Plan inválido.")
		return
	}
	plan, err := p.plan(in.PlanID)
	if err != nil || !plan.Visible || plan.StripePriceID == "" || plan.PriceCents == 0 {
		problem(w, 400, "plan", "Este plan no admite checkout.")
		return
	}
	u := current(r)
	if u.SubscriptionID.Valid && (u.SubscriptionStatus == "active" || u.SubscriptionStatus == "trialing" || u.SubscriptionStatus == "past_due") {
		problem(w, 409, "subscription_exists", "Ya tienes una suscripción. Usa Gestionar facturación para cambiarla.")
		return
	}
	customer, err := p.customer(r.Context(), u)
	if err != nil {
		problem(w, 502, "billing", "No se pudo iniciar Stripe. Reintenta.")
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if len(key) < 16 || len(key) > 100 {
		problem(w, 400, "idempotency", "Envía un Idempotency-Key de 16 a 100 caracteres.")
		return
	}
	// Reconcile against Stripe too: a delayed webhook must not permit a second subscription.
	if err = p.reconcile(r.Context(), u.ID, ""); err != nil {
		problem(w, 502, "billing", "No se pudo comprobar la suscripción actual. Reintenta.")
		return
	} else {
		fresh, _ := p.user(r.Context(), u.ID)
		if fresh.SubscriptionID.Valid && (fresh.SubscriptionStatus == "active" || fresh.SubscriptionStatus == "trialing" || fresh.SubscriptionStatus == "past_due") {
			problem(w, 409, "subscription_exists", "Ya tienes una suscripción. Gestiona tu facturación.")
			return
		}
	}
	attempt, existingURL, err := p.checkoutAttempt(r.Context(), u.ID, plan.ID)
	if err != nil {
		problem(w, 409, "checkout_pending", err.Error())
		return
	}
	if existingURL != "" {
		reply(w, 200, map[string]any{"url": existingURL})
		return
	}
	form := url.Values{"mode": {"subscription"}, "customer": {customer}, "line_items[0][price]": {plan.StripePriceID}, "line_items[0][quantity]": {"1"}, "client_reference_id": {u.ID}, "success_url": {p.config.AppURL + "/app/billing?checkout=success"}, "cancel_url": {p.config.AppURL + "/app/billing?checkout=cancelled"}, "subscription_data[metadata][app]": {"vitruko"}, "subscription_data[metadata][user_id]": {u.ID}, "metadata[app]": {"vitruko"}, "metadata[user_id]": {u.ID}, "allow_promotion_codes": {"false"}}
	value, err := p.stripe(r.Context(), "POST", "/checkout/sessions", form, "vitruko-"+attempt)
	if err != nil {
		problem(w, 502, "billing", "No se pudo abrir el checkout. Reintenta con el mismo intento.")
		return
	}
	if _, err = p.db.ExecContext(r.Context(), "UPDATE checkout_attempts SET session_id=$1 WHERE user_id=$2 AND attempt_id=$3", text(value["id"]), u.ID, attempt); err != nil {
		problem(w, 503, "checkout_pending", "El checkout fue creado. Reintenta para recuperar el mismo enlace.")
		return
	}
	reply(w, 200, map[string]any{"url": value["url"]})
}
func (p *Platform) portal(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !u.CustomerID.Valid {
		problem(w, 400, "billing", "Aún no tienes una cuenta de facturación.")
		return
	}
	form := url.Values{"customer": {u.CustomerID.String}, "return_url": {p.config.AppURL + "/app/billing"}}
	if p.config.StripePortalConfig != "" {
		form.Set("configuration", p.config.StripePortalConfig)
	}
	v, err := p.stripe(r.Context(), "POST", "/billing_portal/sessions", form, "")
	if err != nil {
		problem(w, 502, "billing", "No se pudo abrir la facturación.")
		return
	}
	reply(w, 200, map[string]any{"url": v["url"]})
}
func (p *Platform) syncBilling(w http.ResponseWriter, r *http.Request) {
	if err := p.reconcile(r.Context(), current(r).ID, ""); err != nil {
		problem(w, 502, "billing", "No se pudo sincronizar Stripe. Reintenta.")
		return
	}
	u, _ := p.user(r.Context(), current(r).ID)
	reply(w, 200, u)
}
