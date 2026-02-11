FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.26-alpine as builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

RUN apk update && apk add -U --no-cache ca-certificates

WORKDIR /app/
ADD go.mod go.sum ./
ADD cmd/ ./cmd/
ADD internal/ ./internal/
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-w -s" -o heimdall ./cmd/gatekeeper/main.go

FROM --platform=${TARGETPLATFORM:-linux/amd64} scratch
WORKDIR /app/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/heimdall /app/heimdall
ADD config.yaml ./
ENV CONFIG_FILE=/app/config.yaml
ENTRYPOINT ["/app/heimdall"]
