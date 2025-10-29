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

if [ ! -e "/app/data/state.db" ]; then
  /app/shardeumd init docker --home /app
fi

case $SHARDEUM_NETWORK in
  testnet)
    cp /app/config/testnet-genesis.json /app/config/genesis.json
    ;;
  mainnet)
    cp /app/config/mainnet-genesis.json /app/config/genesis.json
    ;;
esac

case $NODE_TYPE in
  API)
    OPTIONS='--rpc.laddr tcp://0.0.0.0:26657 --json-rpc.enable --json-rpc.address 0.0.0.0:8545 --api.enable --api.address tcp://0.0.0.0:1317 --json-rpc.ws-address 0.0.0.0:8546 --pruning nothing --minimum-gas-prices=2048130280389041ashm'
    ;;
  VALIDATOR)
    OPTIONS='--p2p.laddr tcp://0.0.0.0:27656'
    ;;
  SENTRY)
    OPTIONS='--p2p.laddr tcp://0.0.0.0:27656'
    ;;
esac

/app/shardeumd start --home /app --chain-id shardeum-${SHARDEUM_NETWORK} --p2p.seeds "$PEERS" --p2p.persistent_peers "$PEERS" $OPTIONS
