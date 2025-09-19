#!/bin/bash

# Helper script to set network environment variables
# Usage: source ./scripts/set_network.sh <network>
# Example: source ./scripts/set_network.sh mainnet

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Show usage if no arguments provided
if [ $# -eq 0 ]; then
  echo "Usage: source $0 <network>"
  echo ""
  echo "Available networks:"
  ls -1 "$REPO_ROOT/configs"/*.json 2>/dev/null | xargs -n1 basename | sed 's/.json$//' | sed 's/^/  /' || echo "  No network configurations found"
  echo ""
  echo "Examples:"
  echo "  source $0 mainnet"
  echo "  source $0 testnet"
  echo "  source $0 devnet"
  echo "  source $0 local"
  return 1 2>/dev/null || exit 1
fi

NETWORK="$1"
CONFIG_FILE="$REPO_ROOT/configs/$NETWORK.json"

# Verify network configuration exists
if [ ! -f "$CONFIG_FILE" ]; then
  echo "Error: Network configuration file not found: $CONFIG_FILE"
  echo ""
  echo "Available networks:"
  ls -1 "$REPO_ROOT/configs"/*.json 2>/dev/null | xargs -n1 basename | sed 's/.json$//' | sed 's/^/  /' || echo "  No network configurations found"
  return 1 2>/dev/null || exit 1
fi

# Load network configuration and set environment variables
echo "Setting environment variables for $NETWORK network..."

export SHARDEUM_NETWORK="$NETWORK"
export SHARDEUM_CHAIN_ID=$(jq -r '.chain_id' "$CONFIG_FILE")
export SHARDEUM_EVM_CHAIN_ID=$(jq -r '.evm_chain_id' "$CONFIG_FILE")
export SHARDEUM_BASE_DENOM=$(jq -r '.base_denom' "$CONFIG_FILE")
export SHARDEUM_DISPLAY_DENOM=$(jq -r '.display_denom' "$CONFIG_FILE")
export SHARDEUM_RPC_PORT=$(jq -r '.ports.rpc' "$CONFIG_FILE")
export SHARDEUM_REST_PORT=$(jq -r '.ports.rest' "$CONFIG_FILE")
export SHARDEUM_JSON_RPC_PORT=$(jq -r '.ports.json_rpc' "$CONFIG_FILE")
export SHARDEUM_WEBSOCKET_PORT=$(jq -r '.ports.websocket' "$CONFIG_FILE")
export SHARDEUM_GRPC_PORT=$(jq -r '.ports.grpc' "$CONFIG_FILE")

echo "Environment variables set:"
echo "  SHARDEUM_NETWORK=$SHARDEUM_NETWORK"
echo "  SHARDEUM_CHAIN_ID=$SHARDEUM_CHAIN_ID"
echo "  SHARDEUM_EVM_CHAIN_ID=$SHARDEUM_EVM_CHAIN_ID"
echo "  SHARDEUM_BASE_DENOM=$SHARDEUM_BASE_DENOM"
echo "  SHARDEUM_DISPLAY_DENOM=$SHARDEUM_DISPLAY_DENOM"
echo "  SHARDEUM_RPC_PORT=$SHARDEUM_RPC_PORT"
echo "  SHARDEUM_REST_PORT=$SHARDEUM_REST_PORT"
echo "  SHARDEUM_JSON_RPC_PORT=$SHARDEUM_JSON_RPC_PORT"
echo "  SHARDEUM_WEBSOCKET_PORT=$SHARDEUM_WEBSOCKET_PORT"
echo "  SHARDEUM_GRPC_PORT=$SHARDEUM_GRPC_PORT"
echo ""
echo "You can now run scripts without specifying --network flag"
echo "Example: ./scripts/start_network.sh 4"