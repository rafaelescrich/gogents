# Run your own free gogents HTTPS server

You can get a free **HTTPS** URL in two ways:

1. **All in Go (recommended)** — gogents gets its own Let's Encrypt cert and serves HTTPS on `:443`. No Caddy or ngrok.
2. **Reverse proxy** — Run gogents on `:8080` and put **Caddy** in front for TLS (optional if you prefer a separate proxy).

---

## Option 0: Automatic HTTPS in Go (no Caddy)

gogents can obtain and renew **Let's Encrypt** certificates itself using [CertMagic](https://github.com/caddyserver/certmagic) (pure Go). One binary, no reverse proxy.

### Requirements

- A **domain** pointing to your server's public IP (e.g. `gogents.yourdomain.com`).
- Ports **80** and **443** free (80 is used for ACME HTTP-01 challenge and redirect to HTTPS).

### Run

```bash
# With env
GOGENTS_SERVE=1 GOGENTS_SERVE_DOMAIN=gogents.yourdomain.com ./gogents

# Optional: email for Let's Encrypt (default: admin@<domain>)
GOGENTS_SERVE_ACME_EMAIL=you@example.com
```

Or in config (e.g. `config.json`):

```json
{
  "serve_domain": "gogents.yourdomain.com",
  "serve_acme_email": "you@example.com"
}
```

Then start: `./gogents --serve`

- **HTTPS**: https://gogents.yourdomain.com/v1/chat/completions  
- **HTTP** on port 80 redirects to HTTPS.

First run will request a certificate from Let's Encrypt; certs are cached and renewed automatically.

---

## Option A–C: Reverse proxy (Caddy)

If you prefer a separate reverse proxy:

```
Internet (HTTPS) → Caddy (reverse proxy, TLS) → gogents (HTTP on localhost:8080)
```

- **Caddy**: Listens on 443, gets free certs from Let's Encrypt, forwards to gogents.
- **gogents**: Runs in server mode on `:8080` (HTTP only).

## Option A: Caddy (recommended, auto HTTPS)

### 1. Install Caddy

- **macOS**: `brew install caddy`
- **Linux**: [caddyserver.com/docs/install](https://caddyserver.com/docs/install)

### 2. Run gogents in server mode

```bash
cd /path/to/gogents
GOGENTS_CONFIG=$PWD/config.json ./gogents --serve
```

Leave it running (or run in background / as a service). It listens on `http://127.0.0.1:8080`.

### 3. Caddyfile (reverse proxy to gogents)

Create `Caddyfile` in the project (or any path):

```caddyfile
# Replace with your domain. Caddy will get a free TLS cert automatically.
gogents.yourdomain.com {
    reverse_proxy localhost:8080
}
```

If you don't have a domain yet, use a placeholder and run Caddy in front of gogents for local HTTPS only (see "Local only" below).

### 4. Start Caddy

```bash
caddy run --config Caddyfile
```

Or with reload: `caddy run --config Caddyfile --watch`

- First run: Caddy will request a Let's Encrypt certificate for `gogents.yourdomain.com`. Your DNS for that host must point to this machine’s public IP.
- Your gogents API is then available at **https://gogents.yourdomain.com** (e.g. `https://gogents.yourdomain.com/v1/chat/completions`).

### 5. Use in Cursor

- **Base URL**: `https://gogents.yourdomain.com`
- **Model**: e.g. `gogents`
- **API key**: optional; set `GOGENTS_SERVE_API_KEY` when starting gogents and use the same value in Cursor.

---

## Option B: Local HTTPS only (no public domain)

If you only need HTTPS on your machine (e.g. Cursor on the same host):

### Caddyfile (localhost with self-signed cert)

```caddyfile
https://localhost:8443 {
    tls internal
    reverse_proxy localhost:8080
}
```

Run:

```bash
caddy run --config Caddyfile
```

Use **Base URL** `https://localhost:8443` in Cursor. You may need to accept the self-signed cert in the OS/browser once.

---

## Option C: Docker Compose (gogents + Caddy)

Run both in one place. You need a domain pointing to the host.

**docker-compose.yml** (in project root or next to Caddyfile):

```yaml
services:
  gogents:
    build: .
    ports: []   # not exposed; only Caddy talks to it
    environment:
      - GOGENTS_SERVE=1
      - GOGENTS_LISTEN=:8080
      - OPENROUTER_API_KEY=${OPENROUTER_API_KEY}
      - GOGENTS_MODEL=${GOGENTS_MODEL:-openrouter/free}
    volumes:
      - ./config.json:/app/config.json:ro
    restart: unless-stopped

  caddy:
    image: caddy:latest
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
    environment:
      - GOGENTS_DOMAIN=${GOGENTS_DOMAIN:-gogents.yourdomain.com}
    restart: unless-stopped
    depends_on:
      - gogents

volumes:
  caddy_data: {}
```

**Caddyfile** for Docker (use env for domain): copy `Caddyfile.docker.example` to `Caddyfile`:

```caddyfile
{$GOGENTS_DOMAIN} {
    reverse_proxy gogents:8080
}
```

Create a **Dockerfile** in the repo root if missing:

```dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o gogents ./cmd/gogents

FROM alpine:latest
WORKDIR /app
COPY --from=build /app/gogents .
EXPOSE 8080
ENTRYPOINT ["./gogents", "--serve"]
```

Run:

```bash
cp docker-compose.example.yml docker-compose.yml
cp Caddyfile.docker.example Caddyfile
export GOGENTS_DOMAIN=gogents.yourdomain.com
export OPENROUTER_API_KEY=sk-or-...
docker compose up -d
```

You get **https://gogents.yourdomain.com** as your free gogents HTTPS server. Ensure DNS for that host points to this machine.

---

## Summary

| Goal | Solution |
|------|----------|
| Free HTTPS URL with your domain | Caddy in front of gogents (Option A or C). |
| Local HTTPS only | Caddy with `tls internal` (Option B). |
| One-command deploy | Docker Compose (Option C). |

All options use gogents in **server mode** (`--serve` on `:8080`); the reverse proxy provides HTTPS and (with a domain) a stable public URL.
