# API image. Skills and the server binary live in the image;
# workspace and logs are mounted volumes.

FROM golang:1.26-alpine AS builder

WORKDIR /src
ARG GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}

COPY . .
RUN go mod vendor
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 10001 --create-home eino \
    && mkdir -p /data/workspace /data/logs /app \
    && chown -R eino:eino /data /app

WORKDIR /app
COPY --from=builder /out/server /app/server
COPY skills /app/skills
# Prefer repo .env; if missing, use .env.example as .env
COPY .env* /tmp/
RUN if [ -f /tmp/.env ]; then \
      cp /tmp/.env /app/.env; \
    else \
      cp /tmp/.env.example /app/.env; \
    fi \
    && rm -rf /tmp/.env /tmp/.env.example \
    && chown eino:eino /app/.env
USER eino

EXPOSE 8180
ENTRYPOINT ["/app/server"]
