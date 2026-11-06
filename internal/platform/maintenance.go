package platform

import (
	"context"
	"log/slog"
	"time"
)

func (p *Platform) Maintain(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.maintainOnce(ctx)
		}
	}
}
func (p *Platform) maintainOnce(ctx context.Context) {
	for _, query := range []string{"DELETE FROM sessions WHERE expires_at<now()", "DELETE FROM reset_tokens WHERE expires_at<now()", "DELETE FROM auth_attempts WHERE window_start<now()-interval '2 days'", "DELETE FROM rate_buckets WHERE minute<now()-interval '2 days'", "DELETE FROM usage_events WHERE created_at<now()-interval '180 days'"} {
		_, _ = p.db.ExecContext(ctx, query)
	}
	rows, err := p.db.QueryContext(ctx, "SELECT id FROM users WHERE customer_id IS NOT NULL AND subscription_status IN ('active','trialing','past_due') ORDER BY billing_end LIMIT 1000")
	if err != nil {
		return
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		job, cancel := context.WithTimeout(ctx, 18*time.Second)
		err = p.reconcile(job, id, "")
		cancel()
		if err != nil {
			slog.Warn("subscription reconciliation pending", "userId", id)
		}
	}
}
