package platform

import (
	"net/http"
	"strings"
	"time"
)

type Key struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	Scopes     []string   `json:"scopes"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	RevokedAt  *time.Time `json:"revokedAt"`
}

func (p *Platform) listKeys(w http.ResponseWriter, r *http.Request) {
	rows, err := p.db.QueryContext(r.Context(), "SELECT id,name,prefix,scopes,created_at,last_used_at,revoked_at FROM api_keys WHERE user_id=$1 ORDER BY created_at DESC LIMIT 100", current(r).ID)
	if err != nil {
		problem(w, 503, "storage", "No se pudieron cargar las claves.")
		return
	}
	defer rows.Close()
	keys := []Key{}
	for rows.Next() {
		var k Key
		if rows.Scan(&k.ID, &k.Name, &k.Prefix, &k.Scopes, &k.CreatedAt, &k.LastUsedAt, &k.RevokedAt) != nil {
			problem(w, 503, "storage", "No se pudieron cargar las claves.")
			return
		}
		keys = append(keys, k)
	}
	reply(w, 200, map[string]any{"keys": keys})
}
func (p *Platform) createKey(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name   string
		Scopes []string
	}
	if decode(w, r, &in) != nil || len(strings.TrimSpace(in.Name)) < 2 || len(in.Name) > 80 || len(in.Scopes) == 0 || len(in.Scopes) > 1 {
		problem(w, 400, "invalid", "Pon un nombre a la clave y elige al menos un permiso.")
		return
	}
	seen := map[string]bool{}
	for _, s := range in.Scopes {
		if s != "extract" || seen[s] {
			problem(w, 400, "scope", "Permisos inválidos.")
			return
		}
		seen[s] = true
	}
	tx, err := p.db.BeginTx(r.Context(), nil)
	if err != nil {
		problem(w, 503, "storage", "Intenta más tarde.")
		return
	}
	defer tx.Rollback()
	var id string
	if tx.QueryRowContext(r.Context(), "SELECT id FROM users WHERE id=$1 FOR UPDATE", current(r).ID).Scan(&id) != nil {
		problem(w, 503, "storage", "Intenta más tarde.")
		return
	}
	var count int
	_ = tx.QueryRowContext(r.Context(), "SELECT count(*) FROM api_keys WHERE user_id=$1 AND revoked_at IS NULL", id).Scan(&count)
	if count >= 10 {
		problem(w, 400, "key_limit", "Puedes tener hasta 10 claves activas. Revoca una para crear otra.")
		return
	}
	raw := token("vt_live_")
	kid := token("key_")
	prefix := raw[:18] + "…" + raw[len(raw)-4:]
	_, err = tx.ExecContext(r.Context(), "INSERT INTO api_keys(id,user_id,name,prefix,token_hash,scopes) VALUES($1,$2,$3,$4,$5,$6)", kid, id, strings.TrimSpace(in.Name), prefix, digest(raw), in.Scopes)
	if err != nil || tx.Commit() != nil {
		problem(w, 503, "storage", "No se pudo crear la clave.")
		return
	}
	reply(w, 201, map[string]any{"id": kid, "key": raw, "prefix": prefix, "message": "Copia esta clave. Solo se muestra una vez."})
}
func (p *Platform) revokeKey(w http.ResponseWriter, r *http.Request) {
	res, err := p.db.ExecContext(r.Context(), "UPDATE api_keys SET revoked_at=now() WHERE id=$1 AND user_id=$2 AND revoked_at IS NULL", r.PathValue("id"), current(r).ID)
	if err != nil {
		problem(w, 503, "storage", "Intenta más tarde.")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		problem(w, 404, "not_found", "Clave no encontrada o ya revocada.")
		return
	}
	reply(w, 200, map[string]bool{"ok": true})
}
