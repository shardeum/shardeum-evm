#!/bin/bash

set -e

CURRENT_DIR="$(pwd)"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Source genesis utilities from unified script
source "$SCRIPT_DIR/genesis_account_split.sh"

# Parse command line arguments
NODE_ID=""
SEED_NODE_RPC=""
NETWORK="local"
CUSTOM_CHAIN_ID=""
NODE_TYPE="validator"
MONIKER=""
WEBSITE=""
IDENTITY=""
SECURITY=""
DETAILS=""

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
  echo "  --moniker <name>     Custom moniker for the node (default: <node_id>)"
  echo "  --website <url>      Website URL for validator"
  echo "  --identity <id>      Keybase identity for validator verification"
  echo "  --security <email>   Security contact email"
  echo "  --details <text>     Additional details/description for validator"
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
  echo "  $0 node7 --moniker 'My Validator' --website https://mysite.com --details 'Production validator'"
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
    --moniker)
      MONIKER="$2"
      shift 2
      ;;
    --website)
      WEBSITE="$2"
      shift 2
      ;;
    --identity)
      IDENTITY="$2"
      shift 2
      ;;
    --security)
      SECURITY="$2"
      shift 2
      ;;
    --details)
      DETAILS="$2"
      shift 2
      ;;
    --help)
      usage
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
MONIKER="${MONIKER:-$NODE_ID}"

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

# Function to fetch genesis using chunked API
fetch_genesis_chunked() {
  local RPC_URL="$1"
  local OUTPUT_FILE="$2"

  echo -e "${YELLOW}Fetching genesis metadata and chunk 0...${NC}"

  # Create temporary file to store combined chunks
  local TEMP_COMBINED="/tmp/genesis_combined_$$.txt"
  rm -f "$TEMP_COMBINED"  # Ensure clean start

  # Fetch chunk 0 to get total count AND the first chunk data with retry
  local TEMP_CHUNK_0="/tmp/chunk_0_$$.json"
  local CHUNK_0_RETRIES=3
  local CHUNK_0_ATTEMPT=0
  local CHUNK_0_SUCCESS=false

  while [ $CHUNK_0_ATTEMPT -lt $CHUNK_0_RETRIES ] && [ "$CHUNK_0_SUCCESS" = false ]; do
    CHUNK_0_ATTEMPT=$((CHUNK_0_ATTEMPT + 1))
    echo -e "${YELLOW}Fetching chunk 0 (attempt $CHUNK_0_ATTEMPT/$CHUNK_0_RETRIES)...${NC}"

    # Use -C - to enable resume on partial transfers
    curl -s --max-time 600 --connect-timeout 30 -C - "$RPC_URL/genesis_chunked?chunk=0" -o "$TEMP_CHUNK_0"
    local CURL_EXIT=$?

    if [ $CURL_EXIT -eq 0 ] && [ -s "$TEMP_CHUNK_0" ]; then
      # Verify we can parse it
      if jq -r '.result.total' "$TEMP_CHUNK_0" > /dev/null 2>&1; then
        CHUNK_0_SUCCESS=true
        echo -e "${GREEN}Chunk 0 fetched successfully ($(stat -f%z "$TEMP_CHUNK_0" 2>/dev/null || stat -c%s "$TEMP_CHUNK_0") bytes)${NC}"
      else
        echo -e "${YELLOW}Warning: Chunk 0 downloaded but invalid JSON (attempt $CHUNK_0_ATTEMPT)${NC}"
        rm -f "$TEMP_CHUNK_0"
      fi
    else
      echo -e "${YELLOW}Warning: Failed to fetch chunk 0 (curl exit: $CURL_EXIT, attempt $CHUNK_0_ATTEMPT)${NC}"
      rm -f "$TEMP_CHUNK_0"

      if [ $CHUNK_0_ATTEMPT -lt $CHUNK_0_RETRIES ]; then
        echo -e "${YELLOW}Retrying in 3 seconds...${NC}"
        sleep 3
      fi
    fi
  done

  if [ "$CHUNK_0_SUCCESS" = false ]; then
    echo -e "${RED}Error: Failed to fetch chunk 0 after $CHUNK_0_RETRIES attempts${NC}"

    # Try regular genesis endpoint as fallback
    echo -e "${YELLOW}Trying regular genesis endpoint...${NC}"
    local TEMP_GENESIS="/tmp/genesis_full_$$.json"

    if curl -s --max-time 300 --connect-timeout 30 "$RPC_URL/genesis" 2>/dev/null | jq -r '.result.genesis // empty' > "$TEMP_GENESIS" 2>/dev/null; then
      # Check if the result is valid
      if [ -s "$TEMP_GENESIS" ] && [ "$(head -c 10 "$TEMP_GENESIS")" != "null" ] && [ "$(head -c 1 "$TEMP_GENESIS")" = "{" ]; then
        mv "$TEMP_GENESIS" "$OUTPUT_FILE"
        echo -e "${GREEN}Successfully fetched genesis from regular endpoint${NC}"
        return 0
      fi
    fi

    rm -f "$TEMP_GENESIS"
    echo -e "${RED}Error: Could not fetch genesis from either endpoint${NC}"
    return 1
  fi

  # Extract total chunks count
  local TOTAL_CHUNKS=$(jq -r '.result.total // empty' "$TEMP_CHUNK_0" 2>/dev/null)

  if [ -z "$TOTAL_CHUNKS" ] || ! [[ "$TOTAL_CHUNKS" =~ ^[0-9]+$ ]]; then
    echo -e "${RED}Error: Could not parse total chunks from response${NC}"
    rm -f "$TEMP_CHUNK_0" "$TEMP_COMBINED"
    return 1
  fi

  echo -e "${GREEN}Total chunks: $TOTAL_CHUNKS${NC}"

  # Extract data from chunk 0
  local CHUNK_0_DATA=$(jq -r '.result.data // empty' "$TEMP_CHUNK_0" 2>/dev/null)
  if [ -z "$CHUNK_0_DATA" ] || [ "$CHUNK_0_DATA" = "null" ]; then
    echo -e "${RED}Error: Could not extract data from chunk 0${NC}"
    rm -f "$TEMP_CHUNK_0" "$TEMP_COMBINED"
    return 1
  fi

  # Save chunk 0 data
  echo "$CHUNK_0_DATA" > "$TEMP_COMBINED"
  echo -e "${GREEN}Chunk 1/$TOTAL_CHUNKS fetched successfully ($(stat -f%z "$TEMP_CHUNK_0" 2>/dev/null || stat -c%s "$TEMP_CHUNK_0") bytes)${NC}"
  rm -f "$TEMP_CHUNK_0"

  # Fetch remaining chunks (1 to TOTAL_CHUNKS-1)
  for i in $(seq 1 $((TOTAL_CHUNKS - 1))); do
    echo -e "${YELLOW}Fetching chunk $((i + 1))/$TOTAL_CHUNKS...${NC}"

    # Retry logic for fetching each chunk
    local MAX_RETRIES=3
    local RETRY_COUNT=0
    local CHUNK_SUCCESS=false

    while [ $RETRY_COUNT -lt $MAX_RETRIES ] && [ "$CHUNK_SUCCESS" = false ]; do
      # Fetch chunk with timeout and retry
      # Use --max-time for overall timeout (600s = 10 min for very large chunks)
      # Use --connect-timeout for connection timeout (30s)
      # Write to temp file to avoid memory issues with large responses
      local TEMP_CHUNK="/tmp/chunk_${i}_$$.json"

      curl -s --max-time 600 --connect-timeout 30 "$RPC_URL/genesis_chunked?chunk=$i" -o "$TEMP_CHUNK"
      local CURL_EXIT_CODE=$?

      if [ $CURL_EXIT_CODE -eq 0 ] && [ -s "$TEMP_CHUNK" ]; then
          # Try to parse the response and extract data field
          local PARSED_DATA=$(jq -r '.result.data // empty' "$TEMP_CHUNK" 2>/dev/null)
          local JQ_EXIT=$?

          if [ $JQ_EXIT -eq 0 ] && [ -n "$PARSED_DATA" ] && [ "$PARSED_DATA" != "null" ]; then
            # Append chunk to combined file (chunk 0 already written)
            echo "$PARSED_DATA" >> "$TEMP_COMBINED"

            # Verify chunk was written and has content
            if [ -s "$TEMP_COMBINED" ]; then
              CHUNK_SUCCESS=true
              echo -e "${GREEN}Chunk $((i + 1))/$TOTAL_CHUNKS fetched successfully ($(stat -f%z "$TEMP_CHUNK" 2>/dev/null || stat -c%s "$TEMP_CHUNK") bytes)${NC}"
            fi
          else
            echo -e "${YELLOW}Warning: Failed to parse chunk $i (attempt $((RETRY_COUNT + 1))/$MAX_RETRIES, jq exit: $JQ_EXIT)${NC}"
          fi
      else
        if [ $CURL_EXIT_CODE -ne 0 ]; then
          echo -e "${YELLOW}Warning: curl failed for chunk $i (exit: $CURL_EXIT_CODE, attempt $((RETRY_COUNT + 1))/$MAX_RETRIES)${NC}"
        else
          echo -e "${YELLOW}Warning: Failed to fetch chunk $i - empty response (attempt $((RETRY_COUNT + 1))/$MAX_RETRIES)${NC}"
        fi
      fi

      rm -f "$TEMP_CHUNK"

      if [ "$CHUNK_SUCCESS" = false ]; then
        RETRY_COUNT=$((RETRY_COUNT + 1))
        if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
          echo -e "${YELLOW}Retrying in 2 seconds...${NC}"
          sleep 2
        fi
      fi
    done

    # If still failed after all retries, abort
    if [ "$CHUNK_SUCCESS" = false ]; then
      echo -e "${RED}Error: Failed to fetch chunk $i after $MAX_RETRIES attempts${NC}"
      rm -f "$TEMP_COMBINED"
      return 1
    fi
  done

  # Verify we have data
  if [ ! -s "$TEMP_COMBINED" ]; then
    echo -e "${RED}Error: No genesis data received${NC}"
    rm -f "$TEMP_COMBINED"
    return 1
  fi

  # Decode base64 and save to file
  # Use -D flag for macOS base64 compatibility
  if [[ "$OSTYPE" == "darwin"* ]]; then
    cat "$TEMP_COMBINED" | base64 -D > "$OUTPUT_FILE"
  else
    base64 -d "$TEMP_COMBINED" > "$OUTPUT_FILE"
  fi

  local DECODE_STATUS=$?
  rm -f "$TEMP_COMBINED"

  if [ $DECODE_STATUS -ne 0 ]; then
    echo -e "${RED}Error: Failed to decode genesis data${NC}"
    return 1
  fi

  # Validate JSON
  if ! jq empty "$OUTPUT_FILE" 2>/dev/null; then
    echo -e "${RED}Error: Invalid JSON in genesis file${NC}"
    return 1
  fi

  echo -e "${GREEN}Successfully fetched and assembled genesis${NC}"
  return 0
}

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
echo -e "${YELLOW}Initializing new node: $NODE_ID with moniker: $MONIKER${NC}"
"$BINARY" init "$MONIKER" --chain-id "$CHAINID" --home "$NODE_DIR" --overwrite > /dev/null 2>&1

# Save validator metadata for later use when creating validator
echo -e "${YELLOW}Saving validator metadata...${NC}"
cat > "$NODE_DIR/validator_metadata.json" << EOF
{
  "moniker": "$MONIKER",
  "website": "$WEBSITE",
  "identity": "$IDENTITY",
  "security": "$SECURITY",
  "details": "$DETAILS"
}
EOF

# Get genesis from seed node
echo -e "${YELLOW}Fetching genesis from seed node...${NC}"
if ! fetch_genesis_chunked "$SEED_NODE_RPC" "$NODE_DIR/config/genesis.json"; then
  echo -e "${RED}Error: Failed to fetch genesis from $SEED_NODE_RPC${NC}"
  exit 1
fi

# Validate the genesis file
echo -e "${YELLOW}Validating genesis file...${NC}"
if [ ! -s "$NODE_DIR/config/genesis.json" ]; then
  echo -e "${RED}Error: Genesis file is empty${NC}"
  exit 1
fi

# Check if genesis has required fields
if ! jq -e '.chain_id' "$NODE_DIR/config/genesis.json" > /dev/null 2>&1; then
  echo -e "${RED}Error: Genesis file is missing required fields${NC}"
  exit 1
fi

GENESIS_CHAIN_ID=$(jq -r '.chain_id' "$NODE_DIR/config/genesis.json")
echo -e "${GREEN}Genesis validated - Chain ID: $GENESIS_CHAIN_ID${NC}"

# Verify chain ID matches
if [ "$GENESIS_CHAIN_ID" != "$CHAINID" ]; then
  echo -e "${YELLOW}Warning: Genesis chain ID ($GENESIS_CHAIN_ID) doesn't match expected ($CHAINID)${NC}"
  echo -e "${YELLOW}Using chain ID from genesis: $GENESIS_CHAIN_ID${NC}"
  CHAINID="$GENESIS_CHAIN_ID"
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
SEED_HOST=$(echo $SEED_NODE_RPC | sed 's|http://||' | sed 's|https://||' | sed 's|:.*||')
SEED_NODE_ID=$(curl -s "$SEED_NODE_RPC/status" | jq -r '.result.node_info.id')

if [ -z "$SEED_NODE_ID" ] || [ "$SEED_NODE_ID" = "null" ]; then
  echo -e "${RED}Error: Could not get seed node ID${NC}"
  exit 1
fi

SEED_ADDRESS="$SEED_NODE_ID@$SEED_HOST:$SEED_P2P_PORT"
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
