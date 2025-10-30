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
    CHAIN_ID=shardeum-8119-2
    EVM_CHAIN_ID=8119
    cp /app/config/testnet-genesis.json /app/config/genesis.json
    ;;
  mainnet)
    CHAIN_ID=shardeum_8118-1
    EVM_CHAIN_ID=8118
    cp /app/config/mainnet-genesis.json /app/config/genesis.json
    ;;
esac

case $NODE_TYPE in
  API)
    OPTIONS="--rpc.laddr tcp://0.0.0.0:26657 \
             --api.enable \
             --api.address tcp://0.0.0.0:1317 \
             --json-rpc.enable \
             --json-rpc.address 0.0.0.0:8545 \
             --json-rpc.ws-address 0.0.0.0:8546 \
             --json-rpc.api eth,txpool,personal,net,debug,web3 \
             --evm.evm-chain-id $EVM_CHAIN_ID \
             --minimum-gas-prices=2048130280389041ashm \
             --pruning nothing"
    ;;
  VALIDATOR)
    OPTIONS='--p2p.laddr tcp://0.0.0.0:27656'
    ;;
  SENTRY)
    OPTIONS='--p2p.laddr tcp://0.0.0.0:27656'
    ;;
esac

/app/shardeumd start --home /app --chain-id $CHAIN_ID --p2p.seeds "$PEERS" --p2p.persistent_peers "$PEERS" $OPTIONS
