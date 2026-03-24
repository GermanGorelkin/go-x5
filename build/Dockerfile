ARG GO_VERSION=1.24.0

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder
ARG CMD
WORKDIR /src

RUN apk add --no-cache git ca-certificates && update-ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/app ./cmd/${CMD}


FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/app /app

ENTRYPOINT ["/app"]
