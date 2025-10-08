#!/bin/sh

set -e

/app/shardeumd init docker --home /app

curl -s http://$GENESIS_SOURCE/genesis | jq -r '.result.genesis' > "/app/config/genesis.json"

/app/shardeumd start --home /app --chain-id shardeum-testnet --p2p.seeds "$PEERS" --p2p.persistent_peers "$PEERS"
