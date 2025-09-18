#!/bin/bash

set -e

# Resolve repo root regardless of where the script is invoked from
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

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
BASE_DIR="$REPO_ROOT/.testnet"
MIN_GAS="0.000006shm"
BINARY="${BINARY:-$REPO_ROOT/build/shardeumd}"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}Starting $NODES node Shardeum testnet${NC}"

# Clean up
rm -rf "$BASE_DIR"
pkill -f "shardeumd.*shardeum-testnet" || true

# Build our own binary locally (skip if called from makefile)
if [ "$SKIP_BUILD" != "1" ]; then
  echo -e "${YELLOW}Building shardeumd binary${NC}"
  (cd "$REPO_ROOT" && make build)
fi

# Verify binary exists (build again if needed when SKIP_BUILD was set externally)
if [ ! -f "$BINARY" ]; then
  echo -e "${YELLOW}Binary not found at $BINARY, attempting to build...${NC}"
  (cd "$REPO_ROOT" && make build)
fi
if [ ! -f "$BINARY" ]; then
  echo -e "${RED}Error: Binary not found at $BINARY after build${NC}"
  echo "Make sure 'make build' completed successfully or set BINARY=/absolute/path/to/shardeumd"
  exit 1
fi

# Verify genesis file exists
if [ ! -f "$REPO_ROOT/config/genesis.json" ]; then
  echo -e "${RED}Error: Genesis file not found at $REPO_ROOT/config/genesis.json${NC}"
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
echo -e "${YELLOW}Using custom genesis with shm denomination for Cosmos, ashm for EVM${NC}"
cp "$REPO_ROOT/config/genesis.json" "$NODE0_DIR/config/genesis.json"

# Create validator key for node0 only
echo -e "${YELLOW}Creating validator key for primary node${NC}"
"$BINARY" keys add "validator" \
  --keyring-backend test --home "$NODE0_DIR" > /dev/null 2>&1 || true

# Add dev accounts to keyring and genesis
echo -e "${YELLOW}Adding dev accounts to keyring${NC}"

# dev0 account
echo "copper push brief egg scan entry inform record adjust fossil boss egg comic alien upon aspect dry avoid interest fury window hint race symptom" | "$BINARY" keys add "dev0" \
  --keyring-backend test --home "$NODE0_DIR" --recover --algo eth_secp256k1 > /dev/null 2>&1 || true

# dev1 account  
echo "maximum display century economy unlock van census kite error heart snow filter midnight usage egg venture cash kick motor survey drastic edge muffin visual" | "$BINARY" keys add "dev1" \
  --keyring-backend test --home "$NODE0_DIR" --recover --algo eth_secp256k1 > /dev/null 2>&1 || true

# dev2 account
echo "will wear settle write dance topic tape sea glory hotel oppose rebel client problem era video gossip glide during yard balance cancel file rose" | "$BINARY" keys add "dev2" \
  --keyring-backend test --home "$NODE0_DIR" --recover --algo eth_secp256k1 > /dev/null 2>&1 || true

# dev3 account
echo "doll midnight silk carpet brush boring pluck office gown inquiry duck chief aim exit gain never tennis crime fragile ship cloud surface exotic patch" | "$BINARY" keys add "dev3" \
  --keyring-backend test --home "$NODE0_DIR" --recover --algo eth_secp256k1 > /dev/null 2>&1 || true

# Add validator account to genesis  
echo -e "${YELLOW}Adding validator account to genesis${NC}"
"$BINARY" genesis add-genesis-account "validator" 3000000000000000000000000ashm \
  --keyring-backend test --home "$NODE0_DIR" > /dev/null 2>&1

# Add dev accounts to genesis with test balances
echo -e "${YELLOW}Adding dev accounts to genesis${NC}"
"$BINARY" genesis add-genesis-account "dev0" 10000000shm \
  --keyring-backend test --home "$NODE0_DIR" > /dev/null 2>&1

"$BINARY" genesis add-genesis-account "dev1" 10000000shm \
  --keyring-backend test --home "$NODE0_DIR" > /dev/null 2>&1

"$BINARY" genesis add-genesis-account "dev2" 10000000shm \
  --keyring-backend test --home "$NODE0_DIR" > /dev/null 2>&1

"$BINARY" genesis add-genesis-account "dev3" 10000000shm \
  --keyring-backend test --home "$NODE0_DIR" > /dev/null 2>&1

# Create genesis transaction for validator
echo -e "${YELLOW}Creating genesis transaction for validator${NC}"
"$BINARY" genesis gentx "validator" 2000000000000000000000000ashm \
  --chain-id "$CHAINID" \
  --moniker "node0" \
  --commission-rate="0.10" \
  --commission-max-rate="0.20" \
  --commission-max-change-rate="0.01" \
  --min-self-delegation="1" \
  --keyring-backend test \
  --home "$NODE0_DIR"

# Collect genesis transaction
echo -e "${YELLOW}Collecting genesis transaction${NC}"
"$BINARY" genesis collect-gentxs --home "$NODE0_DIR"

# Genesis is already properly configured from our template

# Validate genesis
echo -e "${YELLOW}Validating genesis${NC}"
"$BINARY" genesis validate --home "$NODE0_DIR" || echo -e "${YELLOW}Warning: Genesis validation failed, continuing anyway${NC}"

# Initialize other nodes and copy genesis
if [ $NODES -gt 1 ]; then
  echo -e "${YELLOW}Initializing other nodes${NC}"
  for i in $(seq 1 $((NODES-1))); do
    NODE_DIR="$BASE_DIR/node$i"
    "$BINARY" init "node$i" --chain-id "$CHAINID" --home "$NODE_DIR" --overwrite > /dev/null 2>&1
    cp "$NODE0_DIR/config/genesis.json" "$NODE_DIR/config/genesis.json"
  done
fi

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

# Configure API server for each node
echo -e "${YELLOW}Configuring API servers${NC}"
for i in $(seq 0 $((NODES-1))); do
  NODE_DIR="$BASE_DIR/node$i"
  API_PORT=$((1317 + i))
  
  # Enable API server and set correct address in app.toml
  if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "s/enable = false/enable = true/" "$NODE_DIR/config/app.toml"
    sed -i '' "s/address = \"tcp:\/\/localhost:1317\"/address = \"tcp:\/\/127.0.0.1:$API_PORT\"/" "$NODE_DIR/config/app.toml"
  else
    sed -i "s/enable = false/enable = true/" "$NODE_DIR/config/app.toml"
    sed -i "s/address = \"tcp:\/\/localhost:1317\"/address = \"tcp:\/\/127.0.0.1:$API_PORT\"/" "$NODE_DIR/config/app.toml"
  fi
  
  echo "Node $i API server enabled on port $API_PORT"
done

echo -e "${GREEN}Network setup:${NC}"
echo -e "  - Node0 is the sole validator"
echo -e "  - Other nodes are regular full nodes"

# Set up seed-based peer discovery
echo -e "${YELLOW}Setting up seed-based peer discovery${NC}"

# Node0 will be the seed node - no seeds needed for it
NODE0_ID=$("$BINARY" cometbft show-node-id --home "$BASE_DIR/node0")
SEED_ADDRESS="$NODE0_ID@127.0.0.1:27656"

echo -e "${YELLOW}Using node0 as seed: $SEED_ADDRESS${NC}"

# Configure all nodes for local development
for i in $(seq 0 $((NODES-1))); do
  if [[ "$OSTYPE" == "darwin"* ]]; then
    # Allow duplicate IPs for local development
    sed -i '' "s/allow_duplicate_ip = false/allow_duplicate_ip = true/" "$BASE_DIR/node$i/config/config.toml"
    
    # Configure seeds for non-seed nodes
    if [ $i -ne 0 ]; then
      sed -i '' "s/seeds = \"\"/seeds = \"$SEED_ADDRESS\"/" "$BASE_DIR/node$i/config/config.toml"
    fi
    
    # Ensure persistent_peers stays empty for regular nodes
    sed -i '' "s/persistent_peers = \".*\"/persistent_peers = \"\"/" "$BASE_DIR/node$i/config/config.toml"
  else
    # Allow duplicate IPs for local development
    sed -i "s/allow_duplicate_ip = false/allow_duplicate_ip = true/" "$BASE_DIR/node$i/config/config.toml"
    
    # Configure seeds for non-seed nodes
    if [ $i -ne 0 ]; then
      sed -i "s/seeds = \"\"/seeds = \"$SEED_ADDRESS\"/" "$BASE_DIR/node$i/config/config.toml"
    fi
    
    # Ensure persistent_peers stays empty for regular nodes
    sed -i "s/persistent_peers = \".*\"/persistent_peers = \"\"/" "$BASE_DIR/node$i/config/config.toml"
  fi
  
  if [ $i -eq 0 ]; then
    echo "Node $i configured as seed node (allow_duplicate_ip=true)"
  else
    echo "Node $i using seed: $SEED_ADDRESS (allow_duplicate_ip=true)"
  fi
done

echo -e "${GREEN}Seed configuration complete:${NC}"
echo -e "  - Node0: Seed node (no seeds configured)"
echo -e "  - Node1-$(($NODES-1)): Use node0 as seed for peer discovery"

# Start nodes
echo -e "${YELLOW}Starting $NODES nodes${NC}"
for i in $(seq 0 $((NODES-1))); do
  NODE_DIR="$BASE_DIR/node$i"
  
  # Ports
  RPC_PORT=$((26657 + i))
  P2P_PORT=$((27656 + i))
  GRPC_PORT=$((9090 + i))
  API_PORT=$((1317 + i))
  JSON_PORT=$((8545 + i * 2))
  WS_PORT=$((8546 + i * 2))
  
  echo -e "  Starting node$i on ports RPC:$RPC_PORT API:$API_PORT JSON-RPC:$JSON_PORT WebSocket:$WS_PORT"
  
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
  sleep 3  # Give nodes more time to start and bind ports
done

echo
echo -e "${GREEN}✅ Testnet started with $NODES nodes${NC}"
echo -e "${YELLOW}Chain ID: $CHAINID${NC}"
echo
echo "Endpoints:"
for i in $(seq 0 $((NODES-1))); do
  RPC_PORT=$((26657 + i))
  API_PORT=$((1317 + i))
  JSON_PORT=$((8545 + i * 2))
  WS_PORT=$((8546 + i * 2))
  echo "  Node$i: RPC=http://localhost:$RPC_PORT API=http://localhost:$API_PORT JSON-RPC=http://localhost:$JSON_PORT WebSocket=ws://localhost:$WS_PORT"
done
echo
echo "Stop: pkill -f 'shardeumd.*shardeum-testnet' or Ctrl+C"
echo "Logs: $BASE_DIR/node*/node.log"
echo
echo "Add more nodes: ./scripts/add_node.sh <node_id>"
echo
echo -e "${GREEN}Dev accounts available for testing:${NC}"
echo "  dev0: 0xC6Fe5D33615a1C52c08018c47E8Bc53646A0E101 | cosmos1cml96vmptgw99syqrrz8az79xer2pcgp84pdun"
echo "  dev1: 0x963EBDf2e1f8DB8707D05FC75bfeFFBa1B5BaC17 | cosmos1jcltmuhplrdcwp7stlr4hlhlhgd4htqh3a79sq"
echo "  dev2: 0x40a0cb1C63e026A81B55EE1308586E21eec1eFa9 | cosmos1gzsvk8rruqn2sx64acfsskrwy8hvrmafqkaze8"
echo "  dev3: 0x498B5AeC5D439b733dC2F58AB489783A23FB26dA | cosmos1fx944mzagwdhx0wz7k9tfztc8g3lkfk6rrgv6l"
echo "  Each account has 10,000,000 shm tokens for testing (via Cosmos) and equivalent ashm via EVM"
echo
echo -e "${YELLOW}Network is running. Press Ctrl+C to stop all nodes.${NC}"

# Wait for interrupt signal
wait