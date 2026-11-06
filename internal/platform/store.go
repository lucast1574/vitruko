package platform

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
	"time"
)

//go:embed schema.sql
var schema string

type Config struct {
	DatabaseURL, AdminEmail, AdminPassword, AppURL, SiteURL, StripeKey, StripeWebhookSecret, StripePortalConfig, ResendKey, MailFrom string
	StripePrices                                                                                                                     map[string]string
	FirebaseProjectID, FirebaseAPIKey, FirebaseAppID                                                                                 string
}
type Platform struct {
	db         *sql.DB
	config     Config
	geo        httpHandler
	stripeHTTP httpClient
	mailHTTP   httpClient
	google     googleVerifier
}

func token(prefix string) string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return prefix + hex.EncodeToString(b)
}
func digest(value string) string { h := sha256.Sum256([]byte(value)); return hex.EncodeToString(h[:]) }

func New(ctx context.Context, c Config, geo httpHandler) (*Platform, error) {
	db, err := sql.Open("pgx", c.DatabaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(15)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	// A transaction-scoped advisory lock serializes schema bootstrap across replicas.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		db.Close()
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(178304622)"); err == nil {
		_, err = tx.ExecContext(ctx, schema)
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		db.Close()
		return nil, err
	}
	p := &Platform{db: db, config: c, geo: geo, stripeHTTP: defaultHTTP(), mailHTTP: defaultHTTP(), google: googleVerifier{client: defaultHTTP(), certURL: "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"}}
	// The explicitly configured bootstrap owner is already trusted by deployment.
	if c.AdminEmail != "" {
		if _, err = db.ExecContext(ctx, "UPDATE users SET email_verified=true WHERE email=$1 AND role='admin'", c.AdminEmail); err != nil {
			db.Close()
			return nil, err
		}
	}
	for plan, price := range c.StripePrices {
		if price != "" {
			if _, err = db.ExecContext(ctx, "UPDATE plans SET stripe_price_id=$1 WHERE id=$2 AND stripe_price_id=''", price, plan); err != nil {
				return nil, err
			}
		}
	}
	if c.AdminEmail != "" {
		var existing string
		err = db.QueryRowContext(ctx, "SELECT id FROM users WHERE email=$1", c.AdminEmail).Scan(&existing)
		if errors.Is(err, sql.ErrNoRows) {
			if len(c.AdminPassword) < 12 {
				return nil, errors.New("bootstrap admin password is required")
			}
			hashed, e := bcrypt.GenerateFromPassword([]byte(c.AdminPassword), 12)
			if e != nil {
				return nil, e
			}
			_, err = db.ExecContext(ctx, "INSERT INTO users(id,email,name,password_hash,role,plan_id,email_verified) VALUES($1,$2,'Lucas',$3,'admin','internal',true) ON CONFLICT(email) DO NOTHING", token("usr_"), c.AdminEmail, string(hashed))
		}
		if err != nil {
			return nil, fmt.Errorf("bootstrap admin: %w", err)
		}
	}
	return p, nil
}

type User struct {
	ID                 string         `json:"id"`
	Email              string         `json:"email"`
	Name               string         `json:"name"`
	Role               string         `json:"role"`
	Disabled           bool           `json:"disabled"`
	PlanID             string         `json:"planId"`
	CustomerID         sql.NullString `json:"-"`
	SubscriptionID     sql.NullString `json:"-"`
	SubscriptionStatus string         `json:"subscriptionStatus"`
	BillingStart       sql.NullTime   `json:"-"`
	BillingEnd         sql.NullTime   `json:"-"`
	CreatedAt          time.Time      `json:"createdAt"`
}

const userColumns = "id,email,name,role,disabled,plan_id,customer_id,subscription_id,subscription_status,billing_start,billing_end,created_at"

type scanner interface{ Scan(...any) error }

func scanUser(row scanner) (u User, err error) {
	err = row.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Disabled, &u.PlanID, &u.CustomerID, &u.SubscriptionID, &u.SubscriptionStatus, &u.BillingStart, &u.BillingEnd, &u.CreatedAt)
	return
}
func (p *Platform) user(ctx context.Context, id string) (User, error) {
	return scanUser(p.db.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE id=$1", id))
}
func (p *Platform) Close() error { return p.db.Close() }
