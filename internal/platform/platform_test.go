package platform

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWebhookSignature(t *testing.T) {
	body := []byte(`{"id":"evt_123"}`)
	now := time.Now()
	timestamp := strconv.FormatInt(now.Unix(), 10)
	mac := hmac.New(sha256.New, []byte("test-secret"))
	mac.Write([]byte(timestamp + "."))
	mac.Write(body)
	header := "t=" + timestamp + ",v1=" + hex.EncodeToString(mac.Sum(nil))
	if !validSignature(header, body, "test-secret", now) {
		t.Fatal("valid signature rejected")
	}
	if validSignature(header, []byte(`{}`), "test-secret", now) || validSignature(header, body, "wrong", now) || validSignature(header, body, "test-secret", now.Add(6*time.Minute)) {
		t.Fatal("invalid signature accepted")
	}
}
func TestPeriods(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	u := User{PlanID: "free"}
	start, end := period(u, now)
	if start.Day() != 1 || end.Month() != 12 {
		t.Fatal(start, end)
	}
}

func databasePlatform(t *testing.T) *Platform {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for database integration tests")
	}
	if !strings.Contains(dsn, "vitruko_test") {
		t.Fatal("refusing a database without the vitruko_test name")
	}
	p, err := New(context.Background(), Config{DatabaseURL: dsn, AppURL: "https://app-vitruko.nexode.app", StripeKey: "sk_test_placeholder"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reply(w, 200, map[string]bool{"ok": true}) }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Close() })
	return p
}
func callHTTP(p *Platform, method, path, body string, cookie *http.Cookie, key string) (*httptest.ResponseRecorder, map[string]any) {
	r := httptest.NewRequest(method, "https://app-vitruko.nexode.app"+path, strings.NewReader(body))
	if cookie != nil {
		r.AddCookie(cookie)
	}
	if key != "" {
		r.Header.Set("X-API-Key", key)
	}
	w := httptest.NewRecorder()
	p.Handler().ServeHTTP(w, r)
	var data map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &data)
	return w, data
}
func signupTest(t *testing.T, p *Platform) (User, *http.Cookie) {
	t.Helper()
	email := token("test_") + "@example.invalid"
	w, data := callHTTP(p, "POST", "/console/signup", `{"email":"`+email+`","name":"Vitruko Test","password":"a-unique-test-password-123"}`, nil, "")
	if w.Code != 201 {
		t.Fatalf("signup %d %s", w.Code, w.Body)
	}
	u, err := p.user(context.Background(), text(data["id"]))
	if err != nil {
		t.Fatal(err)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure || !cookies[0].HttpOnly {
		t.Fatal("unsafe session cookie")
	}
	return u, cookies[0]
}
func keyTest(t *testing.T, p *Platform, cookie *http.Cookie) (string, string) {
	t.Helper()
	w, data := callHTTP(p, "POST", "/console/keys", `{"name":"Test integration","scopes":["extract"]}`, cookie, "")
	if w.Code != 201 {
		t.Fatalf("key creation %d %s", w.Code, w.Body)
	}
	return text(data["key"]), text(data["id"])
}

func TestAccountOwnershipAndRevocation(t *testing.T) {
	p := databasePlatform(t)
	_, one := signupTest(t, p)
	_, two := signupTest(t, p)
	key, id := keyTest(t, p, one)
	w, _ := callHTTP(p, "GET", "/console/session", "", one, "")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	w, _ = callHTTP(p, "GET", "/console/admin/users", "", one, "")
	if w.Code != 403 {
		t.Fatal("normal user reached admin")
	}
	w, _ = callHTTP(p, "DELETE", "/console/keys/"+id, "", two, "")
	if w.Code != 404 {
		t.Fatal("cross-user revocation allowed")
	}
	w, _ = callHTTP(p, "POST", "/v1/extract", "", nil, key)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	w, _ = callHTTP(p, "POST", "/console/keys", `{"name":"Invalid legacy scope","scopes":["search"]}`, one, "")
	if w.Code != 400 {
		t.Fatal("scope bypass")
	}
	w, data := callHTTP(p, "GET", "/console/keys", "", one, "")
	if strings.Contains(w.Body.String(), key) || data["keys"] == nil {
		t.Fatal("raw key leaked")
	}
	w, _ = callHTTP(p, "DELETE", "/console/keys/"+id, "", one, "")
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	w, _ = callHTTP(p, "POST", "/v1/extract", "", nil, key)
	if w.Code != 401 {
		t.Fatal("revoked key accepted")
	}
}

func TestConcurrentQuotaCannotOverspend(t *testing.T) {
	p := databasePlatform(t)
	u, cookie := signupTest(t, p)
	key, _ := keyTest(t, p, cookie)
	plan := token("plan_")
	_, err := p.db.Exec("INSERT INTO plans(id,name,monthly_limit,rpm,visible) VALUES($1,'Test',10,1000,false)", plan)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.db.Exec("UPDATE users SET plan_id=$1 WHERE id=$2", plan, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	var success, rejected atomic.Int64
	var group sync.WaitGroup
	for i := 0; i < 40; i++ {
		group.Go(func() {
			w, _ := callHTTP(p, "POST", "/v1/extract", "", nil, key)
			switch w.Code {
			case 200:
				success.Add(1)
			case 429:
				rejected.Add(1)
			default:
				t.Errorf("unexpected %d %s", w.Code, w.Body)
			}
		})
	}
	group.Wait()
	if success.Load() != 10 || rejected.Load() != 30 {
		t.Fatal(success.Load(), rejected.Load())
	}
	var used, events int64
	_ = p.db.QueryRow("SELECT sum(used) FROM usage_buckets WHERE user_id=$1", u.ID).Scan(&used)
	_ = p.db.QueryRow("SELECT count(*) FROM usage_events WHERE user_id=$1", u.ID).Scan(&events)
	if used != 10 || events != 10 {
		t.Fatal("usage mismatch", used, events)
	}
}

func TestUnlimitedStillCountsRequestsAndErrors(t *testing.T) {
	p := databasePlatform(t)
	u, cookie := signupTest(t, p)
	key, _ := keyTest(t, p, cookie)
	_, err := p.db.Exec("UPDATE users SET plan_id='scale' WHERE id=$1", u.ID)
	if err != nil {
		t.Fatal(err)
	}
	p.geo = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { problem(w, 422, "test_error", "test") })
	for i := 0; i < 3; i++ {
		w, _ := callHTTP(p, "POST", "/v1/extract", "", nil, key)
		if w.Code != 422 || w.Header().Get("X-Quota-Limit") != "unlimited" {
			t.Fatal(w.Code, w.Body)
		}
	}
	w, data := callHTTP(p, "GET", "/console/usage", "", cookie, "")
	if w.Code != 200 || data["used"] != float64(3) || data["unlimited"] != true {
		t.Fatal(w.Code, w.Body)
	}
}

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) Do(r *http.Request) (*http.Response, error) { return f(r) }
func stripeResponse(v any) *http.Response {
	b, _ := json.Marshal(v)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(b))), Header: make(http.Header)}
}
func TestStripeReconciliationAndCancellation(t *testing.T) {
	p := databasePlatform(t)
	u, _ := signupTest(t, p)
	_, err := p.db.Exec("UPDATE plans SET stripe_price_id='price_test_scale' WHERE id='scale'")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.db.Exec("UPDATE users SET customer_id=$1 WHERE id=$2", "cus_"+u.ID, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	status := "active"
	calls := 0
	p.stripeHTTP = testTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Path != "/v1/subscriptions" {
			t.Fatal(r.URL)
		}
		return stripeResponse(map[string]any{"data": []any{map[string]any{"id": "sub_" + u.ID, "status": status, "metadata": map[string]string{"app": "vitruko"}, "items": map[string]any{"data": []any{map[string]any{"price": map[string]string{"id": "price_test_scale"}, "current_period_start": time.Now().Add(-time.Hour).Unix(), "current_period_end": time.Now().AddDate(0, 1, 0).Unix()}}}}}}), nil
	})
	event := token("evt_")
	if err = p.reconcile(context.Background(), u.ID, event); err != nil {
		t.Fatal(err)
	}
	fresh, _ := p.user(context.Background(), u.ID)
	if fresh.PlanID != "scale" {
		t.Fatal(fresh.PlanID)
	}
	if err = p.reconcile(context.Background(), u.ID, event); err != nil || calls != 1 {
		t.Fatal("duplicate event processed", calls, err)
	}
	status = "canceled"
	if err = p.reconcile(context.Background(), u.ID, token("evt_")); err != nil {
		t.Fatal(err)
	}
	fresh, _ = p.user(context.Background(), u.ID)
	if fresh.PlanID != "free" {
		t.Fatal("cancellation kept paid access")
	}
}

func TestPasswordResetTokenSingleUse(t *testing.T) {
	p := databasePlatform(t)
	u, cookie := signupTest(t, p)
	raw := token("reset_")
	_, err := p.db.Exec("INSERT INTO reset_tokens(token_hash,user_id,expires_at) VALUES($1,$2,now()+interval '10 minutes')", digest(raw), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"token":%q,"password":"replacement-password-123"}`, raw)
	w, _ := callHTTP(p, "POST", "/console/reset", body, nil, "")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	w, _ = callHTTP(p, "POST", "/console/reset", body, nil, "")
	if w.Code != 400 {
		t.Fatal("reset token reused")
	}
	w, _ = callHTTP(p, "GET", "/console/session", "", cookie, "")
	if w.Code != 401 {
		t.Fatal("old session survived reset")
	}
}

func TestCheckoutAttemptReusesUnknownResult(t *testing.T) {
	p := databasePlatform(t)
	u, _ := signupTest(t, p)
	first, _, err := p.checkoutAttempt(context.Background(), u.ID, "launch")
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := p.checkoutAttempt(context.Background(), u.ID, "launch")
	if err != nil || first != second {
		t.Fatal("unknown result created a different attempt", first, second, err)
	}
	if _, _, err = p.checkoutAttempt(context.Background(), u.ID, "scale"); err == nil {
		t.Fatal("another plan bypassed pending attempt")
	}
	_, err = p.db.Exec("UPDATE checkout_attempts SET session_id='cs_existing' WHERE user_id=$1", u.ID)
	if err != nil {
		t.Fatal(err)
	}
	p.stripeHTTP = testTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method != "GET" {
			t.Fatal("unexpected external creation")
		}
		return stripeResponse(map[string]any{"status": "open", "url": "https://checkout.stripe.com/test-session"}), nil
	})
	_, existing, err := p.checkoutAttempt(context.Background(), u.ID, "launch")
	if err != nil || existing != "https://checkout.stripe.com/test-session" {
		t.Fatal(existing, err)
	}
}

func TestPerMinuteLimitSharedAcrossKeys(t *testing.T) {
	p := databasePlatform(t)
	u, cookie := signupTest(t, p)
	one, _ := keyTest(t, p, cookie)
	two, _ := keyTest(t, p, cookie)
	plan := token("plan_")
	_, err := p.db.Exec("INSERT INTO plans(id,name,monthly_limit,rpm,visible) VALUES($1,'Rate test',100,2,false)", plan)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.db.Exec("UPDATE users SET plan_id=$1 WHERE id=$2", plan, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	for i, key := range []string{one, two, one} {
		w, _ := callHTTP(p, "POST", "/v1/extract", "", nil, key)
		expected := 200
		if i == 2 {
			expected = 429
		}
		if w.Code != expected {
			t.Fatal(i, w.Code, w.Body)
		}
	}
}

func TestAdminCanAssignPlanButCannotDisableSelf(t *testing.T) {
	p := databasePlatform(t)
	admin, cookie := signupTest(t, p)
	target, _ := signupTest(t, p)
	_, err := p.db.Exec("UPDATE users SET role='admin',plan_id='internal' WHERE id=$1", admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	w, _ := callHTTP(p, "PATCH", "/console/admin/users/"+target.ID, `{"planId":"scale","role":"user","disabled":false}`, cookie, "")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	fresh, _ := p.user(context.Background(), target.ID)
	if fresh.PlanID != "scale" {
		t.Fatal("plan not assigned")
	}
	w, _ = callHTTP(p, "PATCH", "/console/admin/users/"+admin.ID, `{"disabled":true}`, cookie, "")
	if w.Code != 400 {
		t.Fatal("admin self-suspension allowed")
	}
}
