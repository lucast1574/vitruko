package platform

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"io"
	"log/slog"
	"net/http"
	texttemplate "text/template"
	"time"
)

// React Email compiles these during the frontend build; runtime remains Go-only.
//
//go:embed mailtemplates/*
var mailTemplates embed.FS

type mailContent struct{ Preview, Label, Title, Name, Body, Detail, Action, ActionURL, Footnote string }
type mailMessage struct {
	ReplyTo string   `json:"reply_to,omitempty"`
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
	Text    string   `json:"text"`
}

func (p *Platform) renderMail(to, subject string, c mailContent) (mailMessage, error) {
	m := mailMessage{ReplyTo: p.config.MailReplyTo, From: p.config.MailFrom, To: []string{to}, Subject: subject}
	for _, ext := range []string{"html", "txt"} {
		source, err := mailTemplates.ReadFile("mailtemplates/transactional." + ext)
		if err != nil {
			return m, err
		}
		var b bytes.Buffer
		if ext == "html" {
			t, e := htmltemplate.New("email").Parse(string(source))
			if e != nil {
				return m, e
			}
			err = t.Execute(&b, c)
		} else {
			t, e := texttemplate.New("email").Parse(string(source))
			if e != nil {
				return m, e
			}
			err = t.Execute(&b, c)
		}
		if err != nil {
			return m, err
		}
		if ext == "html" {
			m.HTML = b.String()
		} else {
			m.Text = b.String()
		}
	}
	return m, nil
}
func (p *Platform) enqueueMail(ctx context.Context, tx *sql.Tx, key, to, subject string, c mailContent) error {
	if p.config.ResendKey == "" || p.config.MailFrom == "" {
		return nil
	}
	m, err := p.renderMail(to, subject, c)
	if err != nil {
		return err
	}
	body, err := json.Marshal(m)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO notification_outbox(id,payload) VALUES($1,$2) ON CONFLICT(id) DO NOTHING`, digest(key), string(body))
	return err
}
func (p *Platform) newUserMail(ctx context.Context, tx *sql.Tx, id, email, name string) error {
	c := mailContent{Preview: "Tu cuenta de Vitruko está lista.", Label: "Tu cuenta", Title: "Tu próximo proyecto empieza aquí.", Name: name, Body: "Te damos la bienvenida a Vitruko. Ya puedes convertir enlaces públicos de tiendas en fichas de producto con nombre, imágenes, precio y disponibilidad cuando la fuente los publica.", Detail: "Tu plan Free incluye 1.000 peticiones al mes. Crea una API key y consulta las guías cuando estés listo para integrar.", Action: "Abrir mi cuenta", ActionURL: p.config.AppURL + "/app", Footnote: "Recibes este correo porque se creó una cuenta con esta dirección. Si no fuiste tú, puedes ignorarlo."}
	if err := p.enqueueMail(ctx, tx, "welcome/"+id, email, "Te damos la bienvenida a Vitruko", c); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, "SELECT email,name FROM users WHERE role='admin' AND NOT disabled ORDER BY id")
	if err != nil {
		return err
	}
	type recipient struct{ email, name string }
	admins := []recipient{}
	for rows.Next() {
		var a recipient
		if err = rows.Scan(&a.email, &a.name); err != nil {
			rows.Close()
			return err
		}
		admins = append(admins, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, a := range admins {
		c.Name = a.name
		c.Preview = "Una nueva cuenta se registró en Vitruko."
		c.Label = "Aviso para administración"
		c.Title = "Una persona más en Vitruko."
		c.Body = name + " acaba de crear una cuenta en Vitruko."
		c.Detail = "Correo: " + email + ". Plan inicial: Free. Este aviso confirma el registro; no confirma la verificación del correo."
		c.Action = "Ver usuarios"
		c.ActionURL = p.config.AppURL + "/app/admin"
		c.Footnote = "Recibes este aviso porque tu cuenta tiene el rol de administrador en Vitruko."
		if err = p.enqueueMail(ctx, tx, "admin-signup/"+id+"/"+digest(a.email), a.email, "Nuevo usuario en Vitruko", c); err != nil {
			return err
		}
	}
	return nil
}
func (p *Platform) subscriptionMail(ctx context.Context, tx *sql.Tx, u User, version int64, kind, plan string, end int64) error {
	c := mailContent{Name: u.Name, Label: "Tu suscripción", Action: "Ver mi plan", ActionURL: p.config.AppURL + "/app/billing", Footnote: "Este es un aviso de servicio de Vitruko. Puedes consultar los detalles de tu suscripción en tu cuenta."}
	switch kind {
	case "active":
		c.Title = "Tu suscripción está activa."
		c.Body = "Ya tienes acceso al plan " + plan + " de Vitruko."
		c.Detail = "Consulta la cuota, el consumo y la próxima fecha de facturación desde tu cuenta."
	case "changed":
		c.Title = "Tu plan se actualizó."
		c.Body = "Tu cuenta de Vitruko ahora utiliza el plan " + plan + "."
		c.Detail = "Los permisos de tus claves se mantienen. Consulta los límites y tu consumo en el panel."
	case "cancel_scheduled":
		c.Title = "Recibimos tu cancelación."
		c.Body = "Tu suscripción de Vitruko se cancelará al terminar el periodo actual."
		c.Detail = "Conservas el acceso del plan hasta el " + time.Unix(end, 0).UTC().Format("02/01/2006") + ". Después, tu cuenta continuará en Free."
	case "cancel_reverted":
		c.Title = "Tu suscripción continúa."
		c.Body = "Se retiró la cancelación programada de tu suscripción de Vitruko."
		c.Detail = "Tu plan continuará renovándose según el periodo de facturación de tu cuenta."
	case "canceled":
		c.Title = "Tu suscripción finalizó."
		c.Body = "Tu cuenta de Vitruko continúa disponible con el plan Free."
		c.Detail = "Puedes consultar tus claves y consumo, y elegir un plan nuevamente cuando lo necesites."
	case "payment_failed":
		c.Title = "Revisa el pago de tu suscripción."
		c.Body = "La suscripción de Vitruko registra un pago pendiente."
		c.Detail = "Revisa tu método de pago desde el portal de facturación. El acceso depende del estado actual de tu suscripción."
	case "renewed":
		c.Title = "Tu suscripción se renovó."
		c.Body = "Tu plan " + plan + " de Vitruko continúa activo para un nuevo periodo."
		c.Detail = "El nuevo periodo termina el " + time.Unix(end, 0).UTC().Format("02/01/2006") + ". Consulta tus facturas desde tu cuenta."
	default:
		return fmt.Errorf("unknown notification kind")
	}
	c.Preview = c.Title
	return p.enqueueMail(ctx, tx, fmt.Sprintf("subscription/%s/%d/%s", u.ID, version, kind), u.Email, c.Title+" | Vitruko", c)
}

// Claim with a lease: other replicas skip the row. Resend's idempotency key covers
// a crash after acceptance but before our acknowledgement. Never retry past 23h.
func (p *Platform) dispatchMail(ctx context.Context) {
	if p.config.ResendKey == "" || p.config.MailFrom == "" {
		return
	}
	for i := 0; i < 10; i++ {
		var id string
		var payload []byte
		var attempts int
		err := p.db.QueryRowContext(ctx, `UPDATE notification_outbox SET attempts=attempts+1,next_attempt_at=now()+interval '2 minutes' WHERE id=(SELECT id FROM notification_outbox WHERE sent_at IS NULL AND failed_at IS NULL AND next_attempt_at<=now() AND attempts<12 AND created_at>now()-interval '23 hours' ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED) RETURNING id,payload,attempts`).Scan(&id, &payload, &attempts)
		if err == sql.ErrNoRows {
			return
		}
		if err != nil {
			slog.Warn("email queue unavailable")
			return
		}
		req, e := http.NewRequestWithContext(ctx, "POST", "https://api.resend.com/emails", bytes.NewReader(payload))
		if e != nil {
			return
		}
		req.Header.Set("Authorization", "Bearer "+p.config.ResendKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "TransactionalMail/1.0")
		req.Header.Set("Idempotency-Key", "vitruko/"+id)
		res, e := p.mailHTTP.Do(req)
		status := 0
		if e == nil {
			status = res.StatusCode
			io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
			res.Body.Close()
		}
		if status >= 200 && status < 300 {
			_, e = p.db.ExecContext(ctx, "UPDATE notification_outbox SET sent_at=now(),payload='{}'::jsonb WHERE id=$1", id)
		} else {
			permanent := status >= 400 && status < 500 && status != 408 && status != 409 && status != 429
			delay := time.Duration(min(3600, 15*(1<<min(attempts, 8)))) * time.Second
			_, e = p.db.ExecContext(ctx, `UPDATE notification_outbox SET next_attempt_at=$2,last_status=$3,failed_at=CASE WHEN $4 THEN now() ELSE NULL END WHERE id=$1`, id, time.Now().Add(delay), status, permanent || attempts >= 12)
			slog.Warn("email delivery pending", "notificationId", id, "status", status, "attempt", attempts)
		}
		if e != nil {
			slog.Warn("email acknowledgement pending", "notificationId", id)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(600 * time.Millisecond):
		}
	}
}
func (p *Platform) maintainMail(ctx context.Context) {
	p.dispatchMail(ctx)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.dispatchMail(ctx)
		}
	}
}
func (p *Platform) sendReset(r *http.Request, email, raw string) error {
	c := mailContent{Preview: "Restablece tu contraseña de Vitruko.", Label: "Seguridad de tu cuenta", Title: "Una nueva contraseña para tu cuenta.", Name: "", Body: "Recibimos una solicitud para restablecer tu contraseña de Vitruko.", Detail: "El enlace caduca en 30 minutos y solo se puede utilizar una vez.", Action: "Restablecer contraseña", ActionURL: p.config.AppURL + "/reset?token=" + raw, Footnote: "Si no lo solicitaste, ignora este correo. Tu contraseña no cambia hasta que completes el proceso."}
	c.Name = "de nuevo"
	m, err := p.renderMail(email, "Restablece tu contraseña de Vitruko", c)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(m)
	req, err := http.NewRequestWithContext(r.Context(), "POST", "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.config.ResendKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TransactionalMail/1.0")
	req.Header.Set("Idempotency-Key", "vitruko/reset/"+digest(raw))
	res, err := p.mailHTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
	if res.StatusCode >= 300 {
		return fmt.Errorf("mail status %d", res.StatusCode)
	}
	return nil
}
