#!/usr/bin/env python3
"""
Repair a malformed ~/.gogents/config.json (split lines, control chars in strings).
Preserves existing string fields when possible and sets --model (default: Nemotron free).

  python3 scripts/repair-gogents-config.py -c ~/.gogents/config.json
"""
from __future__ import annotations

import argparse
import json
import pathlib
import re
import sys

# OpenRouter free model ID (see https://openrouter.ai/models?q=free)
DEFAULT_MODEL = "nvidia/nemotron-3-super-120b-a12b:free"

STRING_KEYS = (
    "openrouter_api_key",
    "openrouter_url",
    "model",
    "workspace",
    "instructions",
    "redvector_url",
    "embed_api_url",
    "embed_api_key",
    "embed_model",
    "ollama_host",
    "llm_base_url",
    "serve_addr",
    "serve_api_key",
    "serve_tls_cert",
    "serve_tls_key",
    "serve_domain",
    "serve_acme_email",
)


def extract_string_value(text: str, key: str) -> str | None:
    """Value of "key": "..." with newlines allowed inside the string."""
    m = re.search(r'"' + re.escape(key) + r'"\s*:\s*"', text)
    if not m:
        return None
    i = m.end()
    buf: list[str] = []
    while i < len(text):
        if text[i] == "\\" and i + 1 < len(text):
            buf.append(text[i : i + 2])
            i += 2
            continue
        if text[i] == '"':
            raw = "".join(buf)
            return "".join(c for c in raw if ord(c) >= 32)
        buf.append(text[i])
        i += 1
    return None


def parse_loose(text: str) -> dict:
    cfg: dict = {}
    for k in STRING_KEYS:
        v = extract_string_value(text, k)
        if v is not None and v != "":
            cfg[k] = v

    m = re.search(r'"max_iterations"\s*:\s*(\d+)', text)
    if m:
        cfg["max_iterations"] = int(m.group(1))
    m = re.search(r'"max_tokens"\s*:\s*(\d+)', text)
    if m:
        cfg["max_tokens"] = int(m.group(1))
    m = re.search(r'"temperature"\s*:\s*([\d.]+)', text)
    if m:
        cfg["temperature"] = float(m.group(1))

    if not cfg.get("openrouter_api_key"):
        raise ValueError("could not extract openrouter_api_key from file")
    return cfg


def main() -> None:
    ap = argparse.ArgumentParser(description="Repair gogents config.json and set model.")
    ap.add_argument(
        "-c",
        "--config",
        type=pathlib.Path,
        default=pathlib.Path.home() / ".gogents" / "config.json",
        help="Path to config.json",
    )
    ap.add_argument(
        "-m",
        "--model",
        default=DEFAULT_MODEL,
        help=f"Model ID (default: {DEFAULT_MODEL})",
    )
    args = ap.parse_args()

    raw = args.config.read_text(encoding="utf-8", errors="replace")

    try:
        cfg = json.loads(raw)
    except json.JSONDecodeError:
        cfg = parse_loose(raw)

    cfg["model"] = args.model

    out = json.dumps(cfg, indent=2) + "\n"
    json.loads(out)  # sanity check
    args.config.write_text(out, encoding="utf-8")
    print(f"Wrote {args.config} (model={args.model!r})", file=sys.stderr)


if __name__ == "__main__":
    main()
