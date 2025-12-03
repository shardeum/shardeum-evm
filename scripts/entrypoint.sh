#!/bin/sh

set -e

MONIKER=${MONIKER:-docker}

# ============================================
# DETERMINE NODE HOME DIRECTORY
# ============================================
# Cosmos SDK convention for node home
NODE_HOME="/app/.shardeumd"

# Validate NODE_TYPE
case $NODE_TYPE in
  API|SENTRY)
    echo "Node type: $NODE_TYPE"
    ;;
  *)
    echo "ERROR: NODE_TYPE must be 'API' or 'SENTRY'"
    exit 1
    ;;
esac

# ============================================
# VALIDATE REQUIRED ENVIRONMENT VARIABLES
# ============================================
missing=0

# Common required vars
for ENVAR in SHARDEUM_NETWORK CHAIN_ID EVM_CHAIN_ID SEEDS; do
  eval value=\$$ENVAR
  if [ -z "$value" ]; then
    echo "ERROR: Missing required env variable '$ENVAR'"
    missing=1
  fi
done

# Sentry-specific vars (optional - for initial sync without validator)
if [ "$NODE_TYPE" = "SENTRY" ]; then
  if [ -z "$PERSISTENT_PEERS" ]; then
    echo "WARNING: PERSISTENT_PEERS not set - sentry will sync from seeds only"
  fi
  if [ -z "$PRIVATE_PEER_IDS" ]; then
    echo "WARNING: PRIVATE_PEER_IDS not set - no peer hiding configured"
  fi
fi

if [ "$missing" = "1" ]; then exit 1; fi

# ============================================
# INITIALIZE NODE IF NEEDED
# ============================================
if [ ! -e "$NODE_HOME/config/genesis.json" ]; then
  echo "=== Initializing node: $MONIKER ==="
  /app/shardeumd init $MONIKER --home $NODE_HOME
fi

# ============================================
# CONFIGURE GENESIS
# ============================================
case $SHARDEUM_NETWORK in
  local)
    if [ -z "$GENESIS_URL" ]; then
      echo "ERROR: GENESIS_URL required for local network"
      exit 1
    fi
    echo "=== Downloading genesis from $GENESIS_URL ==="
    curl -s $GENESIS_URL | jq -r '.result.genesis' > $NODE_HOME/config/genesis.json
    ;;
  testnet)
    echo "=== Using testnet genesis ==="
    cp /app/config/testnet-genesis.json $NODE_HOME/config/genesis.json
    ;;
  mainnet)
    echo "=== Using mainnet genesis ==="
    cp /app/config/mainnet-genesis.json $NODE_HOME/config/genesis.json
    ;;
  *)
    echo "ERROR: SHARDEUM_NETWORK must be 'local', 'testnet', or 'mainnet'"
    exit 1
    ;;
esac

# ============================================
# CONFIGURE SENTRY-SPECIFIC SETTINGS
# ============================================
if [ "$NODE_TYPE" = "SENTRY" ]; then
  echo "=== Configuring sentry settings in config.toml ==="
  sed -i 's/addr_book_strict = true/addr_book_strict = false/' "$NODE_HOME/config/config.toml"
fi

# ============================================
# BUILD START OPTIONS
# ============================================
case $NODE_TYPE in
  API)
    OPTIONS="--p2p.seeds ${SEEDS} \
             --p2p.persistent_peers ${SEEDS} \
             --rpc.laddr tcp://0.0.0.0:26657 \
             --api.enable \
             --api.address tcp://0.0.0.0:1317 \
             --grpc.enable \
             --grpc.address 0.0.0.0:9090 \
             --json-rpc.enable \
             --json-rpc.address 0.0.0.0:8545 \
             --json-rpc.ws-address 0.0.0.0:8546 \
             --evm.evm-chain-id ${EVM_CHAIN_ID} \
             --minimum-gas-prices=2048130280389041ashm \
             --pruning nothing"
    ;;

  SENTRY)
    P2P_PORT=${P2P_PORT:-26656}
    RPC_PORT=${RPC_PORT:-26657}
    P2P_ADDRESS=${P2P_ADDRESS:-0.0.0.0}
    RPC_ADDRESS=${RPC_ADDRESS:-127.0.0.1}

    OPTIONS="--p2p.seeds ${SEEDS} \
             --p2p.pex=true \
             --p2p.laddr tcp://${P2P_ADDRESS}:${P2P_PORT} \
             --rpc.laddr tcp://${RPC_ADDRESS}:${RPC_PORT} \
             --evm.evm-chain-id ${EVM_CHAIN_ID} \
             --pruning nothing"
    # Add validator connection options only if set
    if [ -n "$PERSISTENT_PEERS" ]; then
      OPTIONS="$OPTIONS --p2p.persistent_peers ${PERSISTENT_PEERS}"
    fi
    if [ -n "$PRIVATE_PEER_IDS" ]; then
      OPTIONS="$OPTIONS --p2p.private_peer_ids ${PRIVATE_PEER_IDS}"
      OPTIONS="$OPTIONS --p2p.unconditional_peer_ids ${PRIVATE_PEER_IDS}"
    fi
    ;;
esac

# ============================================
# START NODE
# ============================================
echo "==========================================="
echo "Starting $NODE_TYPE node"
echo "  Moniker: $MONIKER"
echo "  Chain ID: $CHAIN_ID"
echo "  EVM Chain ID: $EVM_CHAIN_ID"
echo "  Network: $SHARDEUM_NETWORK"
echo "  Seeds: $SEEDS"
echo "  Home: $NODE_HOME"
if [ "$NODE_TYPE" = "SENTRY" ]; then
  echo "  Persistent Peers: $PERSISTENT_PEERS"
  echo "  Private Peer IDs: $PRIVATE_PEER_IDS"
  echo "  P2P: ${P2P_ADDRESS}:${P2P_PORT}"
  echo "  RPC: ${RPC_ADDRESS}:${RPC_PORT}"
fi
echo "==========================================="

exec /app/shardeumd start --home $NODE_HOME --chain-id $CHAIN_ID $OPTIONS
