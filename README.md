# 👺 Goblin

Use Codex app with any OpenAI-compatible provider.

Codex app을 OpenAI compatible API로 사용할 수 있게 만드는 어댑터 서버입니다.

## 왜 이 프로젝트를 만드나요?

Codex 앱은 올인원 데스크톱 AI 개발환경이 되어가고 있습니다.
물론 다른 다양한 TUI 또는 데스크톱 AI 개발환경도 구현되고 있지만,
현재로써는 Codex 앱 만큼의 안정적인 사용성을 제공하는 앱이 드뭅니다.

Codex CLI/TUI는 커스텀 제공자를 지원합니다.
Codex CLI가 사용하는 OpenAI Responses API를 구현하면 커스텀 제공자를 사용할 수 있습니다.

반면에 Codex 앱은 현재 OpenAI의 pay-as-you-go 또는 구독을 사용할 때만 원활하게 동작합니다.
Codex 앱은 Codex CLI (app-server) 위에 빌드됐지만,
타이틀 생성과 같은 앱 수준의 요청은 Codex CLI에서 구분할 수 없습니다.

그래서 오픈소스인 Codex CLI의 Responses API 요청 처리를 구현할 뿐만 아니라,
Codex 앱을 역공학해 타이틀 생성과 같은 요청을 인지하고 처리할 수 있도록 합니다.

## 개발 환경 설정

```bash
# Install golangci-lint
curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.12.2
```

## Attribution

이 프로젝트는 [openai/codex](https://github.com/openai/codex) (Apache-2.0)의 파일들을 포함하고 있습니다.
각 파일의 원본 출처는 Go 소스 코드 내 `//go:embed` 위 주석에서 확인할 수 있습니다.
