#!/bin/bash

set -e

CURRENT_DIR="$(pwd)"

# Parse command line arguments
NODE_ID=""
VALIDATOR_KEY=""
AMOUNT="1000000000000000000"
MONIKER=""
COMMISSION_RATE="0.10"
COMMISSION_MAX_RATE="0.20"
COMMISSION_MAX_CHANGE_RATE="0.01"
MIN_SELF_DELEGATION="1"
NETWORK="local"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Usage function
usage() {
  echo "Usage: $0 <node_id> [options]"
  echo "  node_id: Node ID for which to create validator (e.g., node4, node5)"
  echo ""
  echo "Options:"
  echo "  --validator-key <name>   Name of validator key in keyring (default: validator-<node_id>)"
  echo "  --amount <amount>        Validator stake amount (default: 1000000000000000000)"
  echo "  --moniker <name>         Validator moniker (default: <node_id>-validator)"
  echo "  --commission-rate <rate> Commission rate (default: 0.10)"
  echo "  --network <name>         Network to use (mainnet, testnet, devnet, local) (default: local)"
  echo "  --help                   Show this help message"
  echo ""
  echo "Environment variables:"
  echo "  SHARDEUM_NETWORK         Network to use (overrides --network)"
  echo ""
  echo "Examples:"
  echo "  $0 node5"
  echo "  $0 node5 --network testnet"
  echo "  $0 node5 --validator-key my-validator --amount 2000000000000000000"
  echo "  $0 node5 --moniker 'My Validator' --commission-rate 0.05 --network devnet"
  exit 1
}

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --validator-key)
      VALIDATOR_KEY="$2"
      shift 2
      ;;
    --amount)
      AMOUNT="$2"
      shift 2
      ;;
    --moniker)
      MONIKER="$2"
      shift 2
      ;;
    --commission-rate)
      COMMISSION_RATE="$2"
      shift 2
      ;;
    --network)
      NETWORK="$2"
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

# Validate required arguments
if [[ -z "$NODE_ID" ]]; then
  echo "Error: node_id is required"
  usage
fi

# Set defaults
VALIDATOR_KEY="${VALIDATOR_KEY:-validator-$NODE_ID}"
MONIKER="${MONIKER:-$NODE_ID-validator}"
NETWORK="${SHARDEUM_NETWORK:-${NETWORK:-local}}"

# Paths
BASE_DIR="${HOME:-$CURRENT_DIR}/.$NETWORK"
if [[ "$NETWORK" == "local" ]]; then
  BASE_DIR="$CURRENT_DIR/.$NETWORK"
fi

NODE_DIR="$BASE_DIR/$NODE_ID"
BINARY="${BINARY:-$(command -v shardeumd)}"

# Verify binary exists (skip build if called from makefile)
if [ ! -f "$BINARY" ]; then
  echo -e "${RED}Error: Failed to find binary at $BINARY${NC}"
  echo -e "${RED}Tip:${NC} Make sure shardeumd binary is set in PATH or set BINARY=/absolute/path/to/shardeumd"
  exit 1
fi

# Validate node exists
if [[ ! -d "$NODE_DIR" ]]; then
  echo -e "${RED}Error: Node $NODE_ID not found at $NODE_DIR${NC}"
  echo "Create the node first with: ./scripts/add_node.sh $NODE_ID"
  exit 1
fi

# Get network configuration
CHAINID=$(cat "$NODE_DIR/config/client.toml" | grep "chain-id" | cut -d'"' -f2)
if [[ -z "$CHAINID" ]]; then
  echo -e "${RED}Error: Could not determine chain ID from node configuration${NC}"
  exit 1
fi

# Get base denomination from config
BASE_DENOM="ashm"  # Default, could be made configurable

# Get node RPC port
RPC_PORT=$(cat "$NODE_DIR/config/client.toml" | grep "node" | cut -d':' -f3 | tr -d '"')
if [[ -z "$RPC_PORT" ]]; then
  echo -e "${RED}Error: Could not determine RPC port from node configuration${NC}"
  exit 1
fi

echo -e "${GREEN}Creating validator for node $NODE_ID${NC}"
echo -e "${YELLOW}Chain ID: $CHAINID${NC}"
echo -e "${YELLOW}Validator Key: $VALIDATOR_KEY${NC}"
echo -e "${YELLOW}Moniker: $MONIKER${NC}"
echo -e "${YELLOW}Amount: $AMOUNT$BASE_DENOM${NC}"
echo

# Check if validator key exists
if ! "$BINARY" keys show "$VALIDATOR_KEY" --keyring-backend test --home "$NODE_DIR" > /dev/null 2>&1; then
  echo -e "${RED}Error: Validator key '$VALIDATOR_KEY' not found in keyring${NC}"
  echo "Create a validator key first:"
  echo "  $BINARY keys add $VALIDATOR_KEY --keyring-backend test --home $NODE_DIR"
  echo "Then fund the account before creating the validator."
  exit 1
fi

# Get validator address
VALIDATOR_ADDR=$("$BINARY" keys show "$VALIDATOR_KEY" -a --keyring-backend test --home "$NODE_DIR")
echo -e "${YELLOW}Validator address: $VALIDATOR_ADDR${NC}"

# Check validator account balance
echo -e "${YELLOW}Checking validator account balance...${NC}"
BALANCE=$("$BINARY" query bank balances "$VALIDATOR_ADDR" --node "tcp://localhost:$RPC_PORT" -o json | jq -r ".balances[] | select(.denom==\"$BASE_DENOM\") | .amount")

if [[ -z "$BALANCE" || "$BALANCE" == "null" ]]; then
  echo -e "${RED}Error: Validator account has no $BASE_DENOM balance${NC}"
  echo "Fund the account first with at least $(($AMOUNT + 500000000000000000000)) $BASE_DENOM"
  exit 1
fi

# Use bc for large number arithmetic
REQUIRED_BALANCE=$(echo "$AMOUNT + 500000000000000000000" | bc)
if [[ $(echo "$BALANCE < $REQUIRED_BALANCE" | bc) -eq 1 ]]; then
  echo -e "${RED}Error: Insufficient balance. Required: $REQUIRED_BALANCE, Available: $BALANCE${NC}"
  exit 1
fi

echo -e "${GREEN}✅ Account has sufficient balance: $BALANCE $BASE_DENOM${NC}"

# Get validator public key
echo -e "${YELLOW}Getting validator consensus public key...${NC}"
VALIDATOR_PUBKEY=$("$BINARY" comet show-validator --home "$NODE_DIR")

# Create validator JSON file
VALIDATOR_JSON="$NODE_DIR/validator.json"
echo -e "${YELLOW}Creating validator configuration...${NC}"
cat > "$VALIDATOR_JSON" << EOF
{
  "pubkey": $VALIDATOR_PUBKEY,
  "amount": "$AMOUNT$BASE_DENOM",
  "moniker": "$MONIKER",
  "identity": "",
  "website": "",
  "security": "",
  "details": "Validator for $NODE_ID",
  "commission-rate": "$COMMISSION_RATE",
  "commission-max-rate": "$COMMISSION_MAX_RATE",
  "commission-max-change-rate": "$COMMISSION_MAX_CHANGE_RATE",
  "min-self-delegation": "$MIN_SELF_DELEGATION"
}
EOF

# Create validator transaction
echo -e "${YELLOW}Creating validator transaction...${NC}"
"$BINARY" tx staking create-validator "$VALIDATOR_JSON" \
  --from="$VALIDATOR_KEY" \
  --keyring-backend test \
  --home "$NODE_DIR" \
  --node "tcp://localhost:$RPC_PORT" \
  --chain-id="$CHAINID" \
  --gas 500000 \
  --fees 400000000000000000000${BASE_DENOM} \
  --yes

if [ $? -eq 0 ]; then
  echo
  echo -e "${GREEN}✅ Validator created successfully!${NC}"
  echo -e "${YELLOW}Validator address: $VALIDATOR_ADDR${NC}"
  echo -e "${YELLOW}Moniker: $MONIKER${NC}"
  echo -e "${YELLOW}Stake: $AMOUNT$BASE_DENOM${NC}"
  echo
  echo "Check validator status:"
  echo "  $BINARY query staking validator \$(echo '$VALIDATOR_ADDR' | sed 's/shardeum/shardeumvaloper/') --node tcp://localhost:$RPC_PORT"
  echo
  echo "The validator will be active in the next block."
else
  echo -e "${RED}❌ Validator creation failed${NC}"
  echo "Check the transaction details and try again."
  exit 1
fi