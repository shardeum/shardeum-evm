# Build stage
FROM golang:1.26.11-alpine3.24 AS builder

WORKDIR /app

# git is needed so `make build` can stamp the version (git describe) and commit (git log)
RUN apk add --no-cache build-base gcc linux-headers make git

COPY . .

RUN make build

# Production stage
FROM alpine:3.24.4

# Needed to download genesis.json
RUN apk add --no-cache curl jq

WORKDIR /app

# Set up non-root user
RUN addgroup shardeum
RUN adduser -D shardeum -G shardeum
RUN chown -R shardeum:shardeum /app

COPY --from=builder --chown=shardeum:shardeum /app/build/shardeumd .
COPY --from=builder --chown=shardeum:shardeum /app/config/environments config/environments
COPY --from=builder --chown=shardeum:shardeum /app/config/*-genesis.json config/
COPY --from=builder --chown=shardeum:shardeum --chmod=0755 /app/scripts/entrypoint.sh .

USER shardeum

CMD ["/app/entrypoint.sh"]
