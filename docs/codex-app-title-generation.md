# Codex Desktop App: Thread Title Generation

## Overview

Codex Desktop 앱(Electron)은 사용자의 첫 메시지를 바탕으로 **`gpt-5.4-mini`** 모델을 사용해 **자동으로 스레드 타이틀을 생성**합니다. 생성된 타이틀은 Rust 백엔드(`codex-rs`)로 전송되어 저장만 됩니다.

---

## Architecture

```
사용자 첫 메시지
    ↓
Codex Desktop App (Electron renderer)
    ├─ composer-DXaiOlFj.js       ← UI에서 IPC 요청
    ↓
main-BwqrdVu3.js (main process)
    ├─ "generate-thread-title" IPC handler
    └─ bD() 호출
        ↓
workspace-root-drop-handler-Ds_5iOm2.js
    ├─ wk()  ← 실제 타이틀 생성 로직
    ├─ vk()  ← 시스템 프롬프트 생성
    ├─ yk()  ← 응답 후처리
    └─ Kt()  ← LLM API 호출
        ↓
Rust Backend (codex-rs)
    └─ thread_set_name()  ← 그냥 저장만 함
```

### Key Files (inside `app.asar`)

| File | Role |
|------|------|
| `.vite/build/workspace-root-drop-handler-Ds_5iOm2.js` | **핵심**: 프롬프트, 모델 호출, 후처리 |
| `.vite/build/main-BwqrdVu3.js` | Electron main process IPC handler |
| `webview/assets/composer-DXaiOlFj.js` | Renderer-side UI (`set-thread-title` event) |

---

## Model

| Parameter | Value | Source |
|-----------|-------|--------|
| **Model slug** | `gpt-5.4-mini` | `var nn = \`gpt-5.4-mini\`` |
| **Effort** | `low` | `effort: \`low\`` |
| **Timeout** | `30,000ms` | `var bk = 3e4` |
| **Max prompt length** | `2,000 chars` | `var xk = 2e3` (초과시 앞에서 자름) |
| **Web search** | `disabled` | `web_search: "disabled"` |
| **Codex hooks** | `disabled` | `"features.codex_hooks": false` |

### Structured Output Schema

```typescript
// Response schema
var Sk = e.Er({ title: e.Or().min(1).max(36) })
// { title: string }  // 1-36자

// Request schema (wraps response schema)
var Ck = qt(Sk)
```

> 모델 `gpt-5.4-mini`는 `./codex/codex-rs/models-manager/models.json` 에 정의되어 있으며,
> 오픈소스에는 **모델 정의만** 있고 타이틀 생성 로직 자체는 **Codex 앱 번들에만** 있습니다.

---

## System Prompt (`vk()`)

```text
You are a helpful assistant. You will be presented with a user prompt,
and your job is to provide a short title for a task that will be created
from that prompt.

The tasks typically have to do with coding-related tasks, for example
requests for bug fixes or questions about a codebase. The title you
generate will be shown in the UI to represent the prompt.

Generate a concise UI title (up to 36 characters) for this task.

Fill the structured title field with plain text.
```

> 이 프롬프트는 **오픈소스 저장소에는 존재하지 않으며**, Codex 앱 번들에만 있습니다.

---

## Post-processing (`yk()`)

```javascript
function yk(e) {
  let t = (e
    .replace(/\r\n/g, '\n')
    .split('\n')
    .find(line => line.trim().length > 0) ?? ''
  ).trim()

  if (t.length === 0) return null

  t = t.replace(/^title[:\s]+/i, '')           // "title:" prefix 제거
  t = t.replace(/^[`"'\u201c\u201d\u2018\u2019]+/, '')  // 앞 따옴표 제거
  t = t.replace(/[`"'\u201c\u201d\u2018\u2019]+$/, '')  // 뒤 따옴표 제거
  t = t.replace(/\s+/g, ' ').trim()             // 연속 공백 정리
  t = t.replace(/[.?!]+$/, '').trim()            // 끝 문장부호 제거

  if (t.length === 0) return null
  if (t.length > 36) return t.slice(0, 35).trimEnd() + '…'

  return t
}
```

---

## LLM Call (final)

```javascript
await Kt({
  prompt: o,                            // system prompt + user message
  cwd: t,
  model: "gpt-5.4-mini",
  effort: "low",
  serviceTier: n,
  schema: Ck,                           // structured output schema
  config: {
    web_search: "disabled",
    "features.codex_hooks": false
  },
  timeoutMs: 30000,
  client: r,
  responseSchema: Sk,                   // { title: string (1-36) }
  onTokenUsage: e => { s = e }
})
```

---

## Relationship to Open Source (`./codex`)

| Component | Open Source (`./codex`) | App Bundle (`app.asar`) |
|-----------|:----------------------:|:----------------------:|
| `gpt-5.4-mini` model definition | ✅ `models-manager/models.json` | ❌ |
| `gpt-5.4-mini` string reference | ✅ 여러 Rust files | ✅ |
| Title generation **system prompt** | ❌ Not present | ✅ `vk()` |
| Title generation **LLM call logic** | ❌ Not present | ✅ `wk()` |
| Title generation **post-processing** | ❌ Not present | ✅ `yk()` |
| Title **storage** (Rust) | ✅ `thread_set_name()` | ❌ |

---

## Flow Summary

1. 사용자가 첫 메시지를 보냄
2. Electron renderer가 `generate-thread-title` IPC를 main process로 전송
3. Main process가 `workspace-root-drop-handler`의 `wk()` 호출
4. `wk()`:
   - 프롬프트를 2000자로 제한
   - 시스템 프롬프트 + 사용자 메시지 조합
   - `gpt-5.4-mini` (effort: low) 호출 (structured output)
   - 응답에서 title 추출 및 후처리
   - 최대 36자로 정리
5. 결과 타이틀이 Rust 백엔드로 전송되어 저장
