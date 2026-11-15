package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestReactEmailRenderingEscapesUserContent(t *testing.T) {
	p := &Platform{config: Config{MailFrom: "Vitruko <hola@vitruko.dev>"}}
	m, err := p.renderMail("test@example.invalid", "Bienvenida", mailContent{Title: "Una cuenta", Name: `<img src=x onerror=alert(1)>`, Body: "Prueba", Action: "Entrar", ActionURL: "https://app.vitruko.dev/app"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(m.HTML, "<img src=x") || !strings.Contains(m.HTML, "&lt;img") || !strings.Contains(m.Text, "Una cuenta") || strings.Contains(m.HTML, "{{.") {
		t.Fatal("invalid rendering or unsafe user content")
	}
	if !strings.Contains(m.HTML, "https://app.vitruko.dev/app") {
		t.Fatal("action missing")
	}
}
func TestNewUserNotificationsPersistAndDeduplicate(t *testing.T) {
	p := databasePlatform(t)
	admin, _ := signupTest(t, p)
	if _, err := p.db.Exec("UPDATE users SET role='admin' WHERE id=$1", admin.ID); err != nil {
		t.Fatal(err)
	}
	p.config.ResendKey = "re_test"
	p.config.MailFrom = "Vitruko <hola@vitruko.dev>"
	u, _ := signupTest(t, p)
	ctx := context.Background()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.newUserMail(ctx, tx, u.ID, u.Email, u.Name); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"welcome/" + u.ID, "admin-signup/" + u.ID + "/" + digest(admin.Email)} {
		var n int
		if err = p.db.QueryRow("SELECT count(*) FROM notification_outbox WHERE id=$1", digest(key)).Scan(&n); err != nil || n != 1 {
			t.Fatal("notification missing or duplicate", err, n)
		}
	}
	// Repeated Google sign-in is not a second registration.
	identity := googleClaims{Email: token("mailgoogle_") + "@example.invalid", Name: "Google Email Test"}
	identity.Subject = token("google_")
	g, err := p.linkGoogle(ctx, identity)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.linkGoogle(ctx, identity); err != nil {
		t.Fatal(err)
	}
	var n int
	if err = p.db.QueryRow("SELECT count(*) FROM notification_outbox WHERE id=$1", digest("welcome/"+g.ID)).Scan(&n); err != nil || n != 1 {
		t.Fatal(err, n)
	}
}
func TestDeliveryRetryPreservesPayloadAndIdempotency(t *testing.T) {
	p := databasePlatform(t)
	p.config.ResendKey = "re_test"
	p.config.MailFrom = "Vitruko <hola@vitruko.dev>"
	// Isolate dispatch from other fixtures in the explicitly disposable test database.
	if _, err := p.db.Exec("UPDATE notification_outbox SET next_attempt_at=now()+interval '1 day' WHERE sent_at IS NULL"); err != nil {
		t.Fatal(err)
	}
	tx, err := p.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	key := token("mailretry_")
	if err = p.enqueueMail(context.Background(), tx, key, "test@example.invalid", "Prueba", mailContent{Title: "Correo de prueba", ActionURL: "https://app.vitruko.dev/app"}); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	calls := 0
	firstKey := ""
	firstPayload := ""
	p.mailHTTP = testTransport(func(r *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(r.Body)
		calls++
		k := r.Header.Get("Idempotency-Key")
		if calls == 1 {
			firstKey = k
			firstPayload = string(b)
		} else if firstKey != k || firstPayload != string(b) {
			t.Fatal("retry changed request")
		}
		var m mailMessage
		if json.Unmarshal(b, &m) != nil || m.HTML == "" || m.Text == "" {
			t.Fatal("missing multipart content")
		}
		status := 500
		if calls > 1 {
			status = 200
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})
	p.dispatchMail(context.Background())
	if calls != 1 {
		t.Fatal(calls)
	}
	if _, err = p.db.Exec("UPDATE notification_outbox SET next_attempt_at=$1 WHERE id=$2", time.Now().Add(-time.Second), digest(key)); err != nil {
		t.Fatal(err)
	}
	p.dispatchMail(context.Background())
	p.dispatchMail(context.Background())
	if calls != 2 {
		t.Fatal("delivered message resent", calls)
	}
	var sent bool
	var payload string
	if err = p.db.QueryRow("SELECT sent_at IS NOT NULL,payload::text FROM notification_outbox WHERE id=$1", digest(key)).Scan(&sent, &payload); err != nil || !sent || payload != "{}" {
		t.Fatal(err, sent, payload)
	}
}

func TestScheduledCancellationAndInvoiceOrdering(t *testing.T) {
	p := databasePlatform(t)
	u, _ := signupTest(t, p)
	p.config.ResendKey = "re_test"
	p.config.MailFrom = "Vitruko <hola@vitruko.dev>"
	if _, err := p.db.Exec("UPDATE plans SET stripe_price_id='price_mail_scale' WHERE id='scale'"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.db.Exec("UPDATE users SET customer_id=$1 WHERE id=$2", "cus_"+u.ID, u.ID); err != nil {
		t.Fatal(err)
	}
	status := "active"
	cancel := false
	start := time.Now().Add(-time.Hour).Unix()
	end := time.Now().AddDate(0, 1, 0).Unix()
	subID := "sub_" + u.ID
	p.stripeHTTP = testTransport(func(r *http.Request) (*http.Response, error) {
		return stripeResponse(map[string]any{"data": []any{map[string]any{"id": subID, "status": status, "cancel_at_period_end": cancel, "metadata": map[string]string{"app": "vitruko"}, "items": map[string]any{"data": []any{map[string]any{"price": map[string]string{"id": "price_mail_scale"}, "current_period_start": start, "current_period_end": end}}}}}}), nil
	})
	run := func(kind string, invoice map[string]any, version int64, mailKind string) {
		t.Helper()
		if err := p.reconcileEvent(context.Background(), u.ID, token("evt_"), kind, invoice); err != nil {
			t.Fatal(err)
		}
		var got int64
		if err := p.db.QueryRow("SELECT notification_version FROM users WHERE id=$1", u.ID).Scan(&got); err != nil || got != version {
			t.Fatal("unexpected notification transition", err, got, version)
		}
		if mailKind != "" {
			var count int
			if err := p.db.QueryRow("SELECT count(*) FROM notification_outbox WHERE id=$1", digest(fmt.Sprintf("subscription/%s/%d/%s", u.ID, version, mailKind))).Scan(&count); err != nil || count != 1 {
				t.Fatal("missing transition email", mailKind, err)
			}
		}
	}
	run("customer.subscription.updated", nil, 1, "active")
	cancel = true
	run("customer.subscription.updated", nil, 2, "cancel_scheduled")
	run("customer.subscription.updated", nil, 2, "")
	cancel = false
	run("customer.subscription.updated", nil, 3, "cancel_reverted")
	// Subscription state may arrive before the successful renewal invoice.
	start = end
	end += 30 * 24 * 3600
	run("customer.subscription.updated", nil, 3, "")
	invoice := map[string]any{"billing_reason": "subscription_cycle", "parent": map[string]any{"subscription_details": map[string]any{"subscription": subID}}}
	run("invoice.paid", invoice, 4, "renewed")
	run("invoice.paid", map[string]any{"billing_reason": "subscription_create", "subscription": subID}, 4, "")
	status = "past_due"
	run("invoice.payment_failed", nil, 5, "payment_failed")
	status = "active"
	run("invoice.paid", nil, 6, "active")
	status = "canceled"
	run("customer.subscription.deleted", nil, 7, "canceled")
}
