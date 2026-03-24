# Repository Guidelines

## Project Structure & Module Organization
`go-x5` is a small Go module split by API domain. Library code lives in `logistics/` and `insights/`. CLI entrypoints live in `cmd/<name>/main.go`; current commands are `logistics`, `logistics-reload`, `insights`, and `insights-products`. Cross-platform build artifacts are written to `bin/`. Docker packaging is in `build/Dockerfile`. Sample requests and exported report files live in `examples/`. GitHub Actions CI is defined in `.github/workflows/tests.yml`.

## Build, Test, and Development Commands
Use the Go toolchain declared in `go.mod` (`go 1.22`, toolchain `go1.23.0`).

- `go test ./...` runs the full test suite.
- `go test ./... -v` matches the CI test command.
- `go build ./...` checks that all packages and commands compile.
- `make build cmd=logistics` runs `go mod tidy` and builds `bin/logistics/{linux,darwin,win}`.
- `make docker cmd=insights` builds the command binaries, then creates the image from `build/Dockerfile`.
- `go run ./cmd/logistics` is the fastest way to exercise a command locally with env vars such as `INSTANCE`, `LOGIN`, `PASSWORD`, and `OUT_DIR`.

## Coding Style & Naming Conventions
Follow standard Go formatting with `gofmt` and keep imports normalized. CI also runs `golangci-lint`, so favor idiomatic Go over custom style. Use `PascalCase` for exported names, `camelCase` for internal helpers, and lowercase file names such as `client.go` or `report_test.go`. Keep packages focused on one API surface and prefer adding new command logic under `cmd/<tool>/`.

## Testing Guidelines
Tests use the standard `testing` package with `httptest` and `testify/assert`. Existing tests live beside the code, for example `logistics/auth_test.go` and `logistics/report_test.go`. Name tests like `Test<Type>_<Method>_<Case>` and cover request paths, payloads, and response parsing. Add or update package-level tests for client behavior changes before opening a PR.

## Commit & Pull Request Guidelines
Recent history uses short, imperative commit messages such as `change auth` and `add GroupRequest`. Keep commits focused and descriptive; one logical change per commit is preferred. PRs should include a concise summary, linked issue if applicable, and the commands you ran (`go test ./...`, `go build ./...`). Include sample output only when CLI behavior changes.

## Security & Configuration Tips
All commands are configured through environment variables; do not hardcode credentials, tokens, or customer report data. Treat files under `examples/` as local reference material and avoid committing refreshed exports or secrets.
