# go-x5

[![CI](https://github.com/germangorelkin/go-x5/actions/workflows/tests.yml/badge.svg)](https://github.com/germangorelkin/go-x5/actions/workflows/tests.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/germangorelkin/go-x5)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Go client libraries and CLI tools for the X5 Group Logistics and Insights APIs. Automate report generation, status polling, downloading, and product catalog exports.

## Table of Contents

- [Features](#features)
- [Project Structure](#project-structure)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
  - [logistics](#logistics)
  - [logistics-reload](#logistics-reload)
  - [insights](#insights)
  - [insights-products](#insights-products)
- [Building](#building)
  - [Cross-platform binaries](#cross-platform-binaries)
  - [Docker](#docker)
- [Testing](#testing)
- [Contributing](#contributing)
- [License](#license)

## Features

- **Logistics API client** — authenticate, create reports (sales, inventory, movement, etc.), poll status, and download report parts.
- **Insights API client** — Keycloak + internal JWT authentication, fetch report parameters/dictionaries, build and run Trends Analysis reports, download exports.
- **Product catalog export** — download the full product catalog as an Excel file.
- **Concurrent report execution** — fan-out goroutines with semaphore limiting for batch Insights reports.
- **Cross-platform builds** — single Makefile produces Linux, macOS, and Windows binaries.
- **Minimal Docker images** — multi-stage build targeting `scratch` for any command.
- **CI** — GitHub Actions with golangci-lint and tests across Ubuntu, macOS, and Windows.

## Project Structure

```
go-x5/
├── cmd/
│   ├── logistics/           # Download logistics reports (auto-auth)
│   ├── logistics-reload/    # Download logistics reports (manual auth)
│   ├── insights/            # Batch trends-analysis reports
│   └── insights-products/   # Export product catalog
├── logistics/               # Logistics API client library
│   ├── auth.go              # Authentication service
│   ├── client.go            # HTTP client with auto-auth interceptor
│   ├── report.go            # Report create / status / download
│   └── *_test.go
├── insights/                # Insights API client library
│   ├── auth.go              # Keycloak + internal token exchange
│   ├── client.go            # HTTP client setup
│   ├── parameters.go        # Report dictionaries & product catalog
│   ├── report.go            # Trends Analysis create / poll / download
│   └── *_test.go
├── internal/
│   ├── xconfig/             # Environment variable helpers
│   └── xlog/                # Structured logging (zap) & HTTP interceptor
├── build/
│   └── Dockerfile           # Multi-stage build (golang → scratch)
├── examples/                # Sample requests & exported reports (local only)
├── .github/workflows/
│   └── tests.yml            # CI pipeline
├── Makefile
├── go.mod
└── go.sum
```

## Prerequisites

- **Go 1.24+** (see [`go.mod`](go.mod))
- **Docker** (optional, for container builds)
- **Make** (optional, for convenience targets)

## Installation

```
git clone https://github.com/germangorelkin/go-x5.git
cd go-x5
go mod download
```

Verify everything compiles:

```
go build ./...
```

## Configuration

All commands are configured exclusively through **environment variables**. Never hardcode credentials or tokens.

### Logistics commands

| Variable | Description | Default |
|---|---|---|
| `INSTANCE` | Logistics API base URL | — |
| `LOGIN` | Account login | — |
| `PASSWORD` | Account password | — |
| `AUTO_AUTH` | Enable automatic token refresh (`logistics` only) | `false` |
| `SALES_CHANNEL` | Sales channel filter (`TS5`, `TSX`, `TSK`, `ALL`) | — |
| `TYPE_REPORT` | Report type (`SALES`, `INVENTORY`, `MOVEMENT`, …) | — |
| `START_DATE` | Report start date | today − 4 days |
| `FINISH_DATE` | Report end date | today |
| `ARCHIVE` | Request archived data | `false` |
| `OUT_DIR` | Output directory for downloaded files | `reports` |
| `WAITE_REPORT_STATUS_DELAY_SEC` | Seconds between status polls | `10` |
| `WAITE_REPORT_STATUS_ATTEMPT` | Maximum number of poll attempts | `10` |
| `VERBOSE` | Enable verbose (debug) logging | `false` |

### Insights commands

| Variable | Description | Default |
|---|---|---|
| `KC_URL` | Keycloak server URL | — |
| `KC_RELM` | Keycloak realm | — |
| `CLIENT_ID` | Keycloak client ID | — |
| `LOGIN` | Account login | — |
| `PASSWORD` | Account password | — |
| `API_URL` | Insights API base URL | — |
| `GROUP_REQUEST` | Request grouping strategy (`1` or `2`) | `1` |
| `START_DATE` / `FINISH_DATE` | Monthly date range | previous month → now |
| `START_WEEK_DATE` / `FINISH_WEEK_DATE` | Weekly date range | auto-aligned to Mon–Sun |
| `OUT_DIR` | Output directory for downloaded files | `reports` |
| `WAITE_REPORT_STATUS_DELAY_SEC` | Seconds between status polls | `60` |
| `WAITE_REPORT_STATUS_ATTEMPT` | Maximum number of poll attempts | `10` |
| `VERBOSE` | Enable verbose (debug) logging | `false` |

## Usage

### logistics

Create a logistics report, poll until ready, and download all parts:

```
export INSTANCE=https://api.example.com
export LOGIN=user
export PASSWORD=secret
export AUTO_AUTH=true
export TYPE_REPORT=SALES
export SALES_CHANNEL=ALL
export OUT_DIR=./reports

go run ./cmd/logistics
```

### logistics-reload

Same workflow as `logistics` but performs **manual authentication** (explicit token injection). Useful when you need fine-grained control over the auth step:

```
go run ./cmd/logistics-reload
```

### insights

Batch-generate Trends Analysis reports with concurrent execution:

```
export KC_URL=https://keycloak.example.com
export KC_RELM=x5
export CLIENT_ID=my-client
export LOGIN=user
export PASSWORD=secret
export API_URL=https://insights-api.example.com
export OUT_DIR=./reports

go run ./cmd/insights
```

### insights-products

Export the full product catalog as an Excel file:

```
go run ./cmd/insights-products
```

## Building

### Cross-platform binaries

The Makefile builds Linux (`amd64`), macOS (`amd64`), and Windows (`amd64`) binaries in one step:

```
make build cmd=logistics
make build cmd=insights
```

Binaries are written to `bin/<cmd>/{linux,darwin,win}/`.

### Docker

Build a minimal container image for any command:

```
make docker cmd=logistics
make docker cmd=insights
```

The Dockerfile uses a two-stage build (`golang:1.24-alpine` → `scratch`) so the final image contains only the static binary and CA certificates.

### Clean

Remove all build artifacts:

```
make clean
```

## Testing

Run the full test suite:

```
go test ./...
```

Verbose output (matches CI):

```
go test ./... -v
```

Tests use the standard `testing` package alongside `net/http/httptest` for HTTP stubbing and [`testify/assert`](https://github.com/stretchr/testify) for assertions. Test files live next to the source code they cover (e.g., `logistics/auth_test.go`).

### Naming convention

```
Test<Type>_<Method>_<Case>
```

For example: `TestReportService_Create_Success`, `TestClient_AuthInterceptor_Retry`.

## Contributing

1. Fork the repository and create a feature branch.
2. Follow standard Go formatting (`gofmt`) and pass `golangci-lint`.
3. Add or update tests for any client behavior changes.
4. Run `go test ./...` and `go build ./...` before submitting.
5. Use short, imperative commit messages (e.g., `add weekly report support`).
6. Open a pull request with a concise summary and link any related issues.

## License

This project is licensed under the [MIT License](LICENSE).
