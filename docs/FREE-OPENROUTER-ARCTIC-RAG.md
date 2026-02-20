# Free OpenRouter (think + tasks) + Snowflake Arctic (RAG embeddings)

Use **OpenRouter free models** for the agent (chat + tool use) and **Snowflake Arctic** only for **embeddings** in RAG. RedVector stores vectors; the LLM never sees Arctic — it only sees retrieved text.

## Split

| Role | Service | What it does |
|------|---------|----------------|
| **Think & act** | OpenRouter (free) | Chat completions + tool calls (read_file, run_shell, rag_search, etc.) |
| **RAG embeddings** | Snowflake Arctic (local) | Embed query text for `rag_search`; run via Ollama |
| **Vector store** | RedVector | Store and search vectors; Qdrant-compat REST API |

You need an **OpenRouter API key** (free account); no key for local Arctic embeddings (Ollama).

---

## 1. OpenRouter (free) for the agent

- Sign up: [openrouter.ai](https://openrouter.ai) and get an API key.
- Set:
  ```bash
  export OPENROUTER_API_KEY=sk-or-...
  export GOGENTS_MODEL=openrouter/free
  ```
- `openrouter/free` picks a free model for each request. Or set a specific free model, e.g.:
  - `arcee-ai/trinity-large-preview`
  - `stepfun/step-3.5-flash`
  - `z-ai/glm-4.5-air`
  - `deepseek/deepseek-r1-0528`
  - `meta-llama/llama-3.3-70b-instruct`
  - See [OpenRouter free models](https://openrouter.ai/models?max_price=0) for the full list.

gogents uses this for all **chat** and **tool** calls (including when it calls `rag_search` and then reasons over the results).

---

## 2. Snowflake Arctic for RAG embeddings only

Arctic is used **only** to turn the user’s natural-language query into a vector for RedVector. The model that “thinks” is the OpenRouter free model.

### Option A: Ollama + Arctic embed GGUF (recommended, local)

1. **Install Ollama** and start it (e.g. `ollama serve`).

2. **Add Snowflake Arctic embed** (e.g. [snowflake-arctic-embed-l](https://huggingface.co/Snowflake/snowflake-arctic-embed-l) in GGUF form, e.g. [bcastle/snowflake-arctic-embed-l-Q8_0-GGUF](https://huggingface.co/bcastle/snowflake-arctic-embed-l-Q8_0-GGUF)):
   - Download the GGUF file, then create an Ollama model:
     ```bash
     echo 'FROM /path/to/snowflake-arctic-embed-l-q8_0.gguf' > Modelfile.embed
     ollama create arctic-embed -f Modelfile.embed
     ```
   - Or use a Hugging Face–based flow if your Ollama supports it.

3. **Point gogents at Ollama for embeddings only** (not for chat):
   ```bash
   export EMBED_API_URL=http://localhost:11434/v1
   export EMBED_MODEL=arctic-embed
   ```
   Leave `EMBED_API_KEY` unset; Ollama doesn’t need it.

gogents will call `POST http://localhost:11434/v1/embeddings` only when the agent uses the **rag_search** tool. The OpenRouter model is still used for all reasoning and tool choice.

### Option B: OpenRouter for embeddings (cloud)

If you prefer not to run Ollama, you can use OpenRouter for embeddings too (different model, not Arctic):

```bash
export EMBED_API_URL=https://openrouter.ai/api/v1
export EMBED_API_KEY=sk-or-...
export EMBED_MODEL=openai/text-embedding-3-small
```

That uses OpenRouter’s embedding endpoint; for “Arctic only for RAG” you’d use Option A.

---

## 3. RedVector for RAG storage

Run [RedVector](https://github.com/rafaelescrich/redvector) (e.g. REST on port 8888). Create a collection whose **vector size matches Arctic’s embedding size** (e.g. 1024 for snowflake-arctic-embed-l). Index your documents with that same embed model (e.g. via Ollama + Arctic).

```bash
export REDVECTOR_URL=http://localhost:8888
```

---

## 4. Full env example (free OpenRouter + local Arctic RAG)

```bash
# Agent: OpenRouter free
export OPENROUTER_API_KEY=sk-or-...
export GOGENTS_MODEL=openrouter/free

# RAG: RedVector + Arctic embeddings via Ollama
export REDVECTOR_URL=http://localhost:8888
export EMBED_API_URL=http://localhost:11434/v1
export EMBED_MODEL=arctic-embed
# no EMBED_API_KEY for local Ollama
```

Then run:

```bash
./gogents
```

Or one-shot:

```bash
./gogents "Search my docs for X and summarize."
```

The agent (OpenRouter free model) will call `rag_search` when needed; gogents will embed the query with Arctic (Ollama) and search RedVector, then pass the retrieved text back to the OpenRouter model for answering.

---

## 5. Config file example

`~/.gogents/config.json`:

```json
{
  "openrouter_api_key": "sk-or-...",
  "model": "openrouter/free",
  "redvector_url": "http://localhost:8888",
  "embed_api_url": "http://localhost:11434/v1",
  "embed_model": "arctic-embed"
}
```

Do **not** set `ollama_host` or `openrouter_url` to localhost if you want the **LLM** to be OpenRouter; those would switch the main agent to Ollama. Only `embed_api_url` should point at Ollama for Arctic embeddings.

---

## Summary

- **Think & do tasks**: OpenRouter with `openrouter/free` (or another free model). Requires `OPENROUTER_API_KEY`.
- **RAG embeddings**: Snowflake Arctic via Ollama (`EMBED_API_URL=http://localhost:11434/v1`, `EMBED_MODEL=arctic-embed`). No API key.
- **Vector store**: RedVector (`REDVECTOR_URL`). Index docs with the same Arctic embed model and dimension.
