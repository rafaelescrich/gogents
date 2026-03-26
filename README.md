# gogents

A **local AI agent** written in Go that uses **OpenRouter** for multiple LLM models and **RedVector** for RAG (retrieval-augmented generation). It runs on your machine and can do general tasks (not just code): file I/O, shell commands, web fetch, and RAG search over your RedVector knowledge base.

Inspired by [PicoClaw](https://github.com/sipeed/picoclaw), [OpenRouter create-agent](https://openrouter.ai/skills/create-agent/SKILL.md), and [RedVector](https://github.com/rafaelescrich/redvector).

## Features

- **OpenRouter**: Use any of 300+ models via a single API key (`openrouter/auto` or a specific model ID).
- **Tools**: Read/write/list files, run shell commands (with safety restrictions), fetch URLs, and search a RedVector RAG collection.
- **RedVector RAG**: Optional integration with [RedVector](https://github.com/rafaelescrich/redvector) (Qdrant-compatible REST API) for semantic search over your indexed documents. Requires an embedding API for query encoding.
- **Config**: Env vars or `~/.gogents/config.json`.

## Quick start

**Cloud (OpenRouter):**

1. **Get an OpenRouter API key**: [openrouter.ai/settings/keys](https://openrouter.ai/settings/keys).
2. **Build and run**:
   ```bash
   cd gogents
   go build -o gogents ./cmd/gogents
   OPENROUTER_API_KEY=sk-or-... ./gogents
   ```
   Interactive: type messages at the `>` prompt; `/quit` or Ctrl+D to exit.
3. **One-shot**: `OPENROUTER_API_KEY=sk-or-... ./gogents "What is 2+2?"`

**Local (Ollama + Arctic GGUF + RedVector):**  
No API key. Use [Ollama](https://github.com/ollama/ollama) with a Snowflake Arctic GGUF model and optionally RedVector for RAG. See **[docs/LOCAL-STACK.md](docs/LOCAL-STACK.md)** for the full flow.

**Free OpenRouter (think + tasks) + Snowflake Arctic (RAG embeddings only):**  
Use OpenRouter **free** models for the agent and **Arctic only for RAG embeddings** (e.g. Ollama with Arctic embed GGUF). See **[docs/FREE-OPENROUTER-ARCTIC-RAG.md](docs/FREE-OPENROUTER-ARCTIC-RAG.md)**.

**Use gogents as a custom LLM in Cursor:**  
Run gogents as an OpenAI-compatible HTTP API. **HTTPS** can be all in Go: set `GOGENTS_SERVE_DOMAIN=gogents.yourdomain.com` and gogents will get a free Let's Encrypt cert and serve on :443. Or use **ngrok** or a **reverse proxy**. See **[docs/CURSOR-AGENT.md](docs/CURSOR-AGENT.md)** and **[docs/HTTPS-REVERSE-PROXY.md](docs/HTTPS-REVERSE-PROXY.md)**.

## Configuration

### Environment variables

| Variable | Description |
|----------|-------------|
| `OPENROUTER_API_KEY` | OpenRouter API key. Required for cloud; omit for local (Ollama). |
| `OPENROUTER_URL` | OpenRouter base URL (default: `https://openrouter.ai/api/v1`). |
| `GOGENTS_MODEL` | Model ID (default: `openrouter/free`). Examples: `stepfun/step-3.5-flash`, `openrouter/auto`, `anthropic/claude-3.5-sonnet`. |
| `GOGENTS_WORKSPACE` | Workspace directory for file/shell tools (default: `.`). |
| `GOGENTS_CONFIG` | Path to config JSON (default: `~/.gogents/config.json`). |
| `OLLAMA_HOST` | For local backend: Ollama host (e.g. `localhost:11434`). Sets LLM URL to `http://<host>/v1`. No API key needed. |
| `LLM_BASE_URL` | Override LLM base URL (e.g. `http://localhost:11434/v1` for Ollama). |
| `REDVECTOR_URL` | RedVector REST base URL (e.g. `http://localhost:8888`) for RAG. |
| `EMBED_API_URL` | Embeddings API base URL (e.g. OpenRouter or OpenAI) for RAG query embedding. |
| `EMBED_API_KEY` | API key for embeddings. |
| `EMBED_MODEL` | Embedding model (e.g. `openai/text-embedding-3-small`). |
| `GOGENTS_SERVE` | Set to `1` to run HTTP server (OpenAI-compat) instead of CLI. |
| `GOGENTS_LISTEN` | Server listen address (default `:8080`). |
| `GOGENTS_SERVE_API_KEY` | Optional Bearer token for server; Cursor sends this as API key. |
| `GOGENTS_TLS_CERT` / `GOGENTS_TLS_KEY` | Paths to TLS cert and key for HTTPS server. |
| `GOGENTS_SERVE_DOMAIN` | Domain for automatic HTTPS (Let's Encrypt); e.g. `gogents.yourdomain.com`. Server listens on :443. |
| `GOGENTS_SERVE_ACME_EMAIL` | Email for Let's Encrypt (default: `admin@` + serve_domain). |

### Config file

Config path: `GOGENTS_CONFIG` or `~/.gogents/config.json`. Optional fields (e.g. `ollama_host`, `llm_base_url`, `instructions`) override env.

**Cloud example** — `~/.gogents/config.json` or `GOGENTS_CONFIG=./config.json`:

```json
{
  "openrouter_api_key": "sk-or-...",
  "model": "openrouter/free",
  "workspace": ".",
  "max_iterations": 10,
  "max_tokens": 8192,
  "temperature": 0.7,
  "redvector_url": "http://localhost:8888",
  "embed_api_url": "https://openrouter.ai/api/v1",
  "embed_api_key": "sk-or-...",
  "embed_model": "openai/text-embedding-3-small"
}
```

**Run with a repo-local config:** `GOGENTS_CONFIG=$PWD/config.json ./gogents` (interactive) or `GOGENTS_CONFIG=$PWD/config.json ./gogents "your question"` (one-shot).

## RedVector RAG

1. Run [RedVector](https://github.com/rafaelescrich/redvector) (REST API on port 8888).
2. Create a collection and index your documents with embeddings (same dimension as your embed model).
3. Set `REDVECTOR_URL` and optionally `EMBED_API_URL` / `EMBED_API_KEY` / `EMBED_MODEL` so the agent can embed queries and search RedVector.
4. The agent will have a `rag_search` tool: `collection` name and natural language `query`.

If you don’t set an embedder, `rag_search` will report that an embedding API is required.

## Project layout

```
gogents/
├── cmd/gogents/          # CLI entry
├── internal/
│   ├── agent/            # Agent instance and loop (tools + OpenRouter)
│   ├── config/           # Config load (env + JSON)
│   ├── openrouter/       # OpenRouter API client (OpenAI-compat)
│   ├── rag/              # RedVector REST client + optional embedder
│   ├── server/           # OpenAI-compat HTTP API (for Cursor / ngrok)
│   └── tools/            # read_file, write_file, list_dir, run_shell, web_fetch, rag_search
├── docs/                 # LOCAL-STACK, FREE-OPENROUTER-ARCTIC-RAG, CURSOR-AGENT, HTTPS-REVERSE-PROXY
├── Dockerfile            # Server image for docker-compose
├── Caddyfile.example     # Reverse proxy (HTTPS) for gogents
├── config.example.json   # Example config (cloud)
├── config.example.free-arctic.json  # Example: free OpenRouter + Arctic RAG
├── go.mod
└── README.md
```

## Testing

From the repo root:

```bash
go test ./...
```

Or with coverage: `go test ./internal/... -cover`

## License

Apache-2.0.
