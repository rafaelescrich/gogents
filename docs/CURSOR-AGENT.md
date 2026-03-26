# Use gogents as a custom LLM in Cursor (HTTP API + ngrok)

Run gogents as an **OpenAI-compatible HTTP API** so Cursor (or any client) can use it as a custom model. Cursor requires **HTTPS**, so you either expose local gogents with **ngrok** or run gogents with **TLS** directly.

## 1. Start gogents in server mode

```bash
cd /path/to/gogents
go build -o gogents ./cmd/gogents
GOGENTS_CONFIG=$PWD/config.json ./gogents --serve
```

Or with env:

```bash
GOGENTS_SERVE=1 GOGENTS_LISTEN=:8080 ./gogents
```

By default the server listens on **:8080** (HTTP). Endpoints:

- **POST /v1/chat/completions** — OpenAI-compatible chat (Cursor calls this).
- **GET /health** — returns `ok`.

Optional: require an API key so only Cursor (with the key) can call the server:

```bash
GOGENTS_SERVE_API_KEY=your-secret-key ./gogents --serve
```

Then Cursor must send `Authorization: Bearer your-secret-key`.

## 2. Expose with ngrok (HTTPS for Cursor)

Cursor only talks to **HTTPS** endpoints. Use ngrok to get a public HTTPS URL that forwards to your local server.

1. **Install ngrok**: [ngrok.com/download](https://ngrok.com/download) or `brew install ngrok`.
2. **Start gogents** (as above) on `:8080`.
3. **Tunnel**:
   ```bash
   ngrok http 8080
   ```
4. Copy the **HTTPS** URL ngrok shows (e.g. `https://abc123.ngrok-free.app`). That is your **base URL** for Cursor; the API path is `/v1`, so base URL = `https://abc123.ngrok-free.app` (no `/v1` in the base — Cursor appends `/v1` for chat).

## 3. Configure Cursor to use gogents

In Cursor:

1. Open **Settings** → **Cursor Settings** → **Models** (or the place where you set custom models).
2. Add a **Custom** / **OpenAI-compatible** model with:
   - **Base URL**: your ngrok HTTPS URL, e.g. `https://abc123.ngrok-free.app`
   - **Model**: any name (e.g. `gogents`); the server ignores the model field and uses your config.
   - **API key**: if you set `GOGENTS_SERVE_API_KEY`, use that same value; otherwise you can often leave it blank or use a placeholder.

Cursor will send `POST {baseUrl}/v1/chat/completions`; gogents implements that endpoint.

When you use TLS (cert + key), the server speaks **HTTP/2** (h2) by default, which Cursor and other clients can use.

## 4. Optional: TLS (no ngrok)

To serve HTTPS directly (e.g. on a VPS or with a local cert):

```bash
./gogents --serve
```

With config (or env) set:

- `serve_tls_cert` / `GOGENTS_TLS_CERT` — path to PEM cert.
- `serve_tls_key` / `GOGENTS_TLS_KEY` — path to PEM key.

Then listen on something like `:443` and use that HTTPS URL as the base URL in Cursor (or put ngrok in front of 443 if you prefer).

## 5. Config summary

| Env / config           | Description |
|------------------------|-------------|
| `GOGENTS_SERVE=1` or `--serve` | Run HTTP server instead of CLI. |
| `GOGENTS_LISTEN` / `serve_addr` | Listen address (default `:8080`). |
| `GOGENTS_SERVE_API_KEY` / `serve_api_key` | Optional Bearer token for `/v1/chat/completions`. |
| `GOGENTS_TLS_CERT` / `serve_tls_cert` | TLS cert file for HTTPS. |
| `GOGENTS_TLS_KEY` / `serve_tls_key`  | TLS key file. |

## 6. Flow

1. You run gogents with OpenRouter (or Ollama) + tools + optional RAG in config.
2. You start gogents in server mode and expose it via ngrok (or TLS).
3. Cursor sends chat requests to your ngrok HTTPS URL.
4. gogents runs the **agent** (LLM + tools) and returns the final assistant message in OpenAI format.

So Cursor gets one “model” that is actually your full gogents agent (file, shell, web, RAG) behind a single OpenAI-compat endpoint.
