package platform

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type googleClaims struct {
	jwt.RegisteredClaims
	Email    string `json:"email"`
	Verified bool   `json:"email_verified"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
	AuthTime *int64 `json:"auth_time"`
	Firebase struct {
		Provider string `json:"sign_in_provider"`
	} `json:"firebase"`
}
type googleVerifier struct {
	mu      sync.Mutex
	keys    map[string]*rsa.PublicKey
	expires time.Time
	client  *http.Client
	certURL string
}

func (v *googleVerifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if time.Now().Before(v.expires) {
		if key := v.keys[kid]; key != nil {
			return key, nil
		}
		return nil, errors.New("unknown signing key")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", v.certURL, nil)
	if err != nil {
		return nil, err
	}
	res, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, errors.New("certificate service unavailable")
	}
	var certs map[string]string
	if err = json.NewDecoder(io.LimitReader(res.Body, 64<<10)).Decode(&certs); err != nil {
		return nil, err
	}
	keys := map[string]*rsa.PublicKey{}
	for id, raw := range certs {
		block, _ := pem.Decode([]byte(raw))
		if block == nil {
			continue
		}
		cert, e := x509.ParseCertificate(block.Bytes)
		if e != nil {
			continue
		}
		if key, ok := cert.PublicKey.(*rsa.PublicKey); ok {
			keys[id] = key
		}
	}
	if len(keys) == 0 {
		return nil, errors.New("no signing certificates")
	}
	age := 300
	for _, part := range strings.Split(res.Header.Get("Cache-Control"), ",") {
		if s, ok := strings.CutPrefix(strings.TrimSpace(part), "max-age="); ok {
			if n, e := strconv.Atoi(s); e == nil {
				age = max(1, min(n, 3600))
			}
		}
	}
	v.keys = keys
	v.expires = time.Now().Add(time.Duration(age) * time.Second)
	if key := keys[kid]; key != nil {
		return key, nil
	}
	return nil, errors.New("unknown signing key")
}
func (v *googleVerifier) verify(ctx context.Context, raw, project string) (googleClaims, error) {
	var claims googleClaims
	if project == "" || len(raw) > 16000 || raw == "" {
		return claims, errors.New("invalid Google token")
	}
	_, err := jwt.ParseWithClaims(raw, &claims, func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" || len(kid) > 200 {
			return nil, errors.New("invalid signing key")
		}
		return v.key(ctx, kid)
	}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer("https://securetoken.google.com/"+project), jwt.WithAudience(project), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil {
		return claims, err
	}
	now := time.Now().Unix()
	if claims.IssuedAt == nil || claims.Subject == "" || len(claims.Subject) > 128 || claims.AuthTime == nil || *claims.AuthTime < 0 || *claims.AuthTime > now || !claims.Verified || claims.Firebase.Provider != "google.com" {
		return claims, errors.New("untrusted Google identity")
	}
	claims.Email = strings.ToLower(strings.TrimSpace(claims.Email))
	if _, err = mail.ParseAddress(claims.Email); err != nil || strings.ContainsAny(claims.Email, " <>") || len(claims.Email) > 254 {
		return claims, errors.New("invalid email")
	}
	return claims, nil
}
func (p *Platform) providers(w http.ResponseWriter, r *http.Request) {
	var google any
	if p.config.FirebaseProjectID != "" && p.config.FirebaseAPIKey != "" && p.config.FirebaseAppID != "" {
		google = map[string]string{"projectId": p.config.FirebaseProjectID, "apiKey": p.config.FirebaseAPIKey, "authDomain": p.config.FirebaseProjectID + ".firebaseapp.com", "appId": p.config.FirebaseAppID}
	}
	reply(w, 200, map[string]any{"google": google})
}
func (p *Platform) googleLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		IDToken string `json:"idToken"`
	}
	if decode(w, r, &in) != nil {
		problem(w, 400, "invalid", "Acceso de Google inválido.")
		return
	}
	if p.config.FirebaseProjectID == "" {
		problem(w, 503, "google_unavailable", "Google no está configurado.")
		return
	}
	if !p.authAllowed(r, "google:"+digest(in.IDToken)) {
		problem(w, 429, "rate_limited", "Demasiados intentos. Intenta más tarde.")
		return
	}
	identity, err := p.google.verify(r.Context(), in.IDToken, p.config.FirebaseProjectID)
	if err != nil {
		problem(w, 401, "google_identity", "No se pudo verificar tu cuenta de Google. Intenta de nuevo.")
		return
	}
	u, err := p.linkGoogle(r.Context(), identity)
	if err != nil {
		problem(w, 403, "google_account", "Esta cuenta no puede acceder. Revisa el correo o contacta al administrador.")
		return
	}
	if p.startSession(w, r, u.ID) != nil {
		problem(w, 503, "session", "No se pudo iniciar sesión. Reintenta.")
		return
	}
	reply(w, 200, u)
}
func (p *Platform) linkGoogle(ctx context.Context, identity googleClaims) (User, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback()
	// Serialize the same Google identity, including concurrent first logins.
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", identity.Subject); err != nil {
		return User{}, err
	}
	var id, linked string
	err = tx.QueryRowContext(ctx, "SELECT id,COALESCE(firebase_uid,'') FROM users WHERE firebase_uid=$1 FOR UPDATE", identity.Subject).Scan(&id, &linked)
	if err == sql.ErrNoRows {
		err = tx.QueryRowContext(ctx, "SELECT id,COALESCE(firebase_uid,'') FROM users WHERE email=$1 FOR UPDATE", identity.Email).Scan(&id, &linked)
	}
	if err == sql.ErrNoRows {
		name := strings.TrimSpace(identity.Name)
		if len(name) < 2 {
			name = strings.Split(identity.Email, "@")[0]
		}
		if len(name) > 100 {
			name = string([]rune(name)[:min(50, len([]rune(name)))])
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO users(id,email,name,password_hash,firebase_uid,email_verified) VALUES($1,$2,$3,$4,$5,true) ON CONFLICT(email) DO NOTHING", token("usr_"), identity.Email, name, "!google-only", identity.Subject)
		if err == nil {
			err = tx.QueryRowContext(ctx, "SELECT id,COALESCE(firebase_uid,'') FROM users WHERE email=$1 FOR UPDATE", identity.Email).Scan(&id, &linked)
		}
	}
	if err != nil {
		return User{}, err
	}
	u, err := scanUser(tx.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE id=$1", id))
	if err != nil || u.Disabled || (linked != "" && linked != identity.Subject) || u.Email != identity.Email {
		return User{}, errors.New("account unavailable")
	}
	if linked == "" {
		// A pre-registered, unverified email must not leave an attacker's
		// password or sessions usable after the real email owner signs in.
		_, err = tx.ExecContext(ctx, "UPDATE users SET firebase_uid=$1,password_hash=CASE WHEN email_verified THEN password_hash ELSE '!google-only' END,email_verified=true WHERE id=$2", identity.Subject, id)
		if err != nil {
			return User{}, err
		}
		if _, err = tx.ExecContext(ctx, "DELETE FROM sessions WHERE user_id=$1", id); err != nil {
			return User{}, err
		}
		if _, err = tx.ExecContext(ctx, "DELETE FROM reset_tokens WHERE user_id=$1", id); err != nil {
			return User{}, err
		}
	}
	u.AvatarURL = googlePicture(identity.Picture)
	if _, err = tx.ExecContext(ctx, "UPDATE users SET avatar_url=$1 WHERE id=$2", u.AvatarURL, id); err != nil {
		return User{}, err
	}
	if err = tx.Commit(); err != nil {
		return User{}, err
	}
	return u, nil
}
