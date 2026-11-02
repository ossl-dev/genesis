ARG GO_IMAGE=golang:1.23
ARG BASE_IMAGE=debian:bookworm-slim
FROM ${GO_IMAGE} AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY packages ./packages
COPY scripts/smoke ./scripts/smoke
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /genesis-smoke ./scripts/smoke

FROM ${BASE_IMAGE}
COPY --from=builder /genesis-smoke /usr/local/bin/genesis-smoke
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
# Maintain the disposable Arch fixture before testing Genesis package operations.
RUN if command -v pacman >/dev/null 2>&1; then pacman -Syu --noconfirm; fi
WORKDIR /work
ENV GENESIS_DISPOSABLE_HOST=1
ENTRYPOINT ["/usr/local/bin/genesis-smoke"]
CMD ["packages"]
