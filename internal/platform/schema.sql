CREATE TABLE IF NOT EXISTS plans (
 id text PRIMARY KEY, name text NOT NULL, monthly_limit bigint NOT NULL CHECK(monthly_limit>=0),
 rpm integer NOT NULL CHECK(rpm>0), price_cents integer NOT NULL DEFAULT 0,
 stripe_price_id text NOT NULL DEFAULT '', visible boolean NOT NULL DEFAULT true
);
INSERT INTO plans(id,name,monthly_limit,rpm,price_cents,visible) VALUES
 ('free','Free',1000,30,0,true),('launch','Launch',25000,120,900,true),
 ('scale','Scale',0,300,2900,true),('internal','Interno',0,600,0,false)
 ON CONFLICT(id) DO NOTHING;
CREATE TABLE IF NOT EXISTS users (
 id text PRIMARY KEY, email text UNIQUE NOT NULL, name text NOT NULL,
 password_hash text NOT NULL, role text NOT NULL DEFAULT 'user' CHECK(role IN ('user','admin')),
 disabled boolean NOT NULL DEFAULT false, plan_id text NOT NULL DEFAULT 'free' REFERENCES plans(id),
 customer_id text UNIQUE, subscription_id text UNIQUE, subscription_status text NOT NULL DEFAULT 'none',
 billing_start timestamptz, billing_end timestamptz,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS sessions (
 token_hash text PRIMARY KEY,user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 expires_at timestamptz NOT NULL
);
ALTER TABLE users ADD COLUMN IF NOT EXISTS firebase_uid text;
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified boolean NOT NULL DEFAULT false;
CREATE UNIQUE INDEX IF NOT EXISTS users_firebase_uid ON users(firebase_uid) WHERE firebase_uid IS NOT NULL;
CREATE INDEX IF NOT EXISTS session_expiry ON sessions(expires_at);
CREATE TABLE IF NOT EXISTS api_keys (
 id text PRIMARY KEY,user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 name text NOT NULL,prefix text NOT NULL,token_hash text UNIQUE NOT NULL,
 scopes text[] NOT NULL DEFAULT ARRAY['extract'],
 created_at timestamptz NOT NULL DEFAULT now(),last_used_at timestamptz,revoked_at timestamptz
);
CREATE INDEX IF NOT EXISTS keys_owner ON api_keys(user_id);
CREATE TABLE IF NOT EXISTS usage_buckets (
 user_id text NOT NULL REFERENCES users(id),period_start timestamptz NOT NULL,
 used bigint NOT NULL DEFAULT 0 CHECK(used>=0),PRIMARY KEY(user_id,period_start)
);
CREATE TABLE IF NOT EXISTS rate_buckets (
 user_id text NOT NULL REFERENCES users(id),minute timestamptz NOT NULL,used integer NOT NULL DEFAULT 0,
 PRIMARY KEY(user_id,minute)
);
CREATE TABLE IF NOT EXISTS usage_events (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,user_id text NOT NULL REFERENCES users(id),
 key_id text NOT NULL REFERENCES api_keys(id),endpoint text NOT NULL,status integer NOT NULL,
 charged boolean NOT NULL,latency_ms bigint NOT NULL,created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS events_owner_time ON usage_events(user_id,created_at);
CREATE TABLE IF NOT EXISTS auth_attempts (
 identifier text PRIMARY KEY,window_start timestamptz NOT NULL,count integer NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS reset_tokens (
 token_hash text PRIMARY KEY,user_id text NOT NULL REFERENCES users(id),expires_at timestamptz NOT NULL
);
CREATE TABLE IF NOT EXISTS stripe_events (
 id text PRIMARY KEY,processed_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS checkout_attempts (
 user_id text PRIMARY KEY REFERENCES users(id),plan_id text NOT NULL,
 attempt_id text NOT NULL,session_id text,created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS audit_events (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,actor_id text NOT NULL REFERENCES users(id),
 action text NOT NULL,target_id text NOT NULL,created_at timestamptz NOT NULL DEFAULT now()
);
