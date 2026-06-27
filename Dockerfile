# syntax=docker/dockerfile:1.7

FROM node:24-bookworm-slim AS frontend-builder

WORKDIR /src/web/frontend

COPY web/frontend/package.json web/frontend/pnpm-lock.yaml web/frontend/pnpm-workspace.yaml ./
RUN corepack enable \
  && corepack prepare pnpm@11.0.9 --activate \
  && pnpm install --frozen-lockfile

COPY web/frontend ./
RUN pnpm run build

FROM golang:1.26-bookworm AS go-builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY --from=frontend-builder /src/web/frontend/dist ./web/frontend/dist

RUN CGO_ENABLED=0 GOOS=linux go build \
  -trimpath \
  -ldflags="-s -w" \
  -o /out/hi \
  ./cmd/hi

FROM debian:bookworm-slim AS runtime

RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates curl tzdata \
  && rm -rf /var/lib/apt/lists/* \
  && groupadd --system hi \
  && useradd --system --gid hi --home-dir /app --shell /usr/sbin/nologin hi

WORKDIR /app

COPY --from=go-builder /out/hi /usr/local/bin/hi
COPY --from=frontend-builder /src/web/frontend/dist /app/web/frontend/dist

ENV HTTP_ADDR=0.0.0.0:8080
ENV WEB_FRONTEND_DIST=/app/web/frontend/dist

USER hi:hi

EXPOSE 8080

ENTRYPOINT ["hi"]
CMD ["api"]
