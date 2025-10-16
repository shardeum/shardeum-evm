#!/bin/bash
set -e

# -----------------------------
# Colors (define early; used everywhere)
# -----------------------------
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# -----------------------------
# Resolve repo root regardless of where the script is invoked from
# -----------------------------
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CURRENT_DIR="$(pwd)"

# Source genesis utilities from unified script
source "$SCRIPT_DIR/genesis_account_split.sh"

# -----------------------------
# Usage
# -----------------------------
usage() {
  echo "Usage: $0 [num_nodes] [options]"
  echo ""
  echo "Options:"
  echo "  -n, --nodes NUM               Number of nodes to start (default: 4)"
  echo "  -g, --genesis FILE            Path to custom genesis file (optional)"
  echo "      --no-dev-accounts         Do not add dev accounts to genesis"
  echo "      --network <name>          Network to use (mainnet, testnet, devnet, local)"
  echo "      --chain-id <id>           Override chain ID"
  echo "      --create2-factory         Deploy create2 factory to genesis (default: false)"
  echo "      --allow-unprotected-txs   Enable unprotected transactions (default: false)"
  echo "  -h, --help                    Show this help message"
  echo ""
  echo "Environment variables:"
  echo "  SHARDEUM_NETWORK              Network to use (overrides --network)"
  echo "  SHARDEUM_CHAIN_ID             Chain ID to use (overrides --chain-id)"
  echo "  SHARDEUM_CONFIG_DIR           Absolute path to directory containing config/environments/*.json"
  echo "  BINARY                        Path to shardeumd binary"
  echo "  SKIP_BUILD=1                  Skip 'make build' if binary already exists"
  echo ""
  echo "Examples:"
  echo "  $0 4 --network testnet"
  echo "  $0 6 --network devnet --chain-id shardeum-dev-1"
  echo "  SHARDEUM_NETWORK=mainnet $0 4"
  echo "  SHARDEUM_CONFIG_DIR=/path/to/config SHARDEUM_NETWORK=testnet $0 4"
  echo "  $0 -g ./genesis.json                               		# Start 4 nodes with custom genesis"
  echo "  $0 6 ./genesis.json                                 		# Legacy: 6 nodes + custom genesis"
  echo "  $0 --create2-factory                                		# Include create2 factory in genesis"
  echo "  $0 --allow-unprotected-txs                          		# Allow unprotected (non-EIP155) txs"
  exit 1
}

# -----------------------------
# Cleanup function for graceful shutdown
# -----------------------------
cleanup() {
  echo -e "\n${YELLOW}Shutting down nodes...${NC}"
  pkill -f "shardeumd.*$CHAINID" || true
  exit 0
}
trap cleanup SIGINT SIGTERM

# -----------------------------
# Parse command line arguments (merged flags)
# -----------------------------
NODES=""
GENESIS_FILE=""
ADD_DEV_ACCOUNTS="true"
NETWORK=""
CUSTOM_CHAIN_ID=""
DEPLOY_CREATE2_FACTORY="false"
ALLOW_UNPROTECTED_TXS="false"

while [[ $# -gt 0 ]]; do
  case $1 in
    -n|--nodes)
      NODES="$2"; shift 2 ;;
    -g|--genesis)
      GENESIS_FILE="$2"; shift 2 ;;
    --no-dev-accounts)
      ADD_DEV_ACCOUNTS="false"; shift ;;
    --network)
      NETWORK="$2"; shift 2 ;;
    --chain-id)
      CUSTOM_CHAIN_ID="$2"; shift 2 ;;
    --create2-factory)
      DEPLOY_CREATE2_FACTORY="true"; shift ;;
    --allow-unprotected-txs)
      ALLOW_UNPROTECTED_TXS="true"; shift ;;
    -h|--help)
      usage ;;
    -* )
      echo "Unknown option: $1"; usage ;;
    *)
      # Legacy positional: first numeric is NODES, second is GENESIS
      if [[ -z "${NODES_SET:-}" && "$1" =~ ^[0-9]+$ ]]; then
        NODES="$1"; NODES_SET=1; shift
      elif [[ -z "$GENESIS_FILE" && -f "$1" ]]; then
        GENESIS_FILE="$1"; shift
      else
        echo "Unknown argument: $1"; usage
      fi
      ;;
  esac
done

# config dir is compulsory - can't start network without it
if [[ -z "$SHARDEUM_CONFIG_DIR" ]]; then
  echo "Error: SHARDEUM_CONFIG_DIR is required"
  usage
fi

# Defaults
NODES="${NODES:-4}"
NETWORK="${SHARDEUM_NETWORK:-${NETWORK:-testnet}}"

# -----------------------------
# Load network configuration
# -----------------------------
CONFIG_FILE="$SHARDEUM_CONFIG_DIR/environments/$NETWORK.json"
if [ ! -f "$CONFIG_FILE" ]; then
  echo -e "${RED}Error: Network configuration file not found: $CONFIG_FILE${NC}"
  echo "Available networks:"
  ls -1 "$SHARDEUM_CONFIG_DIR/environments"/*.json 2>/dev/null | xargs -n1 basename | sed 's/.json$//' | sed 's/^/  /' || echo "  No network configurations found"
  exit 1
fi

# Read network configuration (expects keys: chain_id, evm_chain_id, base_denom)
CHAINID=$(jq -r '.chain_id' "$CONFIG_FILE")
EVM_CHAIN_ID=$(jq -r '.evm_chain_id' "$CONFIG_FILE")
BASE_DENOM=$(jq -r '.base_denom' "$CONFIG_FILE")

# Overrides
if [[ -n "$CUSTOM_CHAIN_ID" ]]; then
  CHAINID="$CUSTOM_CHAIN_ID"
elif [[ -n "$SHARDEUM_CHAIN_ID" ]]; then
  CHAINID="$SHARDEUM_CHAIN_ID"
fi

# Paths & Binary
BINARY="${BINARY:-shardeumd}"
BASE_DIR="${HOME:-$CURRENT_DIR}/.$NETWORK"
if [[ "$NETWORK" == "local" ]]; then
  BASE_DIR="$CURRENT_DIR/.$NETWORK"
fi
MIN_GAS="2048130280389041$BASE_DENOM"

# -----------------------------
# Validate nodes number
# -----------------------------
if ! [[ "$NODES" =~ ^[0-9]+$ ]] || [[ "$NODES" -lt 1 ]]; then
  echo -e "${RED}Error: Number of nodes must be a positive integer${NC}"
  exit 1
fi

# Ensure the binary can resolve configs via SHARDEUM_CONFIG_DIR
export SHARDEUM_CONFIG_DIR="$SHARDEUM_CONFIG_DIR"

# -----------------------------
# Announce
# -----------------------------
if [ -n "$GENESIS_FILE" ]; then
  echo -e "${GREEN}Starting $NODES node Shardeum ${NETWORK} network with custom genesis${NC}"
else
  echo -e "${GREEN}Starting $NODES node Shardeum ${NETWORK} network${NC}"
fi
echo -e "${YELLOW}Network: $NETWORK${NC}"
echo -e "${YELLOW}Chain ID: $CHAINID${NC}"
echo -e "${YELLOW}EVM Chain ID: $EVM_CHAIN_ID${NC}"
echo -e "${YELLOW}Base Denomination: $BASE_DENOM${NC}"

# -----------------------------
# Clean up any previous run
# -----------------------------
rm -rf "$BASE_DIR"
pkill -f "shardeumd.*$CHAINID" || true

# -----------------------------
# Build binary (unless skipped)
# -----------------------------
if [ "$SKIP_BUILD" != "1" ]; then
  if [[ "$NETWORK" == "local" ]]; then
    echo -e "${YELLOW}Building shardeumd binary${NC}"
    (cd "$REPO_ROOT" && make install)
    BINARY="$(command -v shardeumd)"
  fi
fi

# Verify binary
if [ ! -f "$BINARY" ]; then
  echo -e "${RED}Error: Binary not found at $BINARY${NC}"
  echo -e "${RED}It's compulsory for non-local network to have 'shardeumd' in PATH or BINARY is set to /absolute/path/to/shardeumd. For local network, Make sure 'make install' completed successfully${NC}"
  exit 1
else
    echo "will use binary:" $BINARY
fi

# -----------------------------
# Determine which genesis file to use
# -----------------------------
if [ -n "$GENESIS_FILE" ]; then
  if [ ! -f "$GENESIS_FILE" ]; then
    echo -e "${RED}Error: Custom genesis file not found at $GENESIS_FILE${NC}"
    exit 1
  fi
  GENESIS_TO_USE="$GENESIS_FILE"
  echo -e "${YELLOW}Using custom genesis file: $GENESIS_FILE${NC}"
else
  # Check for split genesis files first (pattern: {network}-genesis.genesis.json)
  SPLIT_GENESIS_PATH="$SHARDEUM_CONFIG_DIR/environments/${NETWORK}-genesis.genesis.json"
  
  # Also check for monolithic genesis file (pattern: {network}-genesis.json)
  MONOLITHIC_GENESIS_PATH="$SHARDEUM_CONFIG_DIR/environments/${NETWORK}-genesis.json"
  
  if [ -f "$SPLIT_GENESIS_PATH" ]; then
    # Use split genesis file (base file without accounts)
    GENESIS_TO_USE="$SPLIT_GENESIS_PATH"
    echo -e "${YELLOW}Using network split genesis file: $GENESIS_TO_USE${NC}"
  elif [ -f "$MONOLITHIC_GENESIS_PATH" ]; then
    # Use monolithic genesis file
    GENESIS_TO_USE="$MONOLITHIC_GENESIS_PATH"
    echo -e "${YELLOW}Using network genesis file: $GENESIS_TO_USE${NC}"
  else
    # Fallback to generic genesis.json if network-specific doesn't exist
    GENESIS_PATH="$REPO_ROOT/config/genesis.json"
    if [ ! -f "$GENESIS_PATH" ]; then
      echo -e "${RED}Error: No genesis file found for network '${NETWORK}'${NC}"
      echo "Looked for:"
      echo "  - $SPLIT_GENESIS_PATH (split genesis)"
      echo "  - $MONOLITHIC_GENESIS_PATH (monolithic genesis)"
      echo "  - $GENESIS_PATH (fallback)"
      echo "Make sure you have the proper Shardeum network configuration"
      exit 1
    fi
    GENESIS_TO_USE="$GENESIS_PATH"
    echo -e "${YELLOW}Using fallback genesis file: $GENESIS_TO_USE${NC}"
  fi
fi

# -----------------------------
# Initialize primary node
# -----------------------------
echo -e "${YELLOW}Initializing primary node${NC}"
NODE0_DIR="$BASE_DIR/node0"
mkdir -p "$NODE0_DIR"

# Environment for binary
export SHARDEUM_NETWORK="$NETWORK"
export SHARDEUM_CHAIN_ID="$CHAINID"

# Init to produce template
"$BINARY" init "node0" --chain-id "$CHAINID" --home "$NODE0_DIR" --overwrite > /dev/null 2>&1

# Replace with selected genesis (handle split accounts if present)
echo -e "${YELLOW}Placing selected genesis at node0${NC}"

# Check if genesis has split account files
if has_split_accounts "$GENESIS_TO_USE"; then
  echo -e "${YELLOW}Detected split account files, merging...${NC}"
  MERGED_GENESIS=$(load_genesis_with_accounts "$GENESIS_TO_USE" "$NODE0_DIR/config/genesis.json")
  if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to merge genesis accounts${NC}"
    exit 1
  fi
  # If output was different from target, copy it
  if [ "$MERGED_GENESIS" != "$NODE0_DIR/config/genesis.json" ]; then
    cp "$MERGED_GENESIS" "$NODE0_DIR/config/genesis.json"
    rm -f "$MERGED_GENESIS"  # Clean up temp file
  fi
  ACCOUNT_COUNT=$(jq '.app_state.auth.accounts | length' "$NODE0_DIR/config/genesis.json")
  echo -e "${GREEN}Merged ${ACCOUNT_COUNT} accounts into genesis${NC}"
else
  cp "$GENESIS_TO_USE" "$NODE0_DIR/config/genesis.json"
fi

# -----------------------------
# Keys: validator + optional dev accounts
# -----------------------------
echo -e "${YELLOW}Creating validator key for primary node${NC}"
"$BINARY" keys add "validator" --keyring-backend test --home "$NODE0_DIR" > /dev/null 2>&1 || true

if [ "$ADD_DEV_ACCOUNTS" = "true" ]; then
  # Add dev accounts to keyring only (no balances in genesis)
  echo -e "${YELLOW}Adding dev accounts to keyring${NC}"
  echo "copper push brief egg scan entry inform record adjust fossil boss egg comic alien upon aspect dry avoid interest fury window hint race symptom" | "$BINARY" keys add "dev0" --keyring-backend test --home "$NODE0_DIR" --recover --algo eth_secp256k1 > /dev/null 2>&1 || true
  echo "maximum display century economy unlock van census kite error heart snow filter midnight usage egg venture cash kick motor survey drastic edge muffin visual" | "$BINARY" keys add "dev1" --keyring-backend test --home "$NODE0_DIR" --recover --algo eth_secp256k1 > /dev/null 2>&1 || true
  echo "will wear settle write dance topic tape sea glory hotel oppose rebel client problem era video gossip glide during yard balance cancel file rose" | "$BINARY" keys add "dev2" --keyring-backend test --home "$NODE0_DIR" --recover --algo eth_secp256k1 > /dev/null 2>&1 || true
  echo "doll midnight silk carpet brush boring pluck office gown inquiry duck chief aim exit gain never tennis crime fragile ship cloud surface exotic patch" | "$BINARY" keys add "dev3" --keyring-backend test --home "$NODE0_DIR" --recover --algo eth_secp256k1 > /dev/null 2>&1 || true
fi

# -----------------------------
# Fund accounts in genesis
# -----------------------------
echo -e "${YELLOW}Adding validator account to genesis${NC}"
"$BINARY" genesis add-genesis-account "validator" 10000000000000000000000${BASE_DENOM} \
  --keyring-backend test --home "$NODE0_DIR" > /dev/null 2>&1

# -----------------------------
# Optional: Deploy create2 factory & unprotected txs
# -----------------------------
if [ "$DEPLOY_CREATE2_FACTORY" = "true" ]; then
  echo -e "${YELLOW}Deploying create2 factory to genesis${NC}"
  create2_addr="4e59b44847b379578588920ca78fbf26c0b4956c"
  create2_code="7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe03601600081602082378035828234f58015156039578182fd5b8082525050506014600cf3"

  # Add account with 0 balance
  "$BINARY" genesis add-genesis-account "$("$BINARY" keys parse "$create2_addr" --output json | jq -r '.formats[0]')" 0${BASE_DENOM} --home "$NODE0_DIR" > /dev/null 2>&1

  TMP_GENESIS="$NODE0_DIR/config/genesis_tmp.json"

  # Set nonce/sequence = 1
  jq --arg addr "0x$create2_addr" \
     '(.app_state.auth.accounts[] | select(.address == ($addr | ascii_downcase)) | .sequence) = "1"' \
     "$NODE0_DIR/config/genesis.json" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$NODE0_DIR/config/genesis.json"

  # Add EVM account with code
  jq --arg addr "0x$create2_addr" --arg code "$create2_code" \
     '.app_state.evm.accounts += [{"address": $addr, "code": $code, "storage": []}]' \
     "$NODE0_DIR/config/genesis.json" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$NODE0_DIR/config/genesis.json"
fi

if [ "$ALLOW_UNPROTECTED_TXS" = "true" ]; then
  echo -e "${YELLOW}Enabling unprotected transactions in genesis${NC}"
  TMP_GENESIS="$NODE0_DIR/config/genesis_tmp.json"
  jq '.app_state.evm.params.allow_unprotected_txs = true' \
     "$NODE0_DIR/config/genesis.json" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$NODE0_DIR/config/genesis.json"
fi

# -----------------------------
# Gentx + collect
# -----------------------------
echo -e "${YELLOW}Creating genesis transaction for validator${NC}"

# Set network-specific gas fees to meet minimum global fee requirement
case "$NETWORK" in
  "mainnet")
    GENTX_FEES="2100000000000000000000$BASE_DENOM"   # 2100 SHM (exceeds minimum global fee requirement)
    ;;
  "testnet"|"devnet"|"local")
    GENTX_FEES="2100000000000000000000$BASE_DENOM"   # 2100 SHM (exceeds minimum global fee requirement)
    ;;
  *)
    GENTX_FEES="2100000000000000000000$BASE_DENOM"   # Default to 2100 SHM
    ;;
esac

echo -e "${YELLOW}Using network-specific fees: $GENTX_FEES${NC}"

# Check if genesis has custom initial_height to handle gentx signing properly
INITIAL_HEIGHT=$(jq -r '.initial_height' "$NODE0_DIR/config/genesis.json" 2>/dev/null || echo "1")
echo -e "${YELLOW}Genesis initial height: $INITIAL_HEIGHT${NC}"

# For custom initial heights, we may need to set account-number explicitly
if [ "$INITIAL_HEIGHT" != "1" ] && [ "$INITIAL_HEIGHT" != "0" ]; then
  echo -e "${YELLOW}Custom initial height detected, using explicit account number${NC}"
  
  # Get the validator's account number from genesis
  VALIDATOR_ADDR=$("$BINARY" keys show validator --keyring-backend test --home "$NODE0_DIR" --address)
  ACCOUNT_NUM=$(jq -r ".app_state.auth.accounts[] | select(.address == \"$VALIDATOR_ADDR\") | .account_number" "$NODE0_DIR/config/genesis.json" 2>/dev/null || echo "")
  
  if [ -n "$ACCOUNT_NUM" ] && [ "$ACCOUNT_NUM" != "null" ]; then
    echo -e "${YELLOW}Using account number: $ACCOUNT_NUM${NC}"
    "$BINARY" genesis gentx "validator" 1000000000000000000${BASE_DENOM} \
      --chain-id "$CHAINID" \
      --moniker "node0" \
      --commission-rate="0.10" \
      --commission-max-rate="0.20" \
      --commission-max-change-rate="0.01" \
      --min-self-delegation="1" \
      --fees="$GENTX_FEES" \
      --keyring-backend test \
      --account-number "$ACCOUNT_NUM" \
      --sequence "0" \
      --gas "1000000" \
      --offline \
      --home "$NODE0_DIR"
  else
    echo -e "${YELLOW}Could not determine account number, trying without explicit params${NC}"
    "$BINARY" genesis gentx "validator" 1000000000000000000${BASE_DENOM} \
      --chain-id "$CHAINID" \
      --moniker "node0" \
      --commission-rate="0.10" \
      --commission-max-rate="0.20" \
      --commission-max-change-rate="0.01" \
      --min-self-delegation="1" \
      --fees="$GENTX_FEES" \
      --keyring-backend test \
      --gas "1000000" \
      --home "$NODE0_DIR"
  fi
else
  # Standard gentx for height 1
  "$BINARY" genesis gentx "validator" 1000000000000000000${BASE_DENOM} \
    --chain-id "$CHAINID" \
    --moniker "node0" \
    --commission-rate="0.10" \
    --commission-max-rate="0.20" \
    --commission-max-change-rate="0.01" \
    --min-self-delegation="1" \
    --fees="$GENTX_FEES" \
    --keyring-backend test \
    --gas "1000000" \
    --home "$NODE0_DIR"
fi

echo -e "${YELLOW}Collecting genesis transaction${NC}"
"$BINARY" genesis collect-gentxs --home "$NODE0_DIR"

echo -e "${YELLOW}Validating genesis${NC}"
"$BINARY" genesis validate --home "$NODE0_DIR" || echo -e "${YELLOW}Warning: Genesis validation failed, continuing anyway${NC}"

# -----------------------------
# Initialize other nodes
# -----------------------------
if [ "$NODES" -gt 1 ]; then
  echo -e "${YELLOW}Initializing other nodes${NC}"
  for i in $(seq 1 $((NODES-1))); do
    NODE_DIR="$BASE_DIR/node$i"
    "$BINARY" init "node$i" --chain-id "$CHAINID" --home "$NODE_DIR" --overwrite > /dev/null 2>&1
    cp "$NODE0_DIR/config/genesis.json" "$NODE_DIR/config/genesis.json"
  done
fi

# -----------------------------
# Client configuration
# -----------------------------
echo -e "${YELLOW}Setting up client configuration${NC}"
for i in $(seq 0 $((NODES-1))); do
  NODE_DIR="$BASE_DIR/node$i"
  RPC_PORT=$((26657 + i))
  cat > "$NODE_DIR/config/client.toml" << EOF
chain-id = "$CHAINID"
keyring-backend = "test"
node = "tcp://localhost:$RPC_PORT"
broadcast-mode = "sync"
EOF
done

# -----------------------------
# API servers
# -----------------------------
echo -e "${YELLOW}Configuring API servers${NC}"
for i in $(seq 0 $((NODES-1))); do
  NODE_DIR="$BASE_DIR/node$i"
  API_PORT=$((1317 + i))

  if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "s/enable = false/enable = true/" "$NODE_DIR/config/app.toml"
    sed -i '' "s#address = \"tcp://localhost:1317\"#address = \"tcp://127.0.0.1:$API_PORT\"#" "$NODE_DIR/config/app.toml"
  else
    sed -i "s/enable = false/enable = true/" "$NODE_DIR/config/app.toml"
    sed -i "s#address = \"tcp://localhost:1317\"#address = \"tcp://127.0.0.1:$API_PORT\"#" "$NODE_DIR/config/app.toml"
  fi
  echo "Node $i API server enabled on port $API_PORT"
done

echo -e "${GREEN}Network setup:${NC}"
echo -e "  - Node0 is the sole validator"
echo -e "  - Other nodes are regular full nodes"

# -----------------------------
# Seed-based peer discovery
# -----------------------------
echo -e "${YELLOW}Setting up seed-based peer discovery${NC}"
NODE0_ID=$("$BINARY" cometbft show-node-id --home "$BASE_DIR/node0")
SEED_ADDRESS="$NODE0_ID@127.0.0.1:27656"
echo -e "${YELLOW}Using node0 as seed: $SEED_ADDRESS${NC}"

for i in $(seq 0 $((NODES-1))); do
  CFG="$BASE_DIR/node$i/config/config.toml"

  if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "s/allow_duplicate_ip = false/allow_duplicate_ip = true/" "$CFG"
    if [ $i -ne 0 ]; then sed -i '' "s/seeds = \".*\"/seeds = \"$SEED_ADDRESS\"/" "$CFG"; fi
    sed -i '' "s/persistent_peers = \".*\"/persistent_peers = \"\"/" "$CFG"
  else
    sed -i "s/allow_duplicate_ip = false/allow_duplicate_ip = true/" "$CFG"
    if [ $i -ne 0 ]; then sed -i "s/seeds = \".*\"/seeds = \"$SEED_ADDRESS\"/" "$CFG"; fi
    sed -i "s/persistent_peers = \".*\"/persistent_peers = \"\"/" "$CFG"
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

# -----------------------------
# Start nodes
# -----------------------------
echo -e "${YELLOW}Starting $NODES nodes${NC}"
for i in $(seq 0 $((NODES-1))); do
  NODE_DIR="$BASE_DIR/node$i"

  RPC_PORT=$((26657 + i))
  P2P_PORT=$((27656 + i))
  GRPC_PORT=$((9090 + i))
  API_PORT=$((1317 + i))
  JSON_PORT=$((8545 + i * 2))
  WS_PORT=$((8546 + i * 2))

  # Build start command based on node type (node0 is validator, others are full nodes)
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

  if [ $i -eq 0 ]; then
    echo -e "  Starting node$i (validator) on ports RPC:$RPC_PORT API:$API_PORT"
  else
    # Other nodes are full nodes - enable JSON-RPC
    echo -e "  Starting node$i (full-node) on ports RPC:$RPC_PORT API:$API_PORT JSON-RPC:$JSON_PORT WebSocket:$WS_PORT"
    START_CMD+=(--json-rpc.enable)
    START_CMD+=(--json-rpc.address "127.0.0.1:$JSON_PORT")
    START_CMD+=(--json-rpc.ws-address "127.0.0.1:$WS_PORT")
    START_CMD+=(--json-rpc.api eth,txpool,personal,net,debug,web3)
  fi

  "${START_CMD[@]}" > "$NODE_DIR/node.log" 2>&1 &

  echo $! > "$NODE_DIR/node.pid"
  sleep 3
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
  
  if [ $i -eq 0 ]; then
    # Node0 is validator - no JSON-RPC
    echo "  Node$i (validator): RPC=http://localhost:$RPC_PORT API=http://localhost:$API_PORT"
  else
    # Other nodes are full nodes - show JSON-RPC
    echo "  Node$i (full-node): RPC=http://localhost:$RPC_PORT API=http://localhost:$API_PORT JSON-RPC=http://localhost:$JSON_PORT WebSocket=ws://localhost:$WS_PORT"
  fi
done
echo
echo "Stop: pkill -f 'shardeumd.*$CHAINID' or Ctrl+C"
echo -e "${GREEN}Network: $NETWORK${NC}"
echo -e "${GREEN}Chain ID: $CHAINID${NC}"
echo -e "${GREEN}EVM Chain ID: $EVM_CHAIN_ID${NC}"
echo -e "${GREEN}Node Data Directory: $BASE_DIR${NC}"
echo -e "${GREEN}Logs: $BASE_DIR/node*/node.log${NC}"
echo
echo "Add more nodes: ./scripts/add_node.sh <node_id>"
echo
echo -e "${GREEN}Preset accounts from genesis file funded for testing!${NC}"
echo
echo -e "${YELLOW}Network is running. Press Ctrl+C to stop all nodes.${NC}"

# Wait for interrupt signal
wait
