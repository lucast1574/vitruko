package platform

import (
	"context"
	"strings"
	"testing"
)

func TestProfileSaveReloadAndLogout(t *testing.T) {
	p := databasePlatform(t)
	_, cookie := signupTest(t, p)
	w, data := callHTTP(p, "PATCH", "/console/me", `{"name":"Updated profile"}`, cookie, "")
	if w.Code != 200 || data["name"] != "Updated profile" {
		t.Fatal(w.Code, w.Body)
	}
	w, data = callHTTP(p, "GET", "/console/session", "", cookie, "")
	if w.Code != 200 || data["name"] != "Updated profile" {
		t.Fatal("profile did not persist", w.Body)
	}
	w, _ = callHTTP(p, "POST", "/console/logout", `{}`, cookie, "")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	w, _ = callHTTP(p, "GET", "/console/session", "", cookie, "")
	if w.Code != 401 {
		t.Fatal("logged-out session still valid")
	}
}

func TestAdminRoleSavePreservesStripeAndMissingPlanFails(t *testing.T) {
	p := databasePlatform(t)
	admin, cookie := signupTest(t, p)
	target, _ := signupTest(t, p)
	if _, err := p.db.Exec("UPDATE users SET role='admin' WHERE id=$1", admin.ID); err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"active", "trialing", "past_due"} {
		if _, err := p.db.Exec("UPDATE users SET subscription_id='sub_test',subscription_status=$1,plan_id='launch' WHERE id=$2", status, target.ID); err != nil {
			t.Fatal(err)
		}
		w, _ := callHTTP(p, "PATCH", "/console/admin/users/"+target.ID, `{"role":"admin","disabled":false}`, cookie, "")
		if w.Code != 200 {
			t.Fatal("role-only save failed", status, w.Code, w.Body)
		}
		fresh, err := p.user(context.Background(), target.ID)
		if err != nil || fresh.PlanID != "launch" || fresh.SubscriptionStatus != status || fresh.SubscriptionID.String != "sub_test" {
			t.Fatal("billing changed", status, err)
		}
		w, _ = callHTTP(p, "PATCH", "/console/admin/users/"+target.ID, `{"planId":"scale"}`, cookie, "")
		if w.Code != 409 {
			t.Fatal("Stripe protection bypassed", status, w.Code)
		}
	}
	w, _ := callHTTP(p, "PATCH", "/console/admin/plans/missing-plan", `{"monthlyLimit":10,"rpm":10}`, cookie, "")
	if w.Code != 404 {
		t.Fatal("nonexistent plan falsely saved", w.Code, w.Body)
	}
}

func TestSuspensionBlocksSessionsAndKeysAndCanBeRestored(t *testing.T) {
	p := databasePlatform(t)
	admin, cookie := signupTest(t, p)
	target, targetCookie := signupTest(t, p)
	key, _ := keyTest(t, p, targetCookie)
	if _, err := p.db.Exec("UPDATE users SET role='admin' WHERE id=$1", admin.ID); err != nil {
		t.Fatal(err)
	}
	for _, disabled := range []string{"true", "false"} {
		w, _ := callHTTP(p, "PATCH", "/console/admin/users/"+target.ID, `{"disabled":`+disabled+`}`, cookie, "")
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body)
		}
		w, _ = callHTTP(p, "GET", "/console/session", "", targetCookie, "")
		expected := 200
		if disabled == "true" {
			expected = 401
		}
		if w.Code != expected {
			t.Fatal("suspension session enforcement", w.Code)
		}
		method, endpoint := "POST", "/v1/extract"
		if strings.Contains(p.config.AppURL, "mapora") {
			method, endpoint = "GET", "/v1/search?q=Lima"
		}
		w, _ = callHTTP(p, method, endpoint, "", nil, key)
		expected = 200
		if disabled == "true" {
			expected = 403
		}
		if w.Code != expected {
			t.Fatal("suspension key enforcement", w.Code, w.Body)
		}
	}
}

func TestActiveKeyLimitAndFreedSlot(t *testing.T) {
	p := databasePlatform(t)
	_, cookie := signupTest(t, p)
	_, first := keyTest(t, p, cookie)
	for i := 1; i < 10; i++ {
		keyTest(t, p, cookie)
	}
	scope := "extract"
	if strings.Contains(p.config.AppURL, "mapora") {
		scope = "search"
	}
	body := `{"name":"Overflow","scopes":["` + scope + `"]}`
	w, data := callHTTP(p, "POST", "/console/keys", body, cookie, "")
	if w.Code != 400 || data["code"] != "key_limit" {
		t.Fatal("eleventh active key accepted", w.Code, w.Body)
	}
	w, _ = callHTTP(p, "DELETE", "/console/keys/"+first, "", cookie, "")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	w, _ = callHTTP(p, "POST", "/console/keys", body, cookie, "")
	if w.Code != 201 {
		t.Fatal("revocation did not free a slot", w.Code, w.Body)
	}
}
