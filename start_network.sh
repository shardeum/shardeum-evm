#!/bin/bash

set -e

# Cleanup function for graceful shutdown
cleanup() {
  echo -e "\n${YELLOW}Shutting down nodes...${NC}"
  pkill -f "shardeumd.*$CHAINID" || true
  exit 0
}

# Trap signals for graceful shutdown
trap cleanup SIGINT SIGTERM

NODES="${1:-4}"
CHAINID="shardeum-testnet"  # Use proper Shardeum chain ID
BASE_DIR="./.testnet"
MIN_GAS="0.000006ashm"
BINARY="./build/shardeumd"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}Starting $NODES node Shardeum testnet${NC}"

# Clean up
rm -rf "$BASE_DIR"
pkill -f "shardeumd.*shardeum-testnet" || true

# Build our own binary locally
echo -e "${YELLOW}Building shardeumd binary${NC}"
make build

# Verify binary exists
if [ ! -f "$BINARY" ]; then
  echo -e "${RED}Error: Binary not found at $BINARY${NC}"
  echo "Make sure 'make build' completed successfully"
  exit 1
fi

# Verify genesis file exists
if [ ! -f "config/genesis.json" ]; then
  echo -e "${RED}Error: Genesis file not found at config/genesis.json${NC}"
  echo "Make sure you have the proper Shardeum network configuration"
  exit 1
fi

# Initialize first node and create base genesis
echo -e "${YELLOW}Initializing primary node${NC}"
NODE0_DIR="$BASE_DIR/node0"
mkdir -p "$NODE0_DIR"

# Initialize node0 with proper chain-id to get correct genesis template
"$BINARY" init "node0" --chain-id "$CHAINID" --home "$NODE0_DIR" --overwrite > /dev/null 2>&1

# Replace with our custom genesis immediately after init
echo -e "${YELLOW}Using custom genesis with ashm denomination${NC}"
cp "config/genesis.json" "$NODE0_DIR/config/genesis.json"

# Create validator key for node0 FIRST (before genesis setup)
"$BINARY" keys add "validator0" \
  --keyring-backend test --home "$NODE0_DIR" > /dev/null 2>&1 || true

# Add validator account to genesis with sufficient stake
"$BINARY" genesis add-genesis-account "validator0" 100000000000000000000000000ashm \
  --keyring-backend test --home "$NODE0_DIR" > /dev/null 2>&1

# Create genesis transaction with proper chain-id
echo -e "${YELLOW}Creating genesis transaction for validator${NC}"
if "$BINARY" genesis gentx validator0 1000000000000000000ashm \
  --chain-id "$CHAINID" \
  --moniker "node0" \
  --commission-rate="0.10" \
  --commission-max-rate="0.20" \
  --commission-max-change-rate="0.01" \
  --min-self-delegation="1" \
  --keyring-backend test \
  --home "$NODE0_DIR"; then
  echo -e "${GREEN}Genesis transaction created successfully${NC}"
else
  echo -e "${RED}Genesis transaction failed, but continuing...${NC}"
fi

# Collect genesis transactions
echo -e "${YELLOW}Collecting genesis transactions${NC}"
if "$BINARY" genesis collect-gentxs --home "$NODE0_DIR"; then
  echo -e "${GREEN}Genesis transactions collected successfully${NC}"
else
  echo -e "${RED}Genesis collection failed, but continuing...${NC}"
fi

# Genesis is already properly configured from our template

# Validate genesis
echo -e "${YELLOW}Validating genesis${NC}"
"$BINARY" genesis validate --home "$NODE0_DIR" || echo -e "${YELLOW}Warning: Genesis validation failed, continuing anyway${NC}"

# Initialize remaining nodes as regular nodes (not validators)
echo -e "${YELLOW}Initializing remaining $((NODES-1)) nodes as regular nodes${NC}"
for i in $(seq 1 $((NODES-1))); do
  NODE_DIR="$BASE_DIR/node$i"
  mkdir -p "$NODE_DIR"
  
  # Initialize node
  "$BINARY" init "node$i" --chain-id "$CHAINID" --home "$NODE_DIR" --overwrite > /dev/null 2>&1
  
  # Create keys for potential future validator usage (but don't add to genesis yet)
  "$BINARY" keys add "validator$i" \
    --keyring-backend test --home "$NODE_DIR" > /dev/null 2>&1 || true
done

# Copy the genesis with single bootstrap validator to all nodes
echo -e "${YELLOW}Copying genesis to all nodes${NC}"
for i in $(seq 1 $((NODES-1))); do
  cp "$NODE0_DIR/config/genesis.json" "$BASE_DIR/node$i/config/genesis.json"
done

# Set up client configuration for each node
echo -e "${YELLOW}Setting up client configuration${NC}"
for i in $(seq 0 $((NODES-1))); do
  NODE_DIR="$BASE_DIR/node$i"
  RPC_PORT=$((26657 + i))
  
  # Create client.toml for easier CLI usage
  cat > "$NODE_DIR/config/client.toml" << EOF
chain-id = "$CHAINID"
keyring-backend = "test"
node = "tcp://localhost:$RPC_PORT"
broadcast-mode = "sync"
EOF
done

echo -e "${GREEN}Network setup:${NC}"
echo -e "  - Node0: Bootstrap validator (will start producing blocks)"
echo -e "  - Node1-$((NODES-1)): Regular nodes (can become validators later via staking)"

# Set up persistent peers
echo -e "${YELLOW}Setting up peer connections${NC}"
for i in $(seq 0 $((NODES-1))); do
  PEER_LIST=""
  for j in $(seq 0 $((NODES-1))); do
    if [ $i -ne $j ]; then
      P2P_PORT=$((26656 + j))
      NODE_ID=$("$BINARY" cometbft show-node-id --home "$BASE_DIR/node$j")
      if [ -n "$PEER_LIST" ]; then
        PEER_LIST="$PEER_LIST,"
      fi
      PEER_LIST="$PEER_LIST$NODE_ID@127.0.0.1:$P2P_PORT"
    fi
  done
  
  # Update config with persistent peers (cross-platform sed)
  if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "s/persistent_peers = \"\"/persistent_peers = \"$PEER_LIST\"/" "$BASE_DIR/node$i/config/config.toml"
  else
    sed -i "s/persistent_peers = \"\"/persistent_peers = \"$PEER_LIST\"/" "$BASE_DIR/node$i/config/config.toml"
  fi
done

# Start nodes
echo -e "${YELLOW}Starting $NODES nodes${NC}"
for i in $(seq 0 $((NODES-1))); do
  NODE_DIR="$BASE_DIR/node$i"
  
  # Ports
  RPC_PORT=$((26657 + i))
  P2P_PORT=$((26656 + i))
  GRPC_PORT=$((9090 + i))
  API_PORT=$((1317 + i))
  JSON_PORT=$((8545 + i * 2))
  WS_PORT=$((8546 + i * 2))
  
  echo -e "  Starting node$i on ports RPC:$RPC_PORT JSON-RPC:$JSON_PORT WebSocket:$WS_PORT"
  
  # Start node with correct flags
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
    
  echo $! > "$NODE_DIR/node.pid"
  sleep 1
done

echo
echo -e "${GREEN}✅ Testnet started with $NODES nodes${NC}"
echo -e "${YELLOW}Chain ID: $CHAINID${NC}"
echo
echo "Endpoints:"
for i in $(seq 0 $((NODES-1))); do
  RPC_PORT=$((26657 + i))
  JSON_PORT=$((8545 + i * 2))
  WS_PORT=$((8546 + i * 2))
  echo "  Node$i: RPC=http://localhost:$RPC_PORT JSON-RPC=http://localhost:$JSON_PORT WebSocket=ws://localhost:$WS_PORT"
done
echo
echo "Stop: pkill -f 'shardeumd.*shardeum-testnet' or Ctrl+C"
echo "Logs: $BASE_DIR/node*/node.log"
echo
echo -e "${YELLOW}Network is running. Press Ctrl+C to stop all nodes.${NC}"

# Wait for interrupt signal
wait