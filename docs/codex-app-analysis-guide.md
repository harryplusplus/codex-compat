# Codex Desktop App: Analysis Guide

## App Bundle Path

```
/Applications/Codex.app/Contents/
├── Info.plist
├── MacOS/
│   └── Codex                      # Electron executable
├── Resources/
│   ├── app.asar                   # ★ 메인 애플리케이션 코드 (Electron 번들)
│   ├── app.asar.unpacked/         # 네이티브 모듈 등
│   ├── electron.icns
│   └── codex                      # Rust CLI 바이너리 (Mach-O 64-bit)
├── Frameworks/
│   ├── Electron Framework.framework
│   ├── Codex Helper.app           # Electron helper (GPU, Renderer, Plugin)
│   ├── Sparkle.framework          # 자동 업데이트
│   └── Squirrel.framework
└── _CodeSignature/
```

### Key Paths

| Component | Path |
|-----------|------|
| Electron App | `/Applications/Codex.app/Contents/Resources/app.asar` |
| Rust CLI | `/Applications/Codex.app/Contents/Resources/codex` |
| App version | `Info.plist` → `CFBundleShortVersionString` |

---

## Extracting `app.asar`

### 1. `asar` 도구로 추출

```bash
# @electron/asar 패키지 설치
npm install -g @electron/asar

# 추출
asar extract /Applications/Codex.app/Contents/Resources/app.asar /tmp/codex-app

# 또는 npx로 한 번에
npx @electron/asar extract /Applications/Codex.app/Contents/Resources/app.asar /tmp/codex-app
```

### 2. 추출된 디렉토리 구조

```
/tmp/codex-app/
├── package.json                   # 앱 정보 (name, version 등)
├── .vite/build/
│   ├── bootstrap.js               # Electron entry point
│   ├── main-BwqrdVu3.js           # ★ Main process 코드 (IPC handlers)
│   ├── app-session-O7kcZj7R.js    # ★ App session 유틸리티
│   ├── workspace-root-drop-handler-Ds_5iOm2.js  # ★ 타이틀 생성 핵심 로직
│   ├── preload.js                 # preload script
│   ├── sandbox-preload.js         # sandbox preload
│   └── worker.js                  # worker thread
├── webview/
│   ├── index.html                 # renderer entry
│   ├── assets/
│   │   ├── composer-DXaiOlFj.js   # ★ 타이틀 UI 관련
│   │   ├── local-conversation-thread-BX7YNcUw.js  # 스레드 UI
│   │   ├── app-server-manager-signals-BEaGjuc8.js
│   │   ├── runtime.worker-*.js    # webview worker
│   │   ├── model-queries-*.js     # 모델 관련
│   │   └── ... (수백 개의 code-split chunks)
│   └── apps/                      # IDE 아이콘 등 assets
├── skills/                        # (빈 디렉토리)
└── node_modules/
    ├── better-sqlite3
    ├── node-pty
    └── ... (네이티브 모듈만 포함)
```

---

## Analysis Methods

### 검색 (ripgrep)

모든 JS 파일은 **minified + code-split** 되어 있어 한 줄로 압축되어 있습니다.
그래도 ripgrep으로 패턴 검색은 가능합니다.

```bash
# 특정 패턴 검색
rg "generate-thread-title" /tmp/codex-app/ --glob '*.js'
rg "gpt-5.4-mini" /tmp/codex-app/ --glob '*.js'
rg "set-thread-title" /tmp/codex-app/ --glob '*.js'
rg "small_model\|getSmallModel" /tmp/codex-app/ --glob '*.js'

# 특정 파일에서만 검색
rg "generate.*title" /tmp/codex-app/.vite/build/main-BwqrdVu3.js

# 빈도 확인
rg -c "title" /tmp/codex-app/ --glob '*.js' -g '!node_modules/' | sort -t: -k2 -rn | head -10
```

### Python으로 미니파이드 코드 분석

minified 파일은 한 줄이 수십만 ~ 수백만 자이므로, ripgrep으로 문맥을 보기 어렵습니다.
Python으로 특정 위치 주변을 추출하는 것이 효과적입니다.

```python
import re

with open('/tmp/codex-app/.vite/build/main-BwqrdVu3.js', 'r') as f:
    content = f.read()

# 특정 문자열 주변 문맥 보기
idx = content.find('generate-thread-title')
start = max(0, idx - 500)
end = min(len(content), idx + 1000)
print(content[start:end])

# 함수 정의 찾기
for m in re.finditer(r'async function \w+\(', content):
    print(m.group())
```

### Source map 확인

```bash
# Source map은 번들에 포함되어 있지 않음 (프로덕션 빌드)
find /tmp/codex-app -name '*.map'  # 결과 없음
```

> 프로덕션 빌드이므로 source map이 없습니다. 변수명은 모두 minified 되어
> (a, b, c... → t, n, r, i, o, s, e... 등) 분석이 어렵습니다.
> 함수의 역할은 호출 패턴과 문자열 리터럴로 유추해야 합니다.

### 모듈 간 의존성 추적

`.vite/build/`의 파일들은 CommonJS (`require()`)로 연결됩니다:

```javascript
// main-BwqrdVu3.js 상단
const e = require('./app-session-O7kcZj7R.js')
const t = require('./workspace-root-drop-handler-Ds_5iOm2.js')
// e = app-session module
// t = workspace-root-drop-handler module
```

내보내진 함수는 `Object.defineProperty(exports, 'F', { get: function() { return wk } })`
패턴으로 등록되며, minified 2글자 이름으로 맵핑됩니다:

```
main.js에서 사용:   t.F(...)   → workspace module의 wk()
                   e._(...)   → app-session module의 특정 함수
                   e.Rr(...)  → electron require wrapper
                   e.zt(...)  → host config 체크
                   e.mr(...)  → ?
```

---

## 주요 분석 대상 함수들

| 관심사 | 파일 (`.vite/build/`) | 함수/핸들러 |
|--------|----------------------|-------------|
| **타이틀 생성** | `workspace-root-drop-handler-*.js` | `wk()` / `vk()` / `yk()` |
| **타이틀 생성 IPC** | `main-BwqrdVu3.js` | `"generate-thread-title"` |
| **타이틀 변경 UI** | `webview/assets/composer-*.js` | `"set-thread-title"` event |
| **커밋 메시지 생성** | `workspace-root-drop-handler-*.js` | `"generate-commit-message"` |
| **PR 메시지 생성** | `workspace-root-drop-handler-*.js` | `"generate-pull-request-message"` |
| **모델 쿼리** | `webview/assets/model-queries-*.js` | - |

---

## Version 확인

```bash
# App version
plutil -p /Applications/Codex.app/Contents/Info.plist | grep CFBundleShortVersionString

# package.json
cat /tmp/codex-app/package.json | grep version
```

---

## 주의사항

1. **minified + no source map**: 모든 변수명이 압축되어 있어 가독성이 매우 낮음
2. **code-split**: 파일이 수백 개의 chunk로 나뉘어 있어 특정 기능을 찾기 어려움
3. **업데이트 시 변경됨**: 매 업데이트마다 chunk hash가 바뀌어 파일명이 달라짐
4. **소스 레벨의 TypeScript는 없음**: 모든 코드는 이미 빌드된 JS 번들
5. **네이티브 모듈**: `node_modules/`에는 네이티브 모듈(`better-sqlite3`, `node-pty` 등)만 포함
