# Vitruko

## Google Login con dominio propio

Configura `FIREBASE_PROJECT_ID`, `FIREBASE_API_KEY` y `FIREBASE_APP_ID`. El hostname HTTPS de `SITE_URL` (`vitruko.lat`) se publica como `authDomain`; sin un sitio HTTPS válido se conserva el dominio predeterminado del proyecto. El servidor sirve `/__/auth/*` mediante un proxy al proyecto Firebase y elimina las cookies y credenciales locales de la solicitud saliente. Autoriza los dominios públicos en Firebase y registra `https://vitruko.lat/__/auth/handler` en el cliente OAuth, junto con los orígenes HTTPS de la web y del panel.

Go API + React/Vite console for public product metadata, extracted from The Date linkpreview implementation. Own PostgreSQL, account sessions, API keys, quotas and Nexode Stripe products. No geography dependencies.

## API

POST /v1/extract with application/json `{ "url": "https://store.example/products/item" }` and X-API-Key or Authorization: Bearer. Scope: extract. Returns the original title, imageUrl, description, url plus brand, images (up to 20 distinct HTTPS URLs), price (nullable {amount: decimal string in major units, currency: published three-letter code}), normalized availability and quality (productDetected, missingFields, fieldSources, warnings, extractedAt UTC). No inferred currency, arbitrary offer selection or current-stock guarantee. Missing prices stay null; missing availability is unknown. Images are URLs only, not downloaded or stored. Missing legacy string fields stay empty. HTTPS public URLs only, no credentials/custom ports. DNS and each redirect are checked; connection uses the validated address, without environment proxies. HTML bounded to 4 MiB, time budget 12 seconds. Shopify/VTEX public metadata fallbacks and Mercado Libre selected-item normalization preserved. No JS execution or anti-bot bypass.

Successful results cache in memory up to 5 minutes, max 256 entries; eight simultaneous upstream extractions per replica. Every admitted authenticated call counts once, including cached calls and downstream errors. Invalid keys, denied scopes and exhausted quota are not charged. Quota reservations lock the account row in PostgreSQL, shared across keys/replicas. No raw URLs or keys in usage history.

## Run

Build web before Go: `npm --prefix web ci`, `npm --prefix web run build`, `go test ./...`, `go vet ./...`, `go build ./cmd/api`. Dockerfile embeds the prerendered web build in the Go executable; runtime has no Node dependency. Set DATABASE_URL, ADMIN_EMAIL, ADMIN_PASSWORD, APP_URL, SITE_URL, FIREBASE_PROJECT_ID/API_KEY/APP_ID, STRIPE_SECRET_KEY/WEBHOOK_SECRET/PRICE_LAUNCH/PRICE_SCALE/PORTAL_CONFIG, RESEND_API_KEY, MAIL_FROM. Secrets are deployment-only.

Database tests require TEST_DATABASE_URL pointing at a disposable database named vitruko_test. They verify concurrent quota, ownership, rate sharing, Google token validation, admin access and Stripe reconciliation. Never use production for integration tests.

## Public URLs

Landing https://vitruko.lat · Console https://app.vitruko.lat · API https://api.vitruko.lat

Six prerendered public pages, canonical URLs, Open Graph images, structured data, sitemap and robots. Private console/auth routes use noindex; missing pages return 404. The animated brand entrance runs once per tab, respects reduced motion and never gates the rendered content.

## MCP para Claude Code y Codex

Producción: [sitio](https://vitruko.lat), [panel](https://app.vitruko.lat),
[API](https://api.vitruko.lat), [documentación](https://vitruko.lat/documentacion),
[guía MCP](https://vitruko.lat/mcp) y [OpenAPI](https://api.vitruko.lat/openapi.json).

Abre **Conectar mi asistente** en el Resumen del panel y copia las instrucciones para tu asistente.
La API key privada se prepara automáticamente y se reutiliza entre asistentes.
Se guarda cifrada con AES-256-GCM y AAD por usuario; el hash se valida como el
resto de API keys. Se puede revocar desde API keys; copiar no revive claves revocadas
sin la acción explícita de renovar. La conexión cuenta en el límite de 10 claves.

Endpoint Streamable HTTP: `https://api.vitruko.lat/mcp`, con
`Authorization: Bearer TU_API_KEY`. El SDK oficial implementa el protocolo.
`vitruko_me` consulta cuenta, permisos y cuota sin gastar peticiones.
`vitruko_documentation` devuelve el OpenAPI actual. Las herramientas de datos
reutilizan las validaciones, permisos, protección de proveedores y cuotas REST.
No hay herramientas para modificar suscripciones ni efectuar pagos.

Configurar `MCP_KEY_ENCRYPTION_KEY` como 32 bytes hexadecimales aleatorios por
entorno. Mantenerla estable entre despliegues y protegerla como secreto de producción.
No reemplazarla sin migrar los secretos existentes. Una conexión MCP sin esta
variable falla de forma cerrada; el resto de la plataforma continúa funcionando.

SEO: páginas públicas prerenderizadas, canonical al dominio principal, metadatos
sociales, JSON-LD, sitemap y `llms.txt`. El panel, autenticación y operaciones
de API permanecen fuera del índice. El subdominio API dirige la portada y
documentación al sitio canónico. Las URLs de OAuth son
`https://vitruko.lat/__/auth/handler`, con helpers servidos bajo el mismo dominio.
