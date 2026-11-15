package platform

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestMCPCredentialEncryptionBindsOwnerAndRejectsTampering(t *testing.T) {
	t.Setenv("MCP_KEY_ENCRYPTION_KEY", strings.Repeat("ab", 32))
	a, err := sealMCP("private-test-secret", "owner-a")
	if err != nil {
		t.Fatal(err)
	}
	b, err := sealMCP("private-test-secret", "owner-a")
	if err != nil || a == b || strings.Contains(a, "private-test-secret") {
		t.Fatal("nonce or encryption failure")
	}
	if v, err := openMCP(a, "owner-a"); err != nil || v != "private-test-secret" {
		t.Fatal("roundtrip", err)
	}
	if _, err := openMCP(a, "owner-b"); err == nil {
		t.Fatal("cross-owner recovery accepted")
	}
	if _, err := openMCP(a[:len(a)-4]+"AAAA", "owner-a"); err == nil {
		t.Fatal("tampering accepted")
	}
	t.Setenv("MCP_KEY_ENCRYPTION_KEY", "")
	if _, err := sealMCP("x", "y"); err == nil {
		t.Fatal("missing master key accepted")
	}
}
func TestMCPRejectsMissingAuthenticationAndUntrustedOrigin(t *testing.T) {
	p := &Platform{}
	for _, test := range []struct {
		origin, method string
		status         int
	}{{"", "POST", 401}, {"https://evil.invalid", "POST", 403}, {"", "PUT", 405}} {
		r := httptest.NewRequest(test.method, "https://api.vitruko.lat/mcp", strings.NewReader(`{}`))
		r.Header.Set("Origin", test.origin)
		w := httptest.NewRecorder()
		p.mcpHandler().ServeHTTP(w, r)
		if w.Code != test.status || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal(w.Code, w.Body)
		}
	}
}
func TestMCPProvisioningIsolationRevocationAndQuota(t *testing.T) {
	t.Setenv("MCP_KEY_ENCRYPTION_KEY", strings.Repeat("ab", 32))
	p := databasePlatform(t)
	user, cookie := signupTest(t, p)
	_, otherCookie := signupTest(t, p)
	prepare := func(c *http.Cookie, renew bool) (int, map[string]any) {
		body := `{"renew":false}`
		if renew {
			body = `{"renew":true}`
		}
		w, d := callHTTP(p, "POST", "/console/mcp/prepare", body, c, "")
		return w.Code, d
	}
	status, first := prepare(cookie, false)
	if status != 200 {
		t.Fatal(status, first)
	}
	key := text(first["key"])
	status, second := prepare(cookie, false)
	if status != 200 || second["key"] != key {
		t.Fatal("copy rotated connection", status)
	}
	status, other := prepare(otherCookie, false)
	if status != 200 || other["key"] == key {
		t.Fatal("credential reused across accounts")
	}
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			status, v := prepare(cookie, false)
			if status != 200 || v["key"] != key {
				t.Error("concurrent prepare", status)
			}
		})
	}
	wg.Wait()
	var count int
	p.db.QueryRow("SELECT count(*) FROM api_keys WHERE user_id=$1", user.ID).Scan(&count)
	if count != 1 {
		t.Fatal("copy created duplicate keys", count)
	}
	rpc := func(key, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "https://api.vitruko.lat/mcp", strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+key)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Accept", "application/json, text/event-stream")
		r.Header.Set("MCP-Protocol-Version", "2025-11-25")
		w := httptest.NewRecorder()
		p.Handler().ServeHTTP(w, r)
		return w
	}
	init := rpc(key, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","clientInfo":{"name":"test","version":"1"},"capabilities":{}}}`)
	if init.Code != 200 || !strings.Contains(init.Body.String(), "serverInfo") {
		t.Fatal("initialize", init.Code, init.Body)
	}
	list := rpc(key, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if list.Code != 200 || !strings.Contains(list.Body.String(), "vitruko_documentation") || strings.Contains(list.Body.String(), key) {
		t.Fatal("catalog", list.Code, list.Body)
	}
	me := rpc(key, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"vitruko_me","arguments":{}}}`)
	if me.Code != 200 || !strings.Contains(me.Body.String(), "monthlyLimit") {
		t.Fatal("identity", me.Code, me.Body)
	}
	var used int
	p.db.QueryRow("SELECT COALESCE(sum(used),0) FROM usage_buckets WHERE user_id=$1", user.ID).Scan(&used)
	if used != 0 {
		t.Fatal("metadata consumed quota")
	}
	data := rpc(key, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"vitruko_extract","arguments":{"url":"https://example.com/product"}}}`)
	if data.Code != 200 || strings.Contains(data.Body.String(), `"isError":true`) {
		t.Fatal("data", data.Code, data.Body)
	}
	p.db.QueryRow("SELECT COALESCE(sum(used),0) FROM usage_buckets WHERE user_id=$1", user.ID).Scan(&used)
	if used != 1 {
		t.Fatal("data bypassed quota", used)
	}
	var scopes []string
	p.db.QueryRow("SELECT scopes FROM api_keys WHERE token_hash=$1", digest(key)).Scan(&scopes)
	p.db.Exec("UPDATE api_keys SET scopes=ARRAY[]::text[] WHERE token_hash=$1", digest(key))
	deniedScope := rpc(key, `{"jsonrpc":"2.0","id":40,"method":"tools/call","params":{"name":"vitruko_extract","arguments":{"url":"https://example.com/product"}}}`)
	if !strings.Contains(deniedScope.Body.String(), "403") {
		t.Fatal("scope bypass", deniedScope.Body)
	}
	p.db.Exec("UPDATE api_keys SET scopes=$1 WHERE token_hash=$2", scopes, digest(key))
	// Exhaustion is handled by exactly the same reservation path as REST.
	p.db.Exec("UPDATE usage_buckets SET used=1000 WHERE user_id=$1", user.ID)
	limited := rpc(key, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"vitruko_extract","arguments":{"url":"https://example.com/product"}}}`)
	if !strings.Contains(limited.Body.String(), "429") || !strings.Contains(limited.Body.String(), `"isError":true`) {
		t.Fatal("quota bypass", limited.Body)
	}
	p.db.Exec("UPDATE api_keys SET revoked_at=now() WHERE token_hash=$1", digest(key))
	if denied := rpc(key, `{"jsonrpc":"2.0","id":6,"method":"tools/list"}`); denied.Code != 401 {
		t.Fatal("revoked key accepted", denied.Code)
	}
	status, _ = prepare(cookie, false)
	if status != 409 {
		t.Fatal("copy revived revoked connection", status)
	}
	status, renewed := prepare(cookie, true)
	if status != 200 || renewed["key"] == key {
		t.Fatal("explicit renewal failed", status)
	}
	var schema []mcpOperation
	if json.Unmarshal(mcpCatalog, &schema) != nil {
		t.Fatal("catalog malformed")
	}
	// A suspended owner cannot connect with a newly issued key.
	p.db.Exec("UPDATE users SET disabled=true WHERE id=$1", user.ID)
	if denied := rpc(text(renewed["key"]), `{"jsonrpc":"2.0","id":7,"method":"tools/list"}`); denied.Code != 401 {
		t.Fatal("suspended account accepted", denied.Code)
	}
}
