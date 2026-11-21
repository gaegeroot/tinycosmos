# Production image for cmd/bot. Build context is the repository root:
#   docker build -f infrastructure/deployment/bot.Dockerfile .
# fly.toml at the repository root points here, so `fly deploy` just works.

FROM golang:1.25-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY database ./database
COPY internal ./internal

# modernc.org/sqlite is pure Go, so the binary is fully static.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/bot ./cmd/bot

# distroless/static ships CA certificates (needed for api.telegram.org) and
# nothing else. It runs as root so the bot can write to the Fly volume, which
# is mounted root-owned.
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/bot /bot
ENV TINY_COSMOS_DB_PATH=/data/tiny-cosmos.db
ENTRYPOINT ["/bot"]
