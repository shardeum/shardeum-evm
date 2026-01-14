# Build stage
FROM golang:1.25.1-alpine3.22 AS builder

WORKDIR /app

RUN apk add --no-cache build-base gcc linux-headers make

COPY . .

RUN make build

# Production stage
FROM alpine:3.23.2

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
