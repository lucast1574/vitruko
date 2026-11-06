package platform

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGoogleTokenRequiresTrustedVerifiedIdentity(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	v := googleVerifier{keys: map[string]*rsa.PublicKey{"test": &key.PublicKey}, expires: time.Now().Add(time.Hour)}
	now := time.Now()
	base := func() jwt.MapClaims {
		return jwt.MapClaims{"sub": "google-user", "iss": "https://securetoken.google.com/vitruko-test", "aud": "vitruko-test", "exp": now.Add(time.Hour).Unix(), "iat": now.Add(-time.Second).Unix(), "auth_time": now.Add(-time.Minute).Unix(), "email": "owner@example.invalid", "email_verified": true, "firebase": map[string]any{"sign_in_provider": "google.com"}}
	}
	for _, test := range []struct {
		name   string
		mutate func(jwt.MapClaims)
	}{
		{"valid", func(c jwt.MapClaims) {}},
		{"expired", func(c jwt.MapClaims) { c["exp"] = now.Add(-time.Second).Unix() }},
		{"wrong-audience", func(c jwt.MapClaims) { c["aud"] = "another-project" }},
		{"wrong-issuer", func(c jwt.MapClaims) { c["iss"] = "https://attacker.invalid" }},
		{"unverified", func(c jwt.MapClaims) { c["email_verified"] = false }},
		{"password-provider", func(c jwt.MapClaims) { c["firebase"] = map[string]any{"sign_in_provider": "password"} }},
		{"missing-sub", func(c jwt.MapClaims) { delete(c, "sub") }},
		{"missing-expiry", func(c jwt.MapClaims) { delete(c, "exp") }},
		{"missing-issued", func(c jwt.MapClaims) { delete(c, "iat") }},
		{"future-issued", func(c jwt.MapClaims) { c["iat"] = now.Add(time.Hour).Unix() }},
		{"future-auth", func(c jwt.MapClaims) { c["auth_time"] = now.Add(time.Hour).Unix() }},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := base()
			test.mutate(c)
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, c)
			token.Header["kid"] = "test"
			raw, _ := token.SignedString(key)
			_, err := v.verify(context.Background(), raw, "vitruko-test")
			if (err == nil) != (test.name == "valid") {
				t.Fatal("token policy", err)
			}
		})
	}
	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, base())
	raw, _ := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err = v.verify(context.Background(), raw, "vitruko-test"); err == nil {
		t.Fatal("unsigned identity accepted")
	}
}
func TestGoogleLinksExistingAdminAndCreatesNormalUsers(t *testing.T) {
	p := databasePlatform(t)
	u, cookie := signupTest(t, p)
	_, err := p.db.Exec("UPDATE users SET role='admin',plan_id='internal',email_verified=true WHERE id=$1", u.ID)
	if err != nil {
		t.Fatal(err)
	}
	var before string
	_ = p.db.QueryRow("SELECT password_hash FROM users WHERE id=$1", u.ID).Scan(&before)
	identity := googleClaims{Email: u.Email, Name: "Google Owner", RegisteredClaims: jwt.RegisteredClaims{Subject: token("google_")}}
	linked, err := p.linkGoogle(context.Background(), identity)
	if err != nil || linked.ID != u.ID || linked.Role != "admin" || linked.PlanID != "internal" {
		t.Fatal("admin identity or entitlement changed", linked, err)
	}
	var after string
	_ = p.db.QueryRow("SELECT password_hash FROM users WHERE id=$1", u.ID).Scan(&after)
	if before != after {
		t.Fatal("verified owner's password changed")
	}
	if w, _ := callHTTP(p, "GET", "/console/session", "", cookie, ""); w.Code != 401 {
		t.Fatal("old session not rotated on first Google link")
	}
	again, err := p.linkGoogle(context.Background(), identity)
	if err != nil || again.ID != u.ID {
		t.Fatal("duplicate Google account", err)
	}
	identity.Subject = token("other_google_")
	if _, err = p.linkGoogle(context.Background(), identity); err == nil {
		t.Fatal("different Google UID took linked account")
	}
	identity.Email = token("new_google_") + "@example.invalid"
	identity.Subject = token("new_google_")
	created, err := p.linkGoogle(context.Background(), identity)
	if err != nil || created.Role != "user" || created.PlanID != "free" {
		t.Fatal("new Google user got privileged account", err)
	}
	_, _ = p.db.Exec("UPDATE users SET disabled=true WHERE id=$1", created.ID)
	if _, err = p.linkGoogle(context.Background(), identity); err == nil {
		t.Fatal("disabled Google account admitted")
	}
}
func TestGoogleFirstLinkRemovesUnverifiedPasswordAndSessions(t *testing.T) {
	p := databasePlatform(t)
	u, cookie := signupTest(t, p)
	identity := googleClaims{Email: u.Email, RegisteredClaims: jwt.RegisteredClaims{Subject: token("google_")}}
	if _, err := p.linkGoogle(context.Background(), identity); err != nil {
		t.Fatal(err)
	}
	var hash string
	_ = p.db.QueryRow("SELECT password_hash FROM users WHERE id=$1", u.ID).Scan(&hash)
	if !strings.HasPrefix(hash, "!google-") {
		t.Fatal("unverified pre-registered password remains usable")
	}
	if w, _ := callHTTP(p, "GET", "/console/session", "", cookie, ""); w.Code != 401 {
		t.Fatal("pre-registered session still active")
	}
}
