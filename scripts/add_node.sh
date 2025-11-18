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
API_ENABLE="false"
ONLY_FETCH=false
MULTISIG_KEYS_FILE=""
MULTISIG_THRESHOLD=""

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
  echo "  --api-enable         Enable Cosmos API server (default: false)"
  echo "  --only-fetch         Only download genesis to ./genesis.json, print chain-id and raw SHA256, then exit"
  echo "  --multisig-keys-file <path>  Path to JSON file containing signer public keys (creates multisig operator key)"
  echo "  --multisig-threshold <n>     Multisig threshold (required if --multisig-keys-file is provided)"
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
  echo "  $0 node8 --multisig-keys-file signers.json --multisig-threshold 2"
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
    --only-fetch)
      ONLY_FETCH=true
      shift 1
      ;;
    --api-enable)
      API_ENABLE="true"
      shift
      ;;
    --multisig-keys-file)
      MULTISIG_KEYS_FILE="$2"
      shift 2
      ;;
    --multisig-threshold)
      MULTISIG_THRESHOLD="$2"
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
      echo "  --api-enable         Enable Cosmos API server (default: false)"
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
if [[ -z "$SHARDEUM_CONFIG_DIR" && "$ONLY_FETCH" != "true" ]]; then
  echo "Error: SHARDEUM_CONFIG_DIR is required (omit for --only-fetch)"
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

if [[ "$ONLY_FETCH" != "true" ]]; then
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

  # We will decode each chunk individually to avoid padding conflicts.
  : > "$OUTPUT_FILE"

  # Decode and append chunk 0 first
  if [[ "$OSTYPE" == "darwin"* ]]; then
    printf '%s' "$CHUNK_0_DATA" | base64 -D >> "$OUTPUT_FILE" || { echo -e "${RED}Decode failed (chunk 0)${NC}"; rm -f "$TEMP_COMBINED"; return 1; }
  else
    printf '%s' "$CHUNK_0_DATA" | base64 -d >> "$OUTPUT_FILE" || { echo -e "${RED}Decode failed (chunk 0)${NC}"; rm -f "$TEMP_COMBINED"; return 1; }
  fi
  echo -e "${GREEN}Chunk 1/$TOTAL_CHUNKS decoded${NC}"
  rm -f "$TEMP_CHUNK_0"

  # Fetch and decode remaining chunks (if any)
  if [ "$TOTAL_CHUNKS" -gt 1 ]; then
    for i in $(seq 1 $((TOTAL_CHUNKS - 1))); do
      echo -e "${YELLOW}Fetching chunk $((i + 1))/$TOTAL_CHUNKS...${NC}"
      local TEMP_CHUNK="/tmp/chunk_${i}_$$.json"
      local MAX_RETRIES=3
      local RETRIES=0
      local OK=false
      while [ $RETRIES -lt $MAX_RETRIES ] && [ "$OK" = false ]; do
        curl -s --max-time 600 --connect-timeout 30 "$RPC_URL/genesis_chunked?chunk=$i" -o "$TEMP_CHUNK"
        local CE=$?
        if [ $CE -eq 0 ] && [ -s "$TEMP_CHUNK" ]; then
          local DATA=$(jq -r '.result.data // empty' "$TEMP_CHUNK" 2>/dev/null)
          if [ -n "$DATA" ] && [ "$DATA" != "null" ]; then
            if [[ "$OSTYPE" == "darwin"* ]]; then
              if printf '%s' "$DATA" | base64 -D >> "$OUTPUT_FILE" 2>/dev/null; then OK=true; fi
            else
              if printf '%s' "$DATA" | base64 -d >> "$OUTPUT_FILE" 2>/dev/null; then OK=true; fi
            fi
          fi
        fi
        if [ "$OK" = false ]; then
          RETRIES=$((RETRIES + 1))
          if [ $RETRIES -lt $MAX_RETRIES ]; then
            echo -e "${YELLOW}Retry chunk $i in 2s (attempt $((RETRIES+1))/$MAX_RETRIES)${NC}"
            sleep 2
          fi
        fi
        rm -f "$TEMP_CHUNK"
      done
      if [ "$OK" = false ]; then
        echo -e "${RED}Error: Failed to fetch/decode chunk $i${NC}"
        rm -f "$TEMP_COMBINED"
        return 1
      fi
      echo -e "${GREEN}Chunk $((i + 1))/$TOTAL_CHUNKS decoded${NC}"
    done
  fi

  # Basic sanity
  if [ ! -s "$OUTPUT_FILE" ]; then
    echo -e "${RED}Error: Decoded genesis file empty${NC}"; rm -f "$TEMP_COMBINED"; return 1; fi
  rm -f "$TEMP_COMBINED"

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

if [[ "$ONLY_FETCH" != "true" ]]; then
  # Verify binary exists (skip build if called from makefile)
  if [ ! -f "$BINARY" ]; then
    echo -e "${RED}Error: Failed to find binary at $BINARY${NC}"
    echo -e "${RED}Tip:${NC} Make sure shardeumd binary is set in PATH or set BINARY=/absolute/path/to/shardeumd"
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

if [[ "$ONLY_FETCH" == "true" ]]; then
  echo -e "${YELLOW}--only-fetch mode: downloading genesis to ./genesis.json${NC}"
  if ! fetch_genesis_chunked "$SEED_NODE_RPC" "./genesis.json"; then
    echo -e "${RED}Error: Failed to fetch genesis${NC}"
    exit 1
  fi
  if ! jq -e '.chain_id' ./genesis.json >/dev/null 2>&1; then
    echo -e "${RED}Error: Downloaded genesis missing chain_id${NC}"
    exit 1
  fi
  GENESIS_CHAIN_ID=$(jq -r '.chain_id' ./genesis.json)
  RAW_HASH=$(sha256sum ./genesis.json | awk '{print $1}')
  echo -e "${GREEN}Genesis fetched successfully${NC}"
  echo -e "${YELLOW}Chain ID: $GENESIS_CHAIN_ID${NC}"
  echo -e "${YELLOW}Raw SHA256: $RAW_HASH  genesis.json${NC}"
  echo -e "${GREEN}Done (--only-fetch)${NC}"
  exit 0
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

# Handle multisig operator key creation if requested
if [[ -n "$MULTISIG_KEYS_FILE" || -n "$MULTISIG_THRESHOLD" ]]; then
  # Validate both parameters are provided
  if [[ -z "$MULTISIG_KEYS_FILE" ]]; then
    echo -e "${RED}Error: --multisig-keys-file is required when --multisig-threshold is provided${NC}"
    exit 1
  fi
  if [[ -z "$MULTISIG_THRESHOLD" ]]; then
    echo -e "${RED}Error: --multisig-threshold is required when --multisig-keys-file is provided${NC}"
    exit 1
  fi

  # Validate keys file exists
  if [[ ! -f "$MULTISIG_KEYS_FILE" ]]; then
    echo -e "${RED}Error: Multisig keys file not found: $MULTISIG_KEYS_FILE${NC}"
    exit 1
  fi

  # Validate JSON format
  if ! jq empty "$MULTISIG_KEYS_FILE" 2>/dev/null; then
    echo -e "${RED}Error: Invalid JSON in multisig keys file: $MULTISIG_KEYS_FILE${NC}"
    exit 1
  fi

  echo -e "${YELLOW}Setting up multisig operator key...${NC}"

  # Extract signers array
  SIGNERS_COUNT=$(jq '.signers | length' "$MULTISIG_KEYS_FILE" 2>/dev/null || echo "0")
  if [[ "$SIGNERS_COUNT" -eq 0 ]]; then
    echo -e "${RED}Error: No signers found in multisig keys file. Expected 'signers' array.${NC}"
    exit 1
  fi

  # Validate threshold <= signers count
  if [[ "$MULTISIG_THRESHOLD" -gt "$SIGNERS_COUNT" ]]; then
    echo -e "${RED}Error: Threshold ($MULTISIG_THRESHOLD) cannot be greater than number of signers ($SIGNERS_COUNT)${NC}"
    exit 1
  fi

  if [[ "$MULTISIG_THRESHOLD" -lt 1 ]]; then
    echo -e "${RED}Error: Threshold must be at least 1${NC}"
    exit 1
  fi

  # Import each signer's public key as offline key
  SIGNER_NAMES=()
  SIGNER_ADDRESSES=()
  SIGNER_PUBKEYS=()

  for i in $(seq 0 $((SIGNERS_COUNT - 1))); do
    SIGNER_NAME=$(jq -r ".signers[$i].name" "$MULTISIG_KEYS_FILE" 2>/dev/null)
    SIGNER_PUBKEY=$(jq -c ".signers[$i].pubkey" "$MULTISIG_KEYS_FILE" 2>/dev/null)

    # Validate name is present
    if [[ -z "$SIGNER_NAME" || "$SIGNER_NAME" == "null" ]]; then
      echo -e "${RED}Error: Signer at index $i is missing 'name' field${NC}"
      exit 1
    fi

    # Validate pubkey is present
    if [[ -z "$SIGNER_PUBKEY" || "$SIGNER_PUBKEY" == "null" ]]; then
      echo -e "${RED}Error: Signer '$SIGNER_NAME' is missing 'pubkey' field${NC}"
      exit 1
    fi

    # Validate pubkey format - should be a JSON object
    echo -e "${YELLOW}Validating public key for $SIGNER_NAME...${NC}"
    if ! echo "$SIGNER_PUBKEY" | jq -e '."@type"' > /dev/null 2>&1; then
      echo -e "${RED}Error: Invalid public key format for signer '$SIGNER_NAME'${NC}"
      echo "Expected format: JSON object with @type and key fields"
      echo "Example: {\"@type\":\"/ethermint.crypto.v1.ethsecp256k1.PubKey\",\"key\":\"...\"}"
      exit 1
    fi
    
    # Try to import as a test to validate the key is actually valid
    if ! "$BINARY" keys add "${SIGNER_NAME}_test" --pubkey="$SIGNER_PUBKEY" --keyring-backend test --home "$NODE_DIR" --dry-run > /dev/null 2>&1; then
      echo -e "${RED}Error: Invalid public key for signer '$SIGNER_NAME' - key validation failed${NC}"
      exit 1
    fi
    # Clean up test key if it was created
    "$BINARY" keys delete "${SIGNER_NAME}_test" --keyring-backend test --home "$NODE_DIR" -y > /dev/null 2>&1 || true

    # Import as offline key
    echo -e "${YELLOW}Importing signer '$SIGNER_NAME' as offline key...${NC}"
    if ! "$BINARY" keys add "$SIGNER_NAME" --pubkey="$SIGNER_PUBKEY" --keyring-backend test --home "$NODE_DIR" > /dev/null 2>&1; then
      echo -e "${RED}Error: Failed to import public key for signer '$SIGNER_NAME'${NC}"
      exit 1
    fi

    # Get address for this signer
    SIGNER_ADDR=$("$BINARY" keys show "$SIGNER_NAME" -a --keyring-backend test --home "$NODE_DIR")
    SIGNER_NAMES+=("$SIGNER_NAME")
    SIGNER_ADDRESSES+=("$SIGNER_ADDR")
    SIGNER_PUBKEYS+=("$SIGNER_PUBKEY")

    echo -e "${GREEN}✓ Imported $SIGNER_NAME (address: $SIGNER_ADDR)${NC}"
  done

  # Create multisig key
  MULTISIG_KEY_NAME="validator-operator"
  SIGNER_NAMES_CSV=$(IFS=,; echo "${SIGNER_NAMES[*]}")

  echo -e "${YELLOW}Creating multisig operator key '$MULTISIG_KEY_NAME' (threshold: $MULTISIG_THRESHOLD of $SIGNERS_COUNT)...${NC}"
  if ! "$BINARY" keys add "$MULTISIG_KEY_NAME" \
    --multisig "$SIGNER_NAMES_CSV" \
    --multisig-threshold "$MULTISIG_THRESHOLD" \
    --keyring-backend test \
    --home "$NODE_DIR" > /dev/null 2>&1; then
    echo -e "${RED}Error: Failed to create multisig operator key${NC}"
    exit 1
  fi

  # Get multisig addresses
  MULTISIG_ADDR=$("$BINARY" keys show "$MULTISIG_KEY_NAME" -a --keyring-backend test --home "$NODE_DIR")
  VALIDATOR_OPERATOR_ADDR=$("$BINARY" keys show "$MULTISIG_KEY_NAME" --bech val -a --keyring-backend test --home "$NODE_DIR")

  # Create multisig_info.json
  echo -e "${YELLOW}Creating multisig info file...${NC}"
  cat > "$NODE_DIR/multisig_info.json" << EOF
{
  "multisig_key_name": "$MULTISIG_KEY_NAME",
  "multisig_address": "$MULTISIG_ADDR",
  "validator_operator_address": "$VALIDATOR_OPERATOR_ADDR",
  "threshold": $MULTISIG_THRESHOLD,
  "total_signers": $SIGNERS_COUNT,
  "signers": [
$(for i in $(seq 0 $((SIGNERS_COUNT - 1))); do
    if [[ $i -lt $((SIGNERS_COUNT - 1)) ]]; then
      echo "    {\"name\": \"${SIGNER_NAMES[$i]}\", \"address\": \"${SIGNER_ADDRESSES[$i]}\"},"
    else
      echo "    {\"name\": \"${SIGNER_NAMES[$i]}\", \"address\": \"${SIGNER_ADDRESSES[$i]}\"}"
    fi
  done)
  ]
}
EOF

  echo -e "${GREEN}✅ Multisig operator key created successfully!${NC}"
  echo -e "${YELLOW}Multisig Address: $MULTISIG_ADDR${NC}"
  echo -e "${YELLOW}Validator Operator Address: $VALIDATOR_OPERATOR_ADDR${NC}"
  echo -e "${YELLOW}Threshold: $MULTISIG_THRESHOLD of $SIGNERS_COUNT${NC}"
  echo -e "${YELLOW}Multisig info saved to: $NODE_DIR/multisig_info.json${NC}"
  echo ""
  echo -e "${YELLOW}📝 Next steps:${NC}"
  echo "1. Fund the multisig address: $MULTISIG_ADDR"
  echo "2. Create unsigned validator transaction"
  echo "3. Share with signers for signatures"
  echo "4. Merge signatures and broadcast"
fi

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
fi

# Enable API if flag is set
if [ "$API_ENABLE" = "true" ]; then
  START_CMD+=(--api.enable)
  START_CMD+=(--api.address "tcp://0.0.0.0:$API_PORT")
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
  if [[ -n "$MULTISIG_KEYS_FILE" && -n "$MULTISIG_THRESHOLD" ]]; then
    echo -e "${YELLOW}📝 Multisig operator key created!${NC}"
    echo "1. Fund the multisig address (see $NODE_DIR/multisig_info.json)"
    echo "2. Create unsigned validator transaction"
    echo "3. Share with signers for signatures (threshold: $MULTISIG_THRESHOLD)"
    echo "4. Merge signatures and broadcast"
  else
    echo -e "${YELLOW}📝 To create a validator:${NC}"
    echo "1. Create and fund a validator account"
    echo "2. Run: ./scripts/create_validator.sh $NODE_ID"
  fi
  echo
fi

echo "Stop this node: kill $NODE_PID or Ctrl+C"
echo "Logs: $NODE_DIR/node.log"
echo
echo -e "${YELLOW}Node is starting up and discovering peers. Press Ctrl+C to stop.${NC}"

# Wait for interrupt signal
wait
