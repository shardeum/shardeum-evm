#!/bin/bash

set -e

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

# Parse command line arguments
NODE_ID="${1:-}"
SEED_NODE_RPC="${2:-http://localhost:26657}"
CHAINID="shardeum-testnet"
BASE_DIR="../.testnet"
MIN_GAS="0.000006ashm"
BINARY="${BINARY:-../build/shardeumd}"

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

# Verify binary exists (skip build if called from makefile)
if [ ! -f "$BINARY" ]; then
  if [ "$SKIP_BUILD" != "1" ]; then
    echo -e "${YELLOW}Binary not found, building shardeumd...${NC}"
    (cd .. && make build)
  fi
  if [ ! -f "$BINARY" ]; then
    echo -e "${RED}Error: Failed to build binary at $BINARY${NC}"
    exit 1
  fi
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
if [[ "$OSTYPE" == "darwin"* ]]; then
  sed -i '' "s/seeds = \"\"/seeds = \"$SEED_ADDRESS\"/" "$NODE_DIR/config/config.toml"
  sed -i '' "s/persistent_peers = \"\"//" "$NODE_DIR/config/config.toml"  # Keep empty
  sed -i '' "s/allow_duplicate_ip = false/allow_duplicate_ip = true/" "$NODE_DIR/config/config.toml"
else
  sed -i "s/seeds = \"\"/seeds = \"$SEED_ADDRESS\"/" "$NODE_DIR/config/config.toml"
  sed -i "s/persistent_peers = \"\"//" "$NODE_DIR/config/config.toml"  # Keep empty
  sed -i "s/allow_duplicate_ip = false/allow_duplicate_ip = true/" "$NODE_DIR/config/config.toml"
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
"$BINARY" start \
  --home "$NODE_DIR" \
  --chain-id "$CHAINID" \
  --rpc.laddr "tcp://127.0.0.1:$RPC_PORT" \
  --p2p.laddr "tcp://0.0.0.0:$P2P_PORT" \
  --grpc.address "localhost:$GRPC_PORT" \
  --json-rpc.enable \
  --json-rpc.address "127.0.0.1:$JSON_PORT" \
  --json-rpc.ws-address "127.0.0.1:$WS_PORT" \
  --minimum-gas-prices="$MIN_GAS" \
  --json-rpc.api eth,txpool,personal,net,debug,web3 \
  --pruning nothing \
  > "$NODE_DIR/node.log" 2>&1 &

NODE_PID=$!
echo $NODE_PID > "$NODE_DIR/node.pid"

echo
echo -e "${GREEN}✅ Node $NODE_ID started with seed-based discovery${NC}"
echo -e "${YELLOW}Chain ID: $CHAINID${NC}"
echo
echo "New node endpoints:"
echo "  RPC: http://localhost:$RPC_PORT"
echo "  JSON-RPC: http://localhost:$JSON_PORT"
echo "  WebSocket: ws://localhost:$WS_PORT"
echo
echo "Seed node: $SEED_ADDRESS"
echo "The node will discover peers automatically via PEX protocol"
echo
echo "Stop this node: kill $NODE_PID or Ctrl+C"
echo "Logs: $NODE_DIR/node.log"
echo
echo -e "${YELLOW}Node is starting up and discovering peers. Press Ctrl+C to stop.${NC}"

# Wait for interrupt signal
wait