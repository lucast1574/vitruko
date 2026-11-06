package platform

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"net"
	"net/http"
	"net/mail"
	"strings"
)

func (p *Platform) authAllowed(r *http.Request, email string) bool {
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	for _, v := range []struct {
		key string
		max int
	}{{"ip:" + ip, 50}, {"email:" + email, 12}} {
		var count int
		err := p.db.QueryRowContext(r.Context(), `INSERT INTO auth_attempts(identifier,window_start,count) VALUES($1,date_trunc('hour',now()),1)
   ON CONFLICT(identifier) DO UPDATE SET count=CASE WHEN auth_attempts.window_start=EXCLUDED.window_start THEN auth_attempts.count+1 ELSE 1 END,window_start=EXCLUDED.window_start RETURNING count`, digest(v.key)).Scan(&count)
		if err != nil || count > v.max {
			return false
		}
	}
	return true
}
func (p *Platform) startSession(w http.ResponseWriter, r *http.Request, id string) error {
	raw := token("ses_")
	_, err := p.db.ExecContext(r.Context(), "INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,now()+interval '30 days')", digest(raw), id)
	if err == nil {
		newSession(w, raw)
	}
	return err
}
func (p *Platform) signup(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Name, Password string }
	if decode(w, r, &in) != nil {
		problem(w, 400, "invalid", "Revisa los datos de la cuenta.")
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Name = strings.TrimSpace(in.Name)
	if _, err := mail.ParseAddress(in.Email); err != nil || strings.ContainsAny(in.Email, " <>") || len(in.Email) > 254 || len(in.Name) < 2 || len(in.Name) > 100 || !validPassword(in.Password) {
		problem(w, 400, "invalid", "Usa un correo válido, tu nombre y una contraseña de 12 a 72 caracteres.")
		return
	}
	if !p.authAllowed(r, in.Email) {
		problem(w, 429, "rate_limited", "Demasiados intentos. Vuelve a intentarlo más tarde.")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), 12)
	if err != nil {
		problem(w, 500, "internal", "No se pudo crear la cuenta.")
		return
	}
	id := token("usr_")
	_, err = p.db.ExecContext(r.Context(), "INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,$3,$4)", id, in.Email, in.Name, string(hash))
	if err != nil {
		problem(w, 409, "account_exists", "No se pudo crear la cuenta. Si ya tienes una, inicia sesión o recupera tu contraseña.")
		return
	}
	if p.startSession(w, r, id) != nil {
		problem(w, 503, "session", "Cuenta creada. Inicia sesión para continuar.")
		return
	}
	u, _ := p.user(r.Context(), id)
	reply(w, 201, u)
}
func (p *Platform) login(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password string }
	if decode(w, r, &in) != nil {
		problem(w, 400, "invalid", "Datos inválidos.")
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if !p.authAllowed(r, in.Email) {
		problem(w, 429, "rate_limited", "Demasiados intentos. Vuelve a intentarlo más tarde.")
		return
	}
	var id, hash string
	var disabled bool
	err := p.db.QueryRowContext(r.Context(), "SELECT id,password_hash,disabled FROM users WHERE email=$1", in.Email).Scan(&id, &hash, &disabled)
	if err != nil {
		hash = "$2a$12$5QfKSTmg9LKlc7lH.AwBk.DlrxraP3TI9D5kKQF1RgVKswTUx1SZm"
	}
	valid := bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) == nil
	if err != nil || !valid || disabled {
		problem(w, 401, "credentials", "Correo o contraseña incorrectos.")
		return
	}
	if p.startSession(w, r, id) != nil {
		problem(w, 503, "session", "No se pudo iniciar sesión.")
		return
	}
	u, _ := p.user(r.Context(), id)
	reply(w, 200, u)
}
func (p *Platform) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("__Host-vitruko_session"); err == nil {
		_, _ = p.db.ExecContext(r.Context(), "DELETE FROM sessions WHERE token_hash=$1", digest(c.Value))
	}
	newSession(w, "")
	reply(w, 200, map[string]bool{"ok": true})
}
func (p *Platform) me(w http.ResponseWriter, r *http.Request) { reply(w, 200, current(r)) }
func (p *Platform) profile(w http.ResponseWriter, r *http.Request) {
	var in struct{ Name string }
	if decode(w, r, &in) != nil || len(strings.TrimSpace(in.Name)) < 2 || len(in.Name) > 100 {
		problem(w, 400, "invalid", "Escribe un nombre de 2 a 100 caracteres.")
		return
	}
	_, err := p.db.ExecContext(r.Context(), "UPDATE users SET name=$1 WHERE id=$2", strings.TrimSpace(in.Name), current(r).ID)
	if err != nil {
		problem(w, 503, "storage", "No se pudo actualizar el perfil.")
		return
	}
	u, _ := p.user(r.Context(), current(r).ID)
	reply(w, 200, u)
}
func (p *Platform) password(w http.ResponseWriter, r *http.Request) {
	var in struct{ CurrentPassword, NewPassword string }
	if decode(w, r, &in) != nil || !validPassword(in.NewPassword) {
		problem(w, 400, "invalid", "La nueva contraseña debe tener de 12 a 72 caracteres.")
		return
	}
	var hash string
	_ = p.db.QueryRowContext(r.Context(), "SELECT password_hash FROM users WHERE id=$1", current(r).ID).Scan(&hash)
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.CurrentPassword)) != nil {
		problem(w, 400, "password", "La contraseña actual no es correcta.")
		return
	}
	p.setPassword(w, r, current(r).ID, in.NewPassword)
}
func (p *Platform) setPassword(w http.ResponseWriter, r *http.Request, id, password string) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		problem(w, 500, "internal", "No se pudo cambiar la contraseña.")
		return
	}
	tx, err := p.db.BeginTx(r.Context(), nil)
	if err != nil {
		problem(w, 503, "storage", "Intenta más tarde.")
		return
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(r.Context(), "UPDATE users SET password_hash=$1 WHERE id=$2", string(hash), id); err == nil {
		_, err = tx.ExecContext(r.Context(), "DELETE FROM sessions WHERE user_id=$1", id)
	}
	raw := token("ses_")
	if err == nil {
		_, err = tx.ExecContext(r.Context(), "INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,now()+interval '30 days')", digest(raw), id)
	}
	if err != nil || tx.Commit() != nil {
		problem(w, 503, "storage", "Intenta más tarde.")
		return
	}
	newSession(w, raw)
	reply(w, 200, map[string]bool{"ok": true})
}
func (p *Platform) forgot(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email string }
	_ = decode(w, r, &in)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if !p.authAllowed(r, in.Email) {
		problem(w, 429, "rate_limited", "Intenta de nuevo más tarde.")
		return
	}
	var id string
	err := p.db.QueryRowContext(r.Context(), "SELECT id FROM users WHERE email=$1 AND NOT disabled", in.Email).Scan(&id)
	if err == nil {
		raw := token("rst_")
		_, err = p.db.ExecContext(r.Context(), "INSERT INTO reset_tokens(token_hash,user_id,expires_at) VALUES($1,$2,now()+interval '30 minutes')", digest(raw), id)
		if err == nil && p.config.ResendKey != "" {
			_ = p.sendReset(r, in.Email, raw)
		}
	}
	reply(w, 200, map[string]string{"message": "Si existe una cuenta con ese correo, recibirás un enlace para cambiar la contraseña."})
}
func (p *Platform) sendReset(r *http.Request, email, raw string) error {
	data, _ := json.Marshal(map[string]any{"from": p.config.MailFrom, "to": []string{email}, "subject": "Restablece tu contraseña de Vitruko", "text": "Abre este enlace para definir tu contraseña de Vitruko. Caduca en 30 minutos:\n\n" + p.config.AppURL + "/reset?token=" + raw + "\n\nSi no lo solicitaste, ignora este correo."})
	req, _ := http.NewRequestWithContext(r.Context(), "POST", "https://api.resend.com/emails", bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+p.config.ResendKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := p.mailHTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("mail status %d", res.StatusCode)
	}
	return nil
}
func (p *Platform) reset(w http.ResponseWriter, r *http.Request) {
	var in struct{ Token, Password string }
	if decode(w, r, &in) != nil || !validPassword(in.Password) {
		problem(w, 400, "invalid", "Revisa la contraseña.")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), 12)
	if err != nil {
		problem(w, 500, "internal", "Intenta más tarde.")
		return
	}
	tx, err := p.db.BeginTx(r.Context(), nil)
	if err != nil {
		problem(w, 503, "storage", "Intenta más tarde.")
		return
	}
	defer tx.Rollback()
	var id string
	err = tx.QueryRowContext(r.Context(), "DELETE FROM reset_tokens WHERE token_hash=$1 AND expires_at>now() RETURNING user_id", digest(in.Token)).Scan(&id)
	if err == sql.ErrNoRows {
		problem(w, 400, "token", "El enlace no es válido o ya caducó.")
		return
	}
	if err == nil {
		_, err = tx.ExecContext(r.Context(), "UPDATE users SET password_hash=$1,email_verified=true WHERE id=$2 AND NOT disabled", string(hash), id)
	}
	if err == nil {
		_, err = tx.ExecContext(r.Context(), "DELETE FROM sessions WHERE user_id=$1", id)
	}
	if err == nil {
		_, err = tx.ExecContext(r.Context(), "DELETE FROM reset_tokens WHERE user_id=$1", id)
	}
	if err != nil || tx.Commit() != nil {
		problem(w, 503, "storage", "Intenta más tarde.")
		return
	}
	reply(w, 200, map[string]bool{"ok": true})
}
