# Local stack: Arctic GGUF + Ollama + RedVector + gogents

Run gogents **fully locally** with a GGUF-quantized [Snowflake Arctic](https://huggingface.co/Snowflake/snowflake-arctic-instruct) model via [Ollama](https://github.com/ollama/ollama) and [RedVector](https://github.com/rafaelescrich/redvector) for RAG — no cloud API keys.

## Why Ollama (not importing its code)

[Ollama](https://github.com/ollama/ollama) is a **Go server** that runs GGUF models (via llama.cpp). It is built as a single binary and is **not** published as an importable Go library: the repo is one application (CGo, native deps, server + runner). You use it by:

1. Running the Ollama server (`ollama serve` or the `ollama` app).
2. Calling its **OpenAI-compatible** HTTP API at `http://localhost:11434/v1` (see [Ollama OpenAI compatibility](https://docs.ollama.com/api/openai-compatibility)).

gogents reuses the same OpenAI-compat client: set the base URL to Ollama and the model name to your Ollama model (e.g. `arctic`). No API key is required for local backends.

## 1. Install Ollama and RedVector

- **Ollama**: [Install Ollama](https://ollama.com) and start it (e.g. `ollama serve` or the desktop app).
- **RedVector**: Build and run from [rafaelescrich/redvector](https://github.com/rafaelescrich/redvector) (REST on port 8888), or use Docker:

  ```bash
  docker run -d -p 6379:6379 -p 8888:8888 -p 50051:50051 redvector:latest
  ```

## 2. Add Snowflake Arctic (GGUF) to Ollama

Use a GGUF build of [Snowflake Arctic Instruct](https://huggingface.co/Snowflake/snowflake-arctic-instruct), e.g. [sszymczyk/snowflake-arctic-instruct-GGUF](https://huggingface.co/sszymczyk/snowflake-arctic-instruct-GGUF) (Q4_K_M or similar).

**Option A – Modelfile from local GGUF**

```bash
# Download a GGUF (e.g. Q4_K_M) to a path, then:
echo 'FROM /path/to/snowflake-arctic-instruct-Q4_K_M.gguf' > Modelfile
ollama create arctic
```

**Option B – Modelfile from Hugging Face**

If Ollama supports `FROM` with a Hugging Face URL or you use a local clone:

```dockerfile
# Modelfile
FROM ./snowflake-arctic-instruct-Q4_K_M.gguf
```

Then:

```bash
ollama create arctic
ollama run arctic "Hello"
```

Confirm the model name (e.g. `arctic` or `arctic:latest`).

## 3. Run gogents against Ollama + RedVector

Use **Ollama** for chat and **RedVector** for RAG. No OpenRouter key.

**Environment:**

```bash
# LLM: Ollama (OpenAI-compat at /v1)
export OLLAMA_HOST=localhost:11434
# or: export LLM_BASE_URL=http://localhost:11434/v1

# Model name in Ollama
export GOGENTS_MODEL=arctic

# RAG: RedVector
export REDVECTOR_URL=http://localhost:8888

# Optional: embeddings for RAG (e.g. Ollama embedding model or another local service)
# export EMBED_API_URL=http://localhost:11434/v1
# export EMBED_MODEL=your-embed-model
```

**Config file** (`~/.gogents/config.json`) alternative:

```json
{
  "ollama_host": "localhost:11434",
  "model": "arctic",
  "redvector_url": "http://localhost:8888",
  "workspace": "."
}
```

Then run gogents:

```bash
go run ./cmd/gogents
# or: ./gogents "List files in this directory"
```

gogents will use Ollama for chat (and tool calls if the model supports them) and RedVector for the `rag_search` tool when RAG is configured.

## 4. One-app style: single entrypoint

To make it “one app” from the user’s perspective, use a small script or Docker Compose that starts Ollama, RedVector, and gogents.

**Example script (run everything in the background):**

```bash
#!/bin/bash
# Start Ollama (if not already running)
ollama serve &
# Start RedVector (if using Docker)
docker run -d -p 6379:6379 -p 8888:8888 redvector:latest
# Wait for services
sleep 3
export OLLAMA_HOST=localhost:11434
export GOGENTS_MODEL=arctic
export REDVECTOR_URL=http://localhost:8888
./gogents "$@"
```

**Docker Compose** (optional): define services for `ollama`, `redvector`, and `gogents` (built from the gogents Dockerfile), with `gogents` depending on `ollama` and `redvector` and using the same env vars above.

## Summary

| Component    | Role                          | How gogents uses it                          |
|-------------|-------------------------------|-----------------------------------------------|
| **Ollama**  | Run Arctic (GGUF) locally    | OpenAI-compat base URL; no API key            |
| **RedVector** | RAG vector store            | `REDVECTOR_URL`; optional embedder for queries |
| **gogents** | Agent (tools + loop)          | Same client; backend = Ollama or OpenRouter   |

We do **not** import Ollama’s Go code; we talk to it over HTTP. For Arctic GGUF + RedVector + gogents as one app, run Ollama and RedVector (or wrap them in a script/Compose), set `OLLAMA_HOST` (or `LLM_BASE_URL`) and `GOGENTS_MODEL=arctic`, then run gogents.
