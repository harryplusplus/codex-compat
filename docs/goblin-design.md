# Goblin Design

## Problem

Codex Desktop App is a closed-source Electron application that communicates with OpenAI's proprietary API surface. To use it with any OpenAI-compatible provider, an adapter layer is needed.

### Layers involved

```
Codex Desktop App (Electron)
  ├── App-level features (title generation, etc.)
  │   └── Ht() → startTurn() → app-server (Rust, codex-rs)
  │       └── Responses API → ...
  └── Standard conversation
      └── Responses API → ...
```

### Identified issues

| Problem | Root cause | Impact |
|---------|-----------|--------|
| **Title generation not working** | App sends structured output schema (`text.format`), but non-OpenAI providers either reject `json_schema` in streaming mode or return plain text that app can't parse | Core UX broken |
| **Provider protocol variance** | Each OpenAI-compatible provider supports different subsets: `json_schema` vs `json_object` only, parameter name differences, missing features | Adapter needs per-provider tweaks |
| **Semantic blindness** | Simple protocol translators (like codex-relay) can't distinguish "title generation" from "normal conversation" | Can't apply response transformations contextually |
| **Debugging opacity** | When something fails (empty response, wrong format), user has no idea which layer caused it or what rule fired | Impossible to configure correctly |

### codex-relay lessons

[codex-relay](https://github.com/MetaFARS/codex-relay) is a Rust Responses API ↔ Chat Completions proxy. It revealed several architectural gaps:

1. **App-level semantics invisible** — relay sees `POST /v1/responses` but can't tell "this is a title generation request" vs "this is a coding conversation"
2. **Response_format inflexibility** — Rust types are static; adding json_schema→json_object fallback required code changes
3. **No response transformation** — relay passes through raw model output; can't do "plain text → JSON wrapping" post-processing
4. **Provider differences require code changes** — every provider quirk (crof.ai's streaming+json_schema 500) needs a new release
5. **Rust maintenance burden** — user prefers Go for this kind of integration work

---

## Solution: Goblin

A **semantic API adapter** in Go that sits between Codex Desktop App (or Codex CLI) and any OpenAI-compatible provider.

### Architecture

```
Codex App / Codex CLI
  │ Responses API (or Chat Completions)
  ▼
┌─────────────────────────────────────┐
│         Goblin                      │
│                                     │
│  ┌─────────────────────────────┐    │
│  │ Request Classifier          │    │
│  │  - pattern matching         │    │
│  │  - identifies domain        │    │
│  │    (title, conversation,    │    │
│  │     commit message, etc.)   │    │
│  └──────────┬──────────────────┘    │
│             ▼                       │
│  ┌─────────────────────────────┐    │
│  │ Semantic API Mapper         │    │
│  │  - Responses ↔ Chat Comp.   │    │
│  │  - provider-specific rules  │    │
│  │  - response_format handling │    │
│  └──────────┬──────────────────┘    │
│             ▼                       │
│  ┌─────────────────────────────┐    │
│  │ Response Transformer        │    │
│  │  - JSON parse attempt       │    │
│  │  - plain text → JSON wrap   │    │
│  │  - domain-specific fixes    │    │
│  └──────────┬──────────────────┘    │
│             ▼                       │
│  ┌─────────────────────────────┐    │
│  │ Logger / Debug Tracer      │    │
│  │  - which rules fired       │    │
│  │  - request/response diff   │    │
│  │  - config hints            │    │
│  └─────────────────────────────┘    │
└─────────────────────────────────────┘
  │
  ▼
OpenAI-compatible provider
  (DeepSeek, Kimi, Qwen, Mistral, Groq, xAI, OpenRouter, crof.ai, ...)
```

---

## Core Features

### 1. Request Classification

Goblin inspects incoming requests to determine their **domain**:

| Domain | Detection pattern | Example |
|--------|------------------|---------|
| `title_generation` | System prompt contains `"Generate a concise UI title"`, `"Fill the structured title field"`, or `response_format` includes `{title: string}` schema | First message title auto-generation |
| `commit_message` | System prompt or tool descriptions reference `"commit message"` generation | `"generate-commit-message"` IPC |
| `conversation` | None of the above — standard chat | Normal coding conversation |
| `function_call` | Request includes tool definitions for exec/shell/agent tools | Agent delegation |

Classifier is **configurable via regex patterns** so users can add new domains without code changes.

```yaml
# config.yaml
classifier:
  rules:
    - domain: title_generation
      match_prompt: "Generate a concise UI title"
      match_schema: "title.*string"
```

### 2. Semantic API Mapping

Maps Responses API requests to provider-specific Chat Completions format.

**Standard mapping** (Responses API → OpenAI Chat Completions):
- Messages: `input_text` → `text`, `input_image` → `image_url`
- Tools: `function` → nested `function`, `namespace` → splice children, drop proprietary tools
- Roles: `developer` → `system`, `function_call_output` → `tool`
- Model: name mapping via `model_map` config

**Provider-specific overrides**:

```yaml
# config.yaml
providers:
  crof:
    response_format: json_object  # json_schema not supported in streaming
    strip_schema_fields: ["$schema"]
    supported_params:
      - max_tokens
      - temperature  
      - response_format
      - tools
  
  openai:
    response_format: json_schema  # full support
  
  deepseek:
    response_format: json_object
  
  ollama:
    response_format: none  # no format param at all
```

### 3. Domain-Aware Response Transformation

After the upstream responds, Goblin transforms the output based on the detected domain.

**Title generation flow**:

```
Upstream returns:   "안녕" (plain text)
                          ↓
Goblin detects: domain = title_generation
                  ↓
Transformation rule: domain.title_generation
  try JSON.parse(response)
    → success: validate against expected schema
    → fail: wrap as {"title": response.trim()}
                  ↓
App receives:     {"title": "안녕"}  ← JSON.parse + safeParse 통과!
```

```yaml
# config.yaml
transforms:
  - domain: title_generation
    match: "title"
    on_parse_fail: wrap_json
    wrap_key: "title"
    max_length: 36
    fallback: "New conversation"
```

**Streaming transformation** (for title generation with streaming):

```
SSE events: ... content: "안" → content: "녕" → [DONE]
                           ↓
Goblin buffers text content until [DONE]
                           ↓
Transforms complete text → {"title": "안녕"}
                           ↓
Emits as structured SSE event
```

### 4. Debug Logging & Config Tuning

Every decision is logged with enough context for the user to tune config:

```
[CLASSIFIER] domain=title_generation rule="match_prompt:'Generate a concise UI title'" confidence=0.95
[MAPPER] response_format=json_object reason="provider=crof does not support json_schema+streaming"
[TRANSFORM] action=wrap_json key=title original="안녕" transformed='{"title":"안녕"}'
[RECOMMEND] "If title format is wrong, set transforms.domain.title_generation.wrap_key"
```

Log level controls:
- `error`: Only failures
- `warn`: Unexpected but handled cases
- `info`: High-level decisions  
- `debug`: Full request/response dump
- `trace`: SSE event-level detail

---

## Implementation Plan

### Phase 1: Core Proxy (MVP)

- [ ] Responses API → Chat Completions translation (port from codex-relay)
- [ ] Request classification (simple pattern matching)
- [ ] Response transformation (JSON parse attempt → wrap)
- [ ] Streaming support
- [ ] Model name mapping (`CODEX_RELAY_MODEL_MAP` equivalent)
- [ ] YAML/TOML config file

### Phase 2: Provider Profiles

- [ ] Built-in provider profiles (OpenAI, DeepSeek, Kimi, Qwen, Mistral, Groq, xAI, OpenRouter, crof.ai, Ollama, etc.)
- [ ] Provider-specific parameter filtering
- [ ] Response_format strategy per provider
- [ ] Tool conversion strategies

### Phase 3: Debugging & Observability

- [ ] Structured debug logging with config recommendations
- [ ] Prometheus metrics (request count, latency, errors per domain)
- [ ] Health check endpoint (`/health`)
- [ ] Model catalog proxy (`GET /v1/models`)

### Phase 4: Advanced Features

- [ ] Domain-specific prompt injection (e.g., append "Return JSON" to system prompt for title gen)
- [ ] Rate limiting / queue management
- [ ] Multi-upstream load balancing
- [ ] Caching (for model list, etc.)

---

## Config File Design

```yaml
# ~/.codex/goblin.yaml
server:
  port: 4444
  log_level: info       # error | warn | info | debug | trace

upstream:
  base_url: https://api.deepseek.com/v1
  api_key_env: DEEPSEEK_API_KEY
  model_map:
    gpt-5.4-mini: deepseek-chat
    gpt-5.5: deepseek-reasoner

classifier:
  rules:
    - domain: title_generation
      match_prompt: "(?i)generate.*(?:title|concise UI)"
      match_schema: "title.*string"
    - domain: commit_message
      match_prompt: "(?i)generate.*commit message"

provider:
  name: deepseek                    # built-in profile or custom
  response_format: json_object      # json_schema | json_object | none | passthrough
  strip_fields: ["$schema"]         # JSON schema fields to remove
  supported_params:                 # whitelist of Chat Completions params
    - model
    - messages
    - max_tokens
    - temperature
    - stream
    - tools
    - response_format

transforms:
  - domain: title_generation
    on_parse_fail: wrap_json         # wrap | strip | error | passthrough
    wrap_key: title
    max_length: 36
    fallback: "New conversation"
  - domain: commit_message
    on_parse_fail: passthrough
```

---

## Comparison

| Feature | codex-relay (Rust) | Goblin (Go) |
|---------|-------------------|-------------|
| Protocol translation | ✅ Responses → Chat | ✅ Responses → Chat |
| Domain awareness | ❌ | ✅ Pattern-based classifier |
| Response transformation | ❌ Can't modify output | ✅ JSON wrap / strip / fix |
| Provider profiles | ❌ Code changes needed | ✅ Config-driven |
| Debug logging | ❌ Minimal | ✅ Rule tracing + config hints |
| Streaming transform | ❌ Raw pass-through | ✅ Buffer + transform |
| Language | Rust | Go (user preference) |
