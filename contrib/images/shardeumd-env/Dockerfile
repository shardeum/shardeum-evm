# Info on how to use this docker image can be found in DOCKER_README.md
ARG IMG_TAG=latest

# Compile the shardeumd binary
FROM golang:1.24-alpine AS shardeumd-builder
WORKDIR /work
ENV PACKAGES="curl build-base git bash file linux-headers eudev-dev"
RUN apk add --no-cache $PACKAGES

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN LEDGER_ENABLED=false COSMOS_BUILD_OPTIONS="staticlink" BUILD_TAGS=muslc make build
RUN echo "Checking binary linkage..." \
    && file /work/build/shardeumd \
    && (file /work/build/shardeumd | grep -q "statically linked" || echo "Warning: Binary may not be statically linked")

FROM alpine:$IMG_TAG AS run
RUN apk add --no-cache build-base jq bash curl
RUN addgroup -g 1025 nonroot
RUN adduser -D nonroot -u 1025 -G nonroot

# Set up the runtime environment
EXPOSE 26656 26657 1317 9090 26660 8545 8100
STOPSIGNAL SIGTERM
VOLUME /shardeumd
WORKDIR /shardeumd

# Copy the wrapper script and binary to expected locations
COPY contrib/images/shardeumd-env/wrapper.sh /usr/bin/wrapper.sh
COPY --from=shardeumd-builder /work/build/shardeumd /shardeumd/
COPY --from=shardeumd-builder /work/build/shardeumd /usr/local/bin/

# Set proper ownership and permissions before switching to nonroot user
RUN chown nonroot:nonroot /usr/bin/wrapper.sh && chmod +x /usr/bin/wrapper.sh
RUN chown -R nonroot:nonroot /shardeumd

USER nonroot

ENTRYPOINT ["/usr/bin/wrapper.sh"]
CMD ["start", "--log_format", "plain", "--minimum-gas-prices", "0.000006ashm", "--json-rpc.enable", "true", "--json-rpc.api", "eth,txpool,personal,net,debug,web3", "--chain-id", "shardeum_8119-1"]