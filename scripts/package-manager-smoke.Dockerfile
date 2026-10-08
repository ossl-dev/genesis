ARG BUN_IMAGE=oven/bun:1.3.2
ARG BASE_IMAGE=debian:bookworm-slim
FROM ${BUN_IMAGE} AS runtime
FROM ${BASE_IMAGE}
COPY --from=runtime /usr/local/bin/bun /usr/local/bin/bun
RUN if [ -f /etc/alpine-release ]; then apk add --no-cache libstdc++ libgcc; fi
RUN if command -v pacman >/dev/null 2>&1; then pacman -Syu --noconfirm; fi
WORKDIR /work
ENV GENESIS_DISPOSABLE_HOST=1
CMD ["bun", "scripts/package-manager-smoke.ts"]
