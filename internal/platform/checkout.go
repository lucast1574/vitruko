package platform

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"time"
)

// A durable attempt precedes the external request. Unknown results reuse Stripe's
// idempotency key instead of creating a second subscription checkout.
func (p *Platform) checkoutAttempt(ctx context.Context, userID, planID string) (string, string, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback()
	u, err := scanUser(tx.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE id=$1 FOR UPDATE", userID))
	if err != nil {
		return "", "", err
	}
	if u.SubscriptionStatus == "active" || u.SubscriptionStatus == "trialing" || u.SubscriptionStatus == "past_due" {
		return "", "", errors.New("Ya tienes una suscripción. Gestiona tu facturación.")
	}
	var attempt, existingPlan string
	var session sql.NullString
	var created time.Time
	err = tx.QueryRowContext(ctx, "SELECT attempt_id,plan_id,session_id,created_at FROM checkout_attempts WHERE user_id=$1", userID).Scan(&attempt, &existingPlan, &session, &created)
	if err != nil && err != sql.ErrNoRows {
		return "", "", err
	}
	if err == nil {
		if session.Valid {
			value, e := p.stripe(ctx, "GET", "/checkout/sessions/"+url.PathEscape(session.String), nil, "")
			if e != nil {
				return "", "", e
			}
			switch text(value["status"]) {
			case "open":
				if existingPlan != planID {
					return "", "", errors.New("Tienes un checkout abierto para otro plan. Complétalo o espera a que caduque.")
				}
				return attempt, text(value["url"]), nil
			case "complete":
				return "", "", errors.New("Tu checkout ya se completó. Sincroniza la facturación para continuar.")
			case "expired":
				_, err = tx.ExecContext(ctx, "DELETE FROM checkout_attempts WHERE user_id=$1", userID)
				if err != nil {
					return "", "", err
				}
				attempt = ""
			default:
				return "", "", errors.New("No se pudo determinar el estado del checkout anterior.")
			}
		} else if existingPlan != planID || time.Since(created) > 23*time.Hour {
			return "", "", errors.New("Hay un intento de pago pendiente de reconciliación. Contacta al soporte.")
		}
	}
	if attempt == "" {
		attempt = token("checkout_")
		_, err = tx.ExecContext(ctx, "INSERT INTO checkout_attempts(user_id,plan_id,attempt_id) VALUES($1,$2,$3)", userID, planID, attempt)
		if err != nil {
			return "", "", err
		}
	}
	return attempt, "", tx.Commit()
}
