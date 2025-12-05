#!/bin/sh

set -e

MONIKER=${MONIKER:-docker}

# ============================================
# DETERMINE NODE HOME DIRECTORY
# ============================================
case $NODE_TYPE in
  API)
    NODE_HOME="/app"
    ;;
  SENTRY)
    NODE_HOME="/app/.shardeumd"
    ;;
  *)
    echo "ERROR: NODE_TYPE must be 'API' or 'SENTRY'"
    exit 1
    ;;
esac

# ============================================
# LOGGING CONFIGURATION
# ============================================
LOG_DIR="${NODE_HOME}/logs"
LOG_FILE="${LOG_DIR}/node.log"
LOG_MAX_SIZE_MB=${LOG_MAX_SIZE_MB:-100}      # Max size before rotation (default 100MB)
LOG_MAX_FILES=${LOG_MAX_FILES:-5}             # Number of rotated files to keep

mkdir -p "$LOG_DIR"

# Log rotation function - rotates when file exceeds max size
rotate_logs() {
  if [ -f "$LOG_FILE" ]; then
    FILE_SIZE=$(stat -c%s "$LOG_FILE" 2>/dev/null || stat -f%z "$LOG_FILE" 2>/dev/null || echo "0")
    MAX_SIZE=$((LOG_MAX_SIZE_MB * 1024 * 1024))
    
    if [ "$FILE_SIZE" -gt "$MAX_SIZE" ]; then
      echo "$(date -Iseconds) Rotating logs (size: $FILE_SIZE bytes)" >> "$LOG_FILE"
      
      # Rotate existing logs
      i=$((LOG_MAX_FILES - 1))
      while [ $i -gt 0 ]; do
        prev=$((i - 1))
        [ -f "${LOG_FILE}.$prev" ] && mv "${LOG_FILE}.$prev" "${LOG_FILE}.$i"
        i=$((i - 1))
      done
      
      # Move current log to .0
      mv "$LOG_FILE" "${LOG_FILE}.0"
      
      # Compress old logs (if gzip available)
      if command -v gzip >/dev/null 2>&1; then
        for f in ${LOG_FILE}.[1-9]*; do
          [ -f "$f" ] && [ ! -f "$f.gz" ] && gzip "$f" &
        done
      fi
    fi
  fi
}

# Background log rotation checker (runs every 5 minutes)
start_log_rotator() {
  (
    while true; do
      sleep 300
      rotate_logs
    done
  ) &
  LOG_ROTATOR_PID=$!
  echo "Log rotator started (PID: $LOG_ROTATOR_PID)"
}

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
echo "  Log File: $LOG_FILE"
echo "  Log Max Size: ${LOG_MAX_SIZE_MB}MB"
echo "  Log Max Files: $LOG_MAX_FILES"
if [ "$NODE_TYPE" = "SENTRY" ]; then
  echo "  Persistent Peers: $PERSISTENT_PEERS"
  echo "  Private Peer IDs: $PRIVATE_PEER_IDS"
  echo "  P2P: ${P2P_ADDRESS}:${P2P_PORT}"
  echo "  RPC: ${RPC_ADDRESS}:${RPC_PORT}"
fi
echo "==========================================="

# Start log rotation in background
start_log_rotator

# Run node with logs going to BOTH stdout AND file (using tee)
# This preserves docker logs functionality while also writing to persistent disk
exec /app/shardeumd start --home $NODE_HOME --chain-id $CHAIN_ID $OPTIONS 2>&1 | tee -a "$LOG_FILE"
