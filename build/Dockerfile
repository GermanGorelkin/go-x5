FROM golang:alpine as builder
ARG CMD
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates
COPY bin/${CMD}/linux/${CMD} /go/bin/app


FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/app /go/bin/app

ENTRYPOINT ["/go/bin/app"]