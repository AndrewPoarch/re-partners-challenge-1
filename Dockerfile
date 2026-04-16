# syntax=docker/dockerfile:1.6
# Multi-stage: static Go binary (CGO off, pure Go SQLite), then minimal distroless runtime.

FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static binary for linux; trim symbols for smaller image.
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -trimpath -ldflags="-s -w" -o /out/pack-calculator ./cmd/server

RUN mkdir -p /out/data

FROM gcr.io/distroless/static-debian12

COPY --from=build /out/pack-calculator /pack-calculator
COPY --from=build /out/data /data

# SQLite file lives here; anonymous volume keeps data across container restarts.
VOLUME ["/data"]

ENV PORT=8080 \
    DB_PATH=/data/app.db

EXPOSE 8080

ENTRYPOINT ["/pack-calculator"]
