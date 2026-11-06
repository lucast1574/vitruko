package platform

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func validSignature(header string, body []byte, secret string, now time.Time) bool {
	var timestamp int64
	signatures := []string{}
	for _, part := range strings.Split(header, ",") {
		k, v, _ := strings.Cut(part, "=")
		if k == "t" {
			timestamp, _ = strconv.ParseInt(v, 10, 64)
		}
		if k == "v1" {
			signatures = append(signatures, v)
		}
	}
	age := now.Unix() - timestamp
	if age > 300 || age < -300 || timestamp == 0 || secret == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10) + "."))
	mac.Write(body)
	expected := mac.Sum(nil)
	for _, signature := range signatures {
		decoded, err := hex.DecodeString(signature)
		if err == nil && hmac.Equal(decoded, expected) {
			return true
		}
	}
	return false
}
func (p *Platform) webhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil || !validSignature(r.Header.Get("Stripe-Signature"), body, p.config.StripeWebhookSecret, time.Now()) {
		problem(w, 400, "signature", "Firma de webhook inválida.")
		return
	}
	var event struct {
		ID, Type string
		Data     struct{ Object map[string]any }
	}
	if json.Unmarshal(body, &event) != nil || event.ID == "" {
		problem(w, 400, "event", "Evento inválido.")
		return
	}
	if event.Type != "checkout.session.completed" && !strings.HasPrefix(event.Type, "customer.subscription.") && event.Type != "invoice.paid" && event.Type != "invoice.payment_failed" {
		reply(w, 200, map[string]bool{"received": true})
		return
	}
	customer := text(event.Data.Object["customer"])
	var userID string
	err = p.db.QueryRowContext(r.Context(), "SELECT id FROM users WHERE customer_id=$1", customer).Scan(&userID)
	if err == sql.ErrNoRows {
		reply(w, 200, map[string]bool{"received": true})
		return
	}
	if err != nil {
		problem(w, 503, "storage", "Reintenta el evento.")
		return
	}
	if err = p.reconcile(r.Context(), userID, event.ID); err != nil {
		problem(w, 503, "billing", "No se pudo procesar el evento. Reintenta.")
		return
	}
	reply(w, 200, map[string]bool{"received": true})
}

func (p *Platform) reconcile(ctx context.Context, userID, eventID string) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Lock before fetching Stripe so concurrent/out-of-order events cannot write an older snapshot last.
	u, err := scanUser(tx.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE id=$1 FOR UPDATE", userID))
	if err != nil {
		return err
	}
	if !u.CustomerID.Valid {
		return nil
	}
	if eventID != "" {
		var exists bool
		if err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM stripe_events WHERE id=$1)", eventID).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return nil
		}
	}
	result, err := p.stripe(ctx, "GET", "/subscriptions?"+url.Values{"customer": {u.CustomerID.String}, "status": {"all"}, "limit": {"30"}}.Encode(), nil, "")
	if err != nil {
		return err
	}
	plans, err := tx.QueryContext(ctx, "SELECT id,stripe_price_id FROM plans WHERE stripe_price_id<>''")
	if err != nil {
		return err
	}
	prices := map[string]string{}
	for plans.Next() {
		var id, price string
		if plans.Scan(&id, &price) != nil {
			plans.Close()
			return errors.New("invalid plan")
		}
		prices[price] = id
	}
	plans.Close()
	var chosen map[string]any
	chosenPlan := ""
	rank := 0
	for _, v := range array(result["data"]) {
		sub := object(v)
		if text(object(sub["metadata"])["app"]) != "vitruko" {
			continue
		}
		items := array(object(sub["items"])["data"])
		if len(items) != 1 {
			continue
		}
		planID := prices[text(object(items[0])["price"])]
		if planID == "" {
			continue
		}
		status := text(sub["status"])
		r := 1
		if status == "past_due" {
			r = 2
		}
		if status == "active" || status == "trialing" {
			r = 3
		}
		if r > rank {
			rank = r
			chosen = sub
			chosenPlan = planID
		}
	}
	if chosen != nil {
		status := text(chosen["status"])
		planID := chosenPlan
		if status != "active" && status != "trialing" {
			planID = "free"
		}
		item := object(array(object(chosen["items"])["data"])[0])
		start, end := number(item["current_period_start"]), number(item["current_period_end"])
		if start == 0 {
			start = number(chosen["current_period_start"])
			end = number(chosen["current_period_end"])
		}
		if (status == "active" || status == "trialing") && (start == 0 || end <= start) {
			return errors.New("missing billing period")
		}
		var a, b any
		if start > 0 {
			a = time.Unix(start, 0)
		}
		if end > start {
			b = time.Unix(end, 0)
		}
		_, err = tx.ExecContext(ctx, "UPDATE users SET plan_id=$1,subscription_id=$2,subscription_status=$3,billing_start=$4,billing_end=$5 WHERE id=$6", planID, text(chosen["id"]), status, a, b, userID)
		if err == nil && (status == "active" || status == "trialing" || status == "canceled") {
			_, err = tx.ExecContext(ctx, "DELETE FROM checkout_attempts WHERE user_id=$1", userID)
		}
	} else if u.SubscriptionID.Valid {
		_, err = tx.ExecContext(ctx, "UPDATE users SET plan_id='free',subscription_id=NULL,subscription_status='none',billing_start=NULL,billing_end=NULL WHERE id=$1", userID)
	}
	if err != nil {
		return err
	}
	if eventID != "" {
		_, err = tx.ExecContext(ctx, "INSERT INTO stripe_events(id) VALUES($1) ON CONFLICT DO NOTHING", eventID)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}
