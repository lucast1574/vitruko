# Vitruko

Go API + React/Vite console for public product metadata, extracted from The Date linkpreview implementation. Own PostgreSQL, account sessions, API keys, quotas and Nexode Stripe products. No geography dependencies.

## API

POST /v1/extract with application/json `{ "url": "https://store.example/products/item" }` and X-API-Key or Authorization: Bearer. Scope: extract. Returns title, imageUrl, description, url. Missing optional metadata stays empty. HTTPS public URLs only, no credentials/custom ports. DNS and each redirect are checked; connection uses the validated address, without environment proxies. HTML bounded to 4 MiB, time budget 12 seconds. Shopify/VTEX public metadata fallbacks and Mercado Libre selected-item normalization preserved. No JS execution or anti-bot bypass.

Successful results cache in memory up to 5 minutes, max 256 entries; eight simultaneous upstream extractions per replica. Every admitted authenticated call counts once, including cached calls and downstream errors. Invalid keys, denied scopes and exhausted quota are not charged. Quota reservations lock the account row in PostgreSQL, shared across keys/replicas. No raw URLs or keys in usage history.

## Run

Build web before Go: `npm --prefix web ci`, `npm --prefix web run build`, `go test ./...`, `go vet ./...`, `go build ./cmd/api`. Dockerfile embeds the prerendered web build in the Go executable; runtime has no Node dependency. Set DATABASE_URL, ADMIN_EMAIL, ADMIN_PASSWORD, APP_URL, SITE_URL, FIREBASE_PROJECT_ID/API_KEY/APP_ID, STRIPE_SECRET_KEY/WEBHOOK_SECRET/PRICE_LAUNCH/PRICE_SCALE/PORTAL_CONFIG, RESEND_API_KEY, MAIL_FROM. Secrets are deployment-only.

Database tests require TEST_DATABASE_URL pointing at a disposable database named vitruko_test. They verify concurrent quota, ownership, rate sharing, Google token validation, admin access and Stripe reconciliation. Never use production for integration tests.

## Public URLs

Landing https://vitruko.nexode.app · Console https://app-vitruko.nexode.app · API https://api-vitruko.nexode.app

Six prerendered public pages, canonical URLs, Open Graph images, structured data, sitemap and robots. Private console/auth routes use noindex; missing pages return 404. The animated brand entrance runs once per tab, respects reduced motion and never gates the rendered content.
