# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache build-base gcc linux-headers make

COPY . .

RUN make build

# Production stage
FROM alpine

WORKDIR /app

# Set up non-root user
RUN addgroup shardeum
RUN adduser -D shardeum -G shardeum
RUN chown -R shardeum:shardeum /app

COPY --from=builder --chown=shardeum:shardeum /app/build/shardeumd /app/
COPY --from=builder --chown=shardeum:shardeum /app/config/environments config/environments

USER shardeum

CMD ["/app/shardeumd"]

