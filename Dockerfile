# API image. Skills and the server binary live in the image;
# workspace and logs are mounted volumes.

FROM golang:1.25-bookworm AS build

WORKDIR /src
ARG GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 10001 --create-home eino \
    && mkdir -p /data/workspace /data/logs /app \
    && chown -R eino:eino /data /app

WORKDIR /app
COPY --from=build /out/server /app/server
COPY skills /app/skills
USER eino

EXPOSE 8180
ENTRYPOINT ["/app/server"]
