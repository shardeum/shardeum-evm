#!/bin/sh

set -e

for ENVAR in PEERS SHARDEUM_NETWORK; do
  eval value=\$$ENVAR
  if [ -z "$value" ]; then
    echo "Missing required env variable $ENVAR"
    missing=1
  fi
done

if [ $missing ]; then exit 1; fi

if [ ! -f "/app/data/state.db/LOCK" ]; then
  if [ -z "$GENESIS_SOURCE" ]; then
    echo "Uninitialized nodes require a GENESIS_SOURCE"
    exit 1
  fi
  /app/shardeumd init docker --home /app/data
  curl -s http://$GENESIS_SOURCE/genesis | jq -r '.result.genesis' > "/app/config/genesis.json"
fi

case $NODE_TYPE in
  RPC)
    OPTIONS='--rpc.laddr "tcp://0.0.0.0:26657" --json-rpc.enable --json-rpc.address "0.0.0.0:8545" --api.enable --api.address "tcp://0.0.0.0:1317"'
    ;;
  VALIDATOR)
    OPTIONS='--p2p.laddr "tcp://0.0.0.0:27656"'
    ;;
  SENTRY)
    OPTIONS='--p2p.laddr "tcp://0.0.0.0:27656"'
    ;;
esac

/app/shardeumd start --home /app/data --chain-id shardeum-testnet --p2p.seeds "$PEERS" --p2p.persistent_peers "$PEERS" $OPTIONS
