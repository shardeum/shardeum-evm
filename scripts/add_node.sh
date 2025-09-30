#!/bin/bash

set -e

CURRENT_DIR="$(pwd)"

# Parse command line arguments
NODE_ID=""
SEED_NODE_RPC=""
NETWORK="local"
CUSTOM_CHAIN_ID=""
NODE_TYPE="validator"

# Usage function
usage() {
  echo "Usage: $0 <node_id> [options]"
  echo "  node_id: Unique identifier for the new node (e.g., node4, node5)"
  echo ""
  echo "Options:"
  echo "  --seed-rpc <url>     RPC endpoint of seed node (default: http://localhost:26657)"
  echo "  --network <name>     Network to use (mainnet, testnet, devnet, local)"
  echo "  --chain-id <id>      Override chain ID"
  echo "  --node-type <type>   Node type: validator or full-node (default: validator)"
  echo "  --help               Show this help message"
  echo ""
  echo "Environment variables:"
  echo "  SHARDEUM_NETWORK     Network to use (overrides --network)"
  echo "  SHARDEUM_CHAIN_ID    Chain ID to use (overrides --chain-id)"
  echo "  SHARDEUM_CONFIG_DIR  Absolute path to directory containing config/environments/*.json"
  echo "  BINARY               Path to shardeumd binary"
  echo ""
  echo "Examples:"
  echo "  $0 node4 --network testnet"
  echo "  $0 node5 --seed-rpc http://localhost:26657 --network devnet"
  echo "  $0 node6 --node-type full-node --network testnet"
  echo "  SHARDEUM_CONFIG_DIR=/path/to/config SHARDEUM_NETWORK=testnet $0 node6"
  exit 1
}

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --seed-rpc)
      SEED_NODE_RPC="$2"
      shift 2
      ;;
    --network)
      NETWORK="$2"
      shift 2
      ;;
    --chain-id)
      CUSTOM_CHAIN_ID="$2"
      shift 2
      ;;
    --node-type)
      NODE_TYPE="$2"
      shift 2
      ;;
    --help)
      echo "Usage: $0 <node_id> [options]"
      echo "  node_id: Unique identifier for the new node (e.g., node4, node5)"
      echo ""
      echo "Options:"
      echo "  --seed-rpc <url>     RPC endpoint of seed node (default: http://localhost:26657)"
      echo "  --network <name>     Network to use (mainnet, testnet, devnet, local)"
      echo "  --chain-id <id>      Override chain ID"
      echo "  --node-type <type>   Node type: validator or full-node (default: validator)"
      echo "  --help               Show this help message"
      echo ""
      echo "Environment variables:"
      echo "  SHARDEUM_NETWORK     Network to use (overrides --network)"
      echo "  SHARDEUM_CHAIN_ID    Chain ID to use (overrides --chain-id)"
      echo "  SHARDEUM_CONFIG_DIR  Absolute path to directory containing configs/*.json"
      echo "  BINARY               Path to shardeumd binary"
      echo ""
      echo "Examples:"
      echo "  $0 node4 --network testnet"
      echo "  $0 node5 --seed-rpc http://localhost:26657 --network devnet"
      echo "  $0 node6 --node-type full-node --network testnet"
      echo "  SHARDEUM_CONFIG_DIR=path/to/config SHARDEUM_NETWORK=testnet $0 node6"
      exit 0
      ;;
    --*)
      echo "Unknown option: $1"
      usage
      ;;
    *)
      if [[ -z "$NODE_ID" ]]; then
        NODE_ID="$1"
      else
        echo "Unknown argument: $1"
        usage
      fi
      shift
      ;;
  esac
done

# config dir is compulsory - can't start network without it
if [[ -z "$SHARDEUM_CONFIG_DIR" ]]; then
  echo "Error: SHARDEUM_CONFIG_DIR is required"
  usage
fi

# Validate required arguments
if [[ -z "$NODE_ID" ]]; then
  echo "Error: node_id is required"
  usage
fi

# Validate node type
if [[ "$NODE_TYPE" != "validator" && "$NODE_TYPE" != "full-node" ]]; then
  echo "Error: node-type must be either 'validator' or 'full-node'"
  usage
fi


# Set defaults
SEED_NODE_RPC="${SEED_NODE_RPC:-http://localhost:26657}"
NETWORK="${SHARDEUM_NETWORK:-${NETWORK:-testnet}}"

# Load network configuration
CONFIG_FILE="$SHARDEUM_CONFIG_DIR/environments/$NETWORK.json"
if [ ! -f "$CONFIG_FILE" ]; then
  echo -e "${RED}Error: Network configuration file not found: $CONFIG_FILE${NC}"
  echo "Available networks:"
  ls -1 "$SHARDEUM_CONFIG_DIR/environments"/*.json 2>/dev/null | xargs -n1 basename | sed 's/.json$//' | sed 's/^/  /' || echo "  No network configurations found"
  exit 1
fi

# Read network configuration
CHAINID=$(jq -r '.chain_id' "$CONFIG_FILE")
EVM_CHAIN_ID=$(jq -r '.evm_chain_id' "$CONFIG_FILE")
BASE_DENOM=$(jq -r '.base_denom' "$CONFIG_FILE")

# Override chain ID if provided
if [[ -n "$CUSTOM_CHAIN_ID" ]]; then
  CHAINID="$CUSTOM_CHAIN_ID"
elif [[ -n "$SHARDEUM_CHAIN_ID" ]]; then
  CHAINID="$SHARDEUM_CHAIN_ID"
fi

BINARY="${BINARY:-$(command -v shardeumd)}"
BASE_DIR="${HOME:-$CURRENT_DIR}/.$NETWORK"
if [[ "$NETWORK" == "local" ]]; then
  BASE_DIR="$CURRENT_DIR/.$NETWORK"
fi
MIN_GAS="2048130280389041$BASE_DENOM"

# Ensure the binary can resolve configs via SHARDEUM_CONFIG_DIR
export SHARDEUM_CONFIG_DIR="$SHARDEUM_CONFIG_DIR"
export SHARDEUM_NETWORK="$NETWORK"

# Cleanup function for graceful shutdown
cleanup() {
  echo -e "\n${YELLOW}Shutting down node...${NC}"
  if [ -f "$NODE_DIR/node.pid" ]; then
    NODE_PID=$(cat "$NODE_DIR/node.pid")
    kill $NODE_PID 2>/dev/null || true
  fi
  exit 0
}

# Trap signals for graceful shutdown
trap cleanup SIGINT SIGTERM

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Usage function
usage() {
  echo "Usage: $0 <node_id> [seed_node_rpc]"
  echo "  node_id: Unique identifier for the new node (e.g., node4, node5)"
  echo "  seed_node_rpc: RPC endpoint of seed node (default: http://localhost:26657)"
  echo ""
  echo "Examples:"
  echo "  $0 node4                           # Add node4, use localhost:26657 as seed"
  echo "  $0 node5 http://localhost:26658    # Add node5, use node1 as seed"
  exit 1
}

# Validate arguments
if [ -z "$NODE_ID" ]; then
  echo -e "${RED}Error: Node ID is required${NC}"
  usage
fi

echo -e "${GREEN}Adding new node '$NODE_ID' using seed-based discovery${NC}"
echo -e "${YELLOW}Using seed node: $SEED_NODE_RPC${NC}"
echo -e "${YELLOW}Node type: $NODE_TYPE${NC}"

# Verify binary exists (skip build if called from makefile)
if [ ! -f "$BINARY" ]; then
  echo -e "${RED}Error: Failed to find binary at $BINARY${NC}"
  echo -e "${RED}Tip:${NC} Make sure shardeumd binary is set in PATH or set BINARY=/absolute/path/to/shardeumd"
  exit 1
fi

# Verify we can connect to seed node
echo -e "${YELLOW}Checking connection to seed node...${NC}"
if ! curl -s "$SEED_NODE_RPC/status" > /dev/null; then
  echo -e "${RED}Error: Cannot connect to seed node at $SEED_NODE_RPC${NC}"
  echo "Make sure the seed node is running"
  exit 1
fi

# Find next available node number if NODE_ID is just a number
if [[ "$NODE_ID" =~ ^[0-9]+$ ]]; then
  NODE_ID="node$NODE_ID"
fi

NODE_DIR="$BASE_DIR/$NODE_ID"

# Check if node already exists
if [ -d "$NODE_DIR" ]; then
  echo -e "${RED}Error: Node directory $NODE_DIR already exists${NC}"
  echo "Choose a different node ID or remove the existing directory"
  exit 1
fi

# Initialize new node
echo -e "${YELLOW}Initializing new node: $NODE_ID${NC}"
"$BINARY" init "$NODE_ID" --chain-id "$CHAINID" --home "$NODE_DIR" --overwrite > /dev/null 2>&1

# Get genesis from seed node
echo -e "${YELLOW}Fetching genesis from seed node...${NC}"
GENESIS_URL="$SEED_NODE_RPC/genesis"
if ! curl -s "$GENESIS_URL" | jq -r '.result.genesis' > "$NODE_DIR/config/genesis.json"; then
  echo -e "${RED}Error: Failed to fetch genesis from $GENESIS_URL${NC}"
  exit 1
fi

# Find next available ports
echo -e "${YELLOW}Finding available ports...${NC}"
NEXT_NODE_NUM=0
for existing_dir in "$BASE_DIR"/node*; do
  if [ -d "$existing_dir" ]; then
    existing_num=$(basename "$existing_dir" | sed 's/node//')
    if [[ "$existing_num" =~ ^[0-9]+$ ]] && [ "$existing_num" -ge "$NEXT_NODE_NUM" ]; then
      NEXT_NODE_NUM=$((existing_num + 1))
    fi
  fi
done

# Port assignments
RPC_PORT=$((26657 + NEXT_NODE_NUM))
P2P_PORT=$((27656 + NEXT_NODE_NUM))
GRPC_PORT=$((9090 + NEXT_NODE_NUM))
API_PORT=$((1317 + NEXT_NODE_NUM))
JSON_PORT=$((8545 + NEXT_NODE_NUM * 2))
WS_PORT=$((8546 + NEXT_NODE_NUM * 2))

echo -e "${YELLOW}Assigned ports: RPC=$RPC_PORT, P2P=$P2P_PORT, JSON-RPC=$JSON_PORT${NC}"

# Get seed node info
SEED_RPC_PORT=$(echo $SEED_NODE_RPC | sed 's/.*://' | sed 's/[^0-9]//g')
SEED_P2P_PORT=$((SEED_RPC_PORT + 999))  # RPC port 26657 -> P2P port 27656
SEED_NODE_ID=$(curl -s "$SEED_NODE_RPC/status" | jq -r '.result.node_info.id')

if [ -z "$SEED_NODE_ID" ] || [ "$SEED_NODE_ID" = "null" ]; then
  echo -e "${RED}Error: Could not get seed node ID${NC}"
  exit 1
fi

SEED_ADDRESS="$SEED_NODE_ID@127.0.0.1:$SEED_P2P_PORT"
echo -e "${YELLOW}Using seed: $SEED_ADDRESS${NC}"

# Configure seeds and local development settings
# 1) Set seeds to the provided seed node
# 2) Auto-pick up to 3 existing peers as persistent_peers for better mesh connectivity
# 3) Allow duplicate IPs for localhost multi-node setups

# Build a comma-separated list of up to 3 persistent peers from existing nodes
PERSISTENT_PEERS=""
PEER_COUNT=0
for existing_dir in "$BASE_DIR"/node*; do
  [ -d "$existing_dir" ] || continue
  existing_name=$(basename "$existing_dir")
  # Skip the new node itself
  if [ "$existing_name" = "$NODE_ID" ]; then
    continue
  fi
  idx=${existing_name#node}

  # Query each existing node's RPC for its node ID. If unavailable, skip.
  rpc_port=$((26657 + idx))
  p2p_port=$((27656 + idx))
  peer_id=$(curl -s "http://localhost:${rpc_port}/status" | jq -r '.result.node_info.id')
  if [ -n "$peer_id" ] && [ "$peer_id" != "null" ]; then
    addr="${peer_id}@127.0.0.1:${p2p_port}"
    if [ -z "$PERSISTENT_PEERS" ]; then
      PERSISTENT_PEERS="$addr"
    else
      PERSISTENT_PEERS="$PERSISTENT_PEERS,$addr"
    fi
    PEER_COUNT=$((PEER_COUNT + 1))
  fi

  # Limit to 3 peers
  if [ $PEER_COUNT -ge 3 ]; then
    break
  fi
done

# Escape for sed replacement
_ESCAPED_PERSISTENT=$(printf '%s' "$PERSISTENT_PEERS" | sed 's/[\\/&]/\\&/g')

if [[ "$OSTYPE" == "darwin"* ]]; then
  # Set seeds
  sed -i '' "s/seeds = \"\"/seeds = \"$SEED_ADDRESS\"/" "$NODE_DIR/config/config.toml"
  # Set persistent_peers (line exists by default), even if empty string
  sed -i '' "s/^persistent_peers = \".*\"/persistent_peers = \"${_ESCAPED_PERSISTENT}\"/" "$NODE_DIR/config/config.toml"
  # Allow duplicate IPs
  sed -i '' "s/allow_duplicate_ip = false/allow_duplicate_ip = true/" "$NODE_DIR/config/config.toml"
else
  sed -i "s/seeds = \"\"/seeds = \"$SEED_ADDRESS\"/" "$NODE_DIR/config/config.toml"
  sed -i "s/^persistent_peers = \".*\"/persistent_peers = \"${_ESCAPED_PERSISTENT}\"/" "$NODE_DIR/config/config.toml"
  sed -i "s/allow_duplicate_ip = false/allow_duplicate_ip = true/" "$NODE_DIR/config/config.toml"
  sed -i "s#address = \"tcp://localhost:1317\"#address = \"tcp://0.0.0.0:$API_PORT\"#" "$NODE_DIR/config/app.toml"
fi

if [ -n "$PERSISTENT_PEERS" ]; then
  echo -e "${YELLOW}Configured persistent_peers (${PEER_COUNT}): ${PERSISTENT_PEERS}${NC}"
else
  echo -e "${YELLOW}No available existing peers discovered for persistent_peers; proceeding with seeds only${NC}"
fi

# Set up client configuration
cat > "$NODE_DIR/config/client.toml" << EOF
chain-id = "$CHAINID"
keyring-backend = "test"
node = "tcp://localhost:$RPC_PORT"
broadcast-mode = "sync"
EOF

# Start the new node
echo -e "${YELLOW}Starting new node $NODE_ID...${NC}"

# Build start command based on node type
START_CMD=(
  "$BINARY" start
  --home "$NODE_DIR"
  --chain-id "$CHAINID"
  --rpc.laddr "tcp://127.0.0.1:$RPC_PORT"
  --p2p.laddr "tcp://0.0.0.0:$P2P_PORT"
  --grpc.address "localhost:$GRPC_PORT"
  --minimum-gas-prices="$MIN_GAS"
  --pruning nothing
)

# Configure RPC and validator settings based on node type
if [[ "$NODE_TYPE" == "full-node" ]]; then
#  This flag was working for some, but not all so testing without it for now
#  START_CMD+=(--non-validator)
  START_CMD+=(--json-rpc.enable)
  START_CMD+=(--json-rpc.address "127.0.0.1:$JSON_PORT")
  START_CMD+=(--json-rpc.ws-address "127.0.0.1:$WS_PORT")
  START_CMD+=(--json-rpc.api eth,txpool,personal,net,debug,web3)
  START_CMD+=(--api.enable)
fi

# Execute the start command
"${START_CMD[@]}" > "$NODE_DIR/node.log" 2>&1 &

NODE_PID=$!
echo $NODE_PID > "$NODE_DIR/node.pid"


echo
echo -e "${GREEN}✅ Node $NODE_ID started with seed-based discovery${NC}"
echo -e "${YELLOW}Chain ID: $CHAINID${NC}"
echo -e "${YELLOW}Node Type: $NODE_TYPE${NC}"
echo
echo "New node endpoints:"
echo "  RPC: http://localhost:$RPC_PORT"
if [[ "$NODE_TYPE" == "full-node" ]]; then
  echo "  JSON-RPC: http://localhost:$JSON_PORT"
  echo "  WebSocket: ws://localhost:$WS_PORT"
else
  echo "  (JSON-RPC disabled for validator security)"
fi
echo
echo "Seed node: $SEED_ADDRESS"
echo "The node will discover peers automatically via PEX protocol"
echo

if [[ "$NODE_TYPE" == "validator" ]]; then
echo -e "${YELLOW}📝 To create a validator:${NC}"
echo "1. Create and fund a validator account"
echo "2. Run: ./scripts/create_validator.sh $NODE_ID"
echo
fi

echo "Stop this node: kill $NODE_PID or Ctrl+C"
echo "Logs: $NODE_DIR/node.log"
echo
echo -e "${YELLOW}Node is starting up and discovering peers. Press Ctrl+C to stop.${NC}"

# Wait for interrupt signal
wait
