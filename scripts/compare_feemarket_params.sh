#!/bin/bash
# Compare feemarket module parameters between network and proposal file
#
# This script queries a network's current feemarket module parameters and compares
# them with a proposed feemarket parameter update.
#
# NETWORK CONFIGURATION:
# ======================
# By default, this script points to MAINNET. To point at other networks,
# either pass node-url and chain-id as arguments, or modify the defaults below:

DEFAULT_NODE="https://rpc.shardeum.org:443"      # Mainnet RPC endpoint
DEFAULT_CHAIN_ID="shardeum_8118-1"               # Mainnet chain ID

# For testnet, you would use:
# DEFAULT_NODE="https://rpc.testnet.shardeum.org:443"
# DEFAULT_CHAIN_ID="shardeum_8119-1"

# For local network:
# DEFAULT_NODE="http://localhost:26657"
# DEFAULT_CHAIN_ID="shardeum_8119-1"

# example:
# ./scripts/compare_feemarket_params.sh proposals/example-feemarket-proposal.json
  

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if proposal file path is provided
if [ -z "$1" ]; then
    echo -e "${RED}Error: Proposal file path is required${NC}"
    echo ""
    echo "Usage: $0 <proposal-file.json> [node-url] [chain-id]"
    echo ""
    echo -e "${YELLOW}Examples:${NC}"
    echo "  # Compare against mainnet (default):"
    echo "  $0 proposals/example-feemarket-proposal.json"
    echo ""
    echo "  # Compare against a different network:"
    echo "  $0 proposals/example-feemarket-proposal.json https://rpc.testnet.shardeum.org:443 shardeum_8119-1"
    echo ""
    echo "  # Use environment variables:"
    echo "  export SHARDEUM_NODE=https://rpc.testnet.shardeum.org:443"
    echo "  export SHARDEUM_CHAIN_ID=shardeum_8119-1"
    echo "  $0 proposals/example-feemarket-proposal.json"
    echo ""
    echo -e "${YELLOW}Note:${NC} This script currently defaults to MAINNET."
    echo "      Modify DEFAULT_NODE and DEFAULT_CHAIN_ID in the script to change defaults."
    echo ""
    exit 1
fi

# Parameters
PROPOSAL_FILE="$1"
NODE="${2:-${SHARDEUM_NODE:-$DEFAULT_NODE}}"
CHAIN_ID="${3:-${SHARDEUM_CHAIN_ID:-$DEFAULT_CHAIN_ID}}"
BINARY="${BINARY:-./build/shardeumd}"

echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║           Feemarket Module Parameters Comparison               ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}Network:${NC} $NODE"
echo -e "${YELLOW}Chain ID:${NC} $CHAIN_ID"
echo -e "${YELLOW}Proposal File:${NC} $PROPOSAL_FILE"
echo ""

# Query current params from network
echo -e "${BLUE}Querying current parameters from network...${NC}"
echo -e "${YELLOW}Query Command Used:${NC} $BINARY query feemarket params --node $NODE --chain-id $CHAIN_ID"
echo ""
CURRENT_PARAMS=$($BINARY query feemarket params --node "$NODE" --chain-id "$CHAIN_ID" --output json 2>/dev/null)

if [ $? -ne 0 ]; then
    echo -e "${RED}Error: Failed to query network parameters${NC}"
    exit 1
fi

# Extract proposed params from proposal file
echo -e "${BLUE}Reading proposed parameters from file...${NC}"
if [ ! -f "$PROPOSAL_FILE" ]; then
    echo -e "${RED}Error: Proposal file not found: $PROPOSAL_FILE${NC}"
    exit 1
fi

PROPOSED_PARAMS=$(cat "$PROPOSAL_FILE" | jq -r '.messages[0].params')

echo ""
echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                    PARAMETER COMPARISON                        ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Compare each parameter
compare_param() {
    local param_name=$1
    local param_path=$2
    
    current=$(echo "$CURRENT_PARAMS" | jq -r ".params$param_path // \"N/A\"")
    proposed=$(echo "$PROPOSED_PARAMS" | jq -r "$param_path // \"N/A\"")
    
    if [ "$current" == "$proposed" ]; then
        echo -e "${GREEN}✓${NC} ${YELLOW}${param_name}:${NC}"
        echo -e "    Current:  $current"
        echo -e "    Proposed: $proposed"
        echo -e "    ${GREEN}(No change)${NC}"
    else
        echo -e "${RED}✗${NC} ${YELLOW}${param_name}:${NC}"
        echo -e "    Current:  ${RED}$current${NC}"
        echo -e "    Proposed: ${GREEN}$proposed${NC}"
        echo -e "    ${YELLOW}(Will be changed)${NC}"
    fi
    echo ""
}

# Compare all feemarket module parameters
compare_param "No Base Fee" '.no_base_fee'
compare_param "Base Fee Change Denominator" '.base_fee_change_denominator'
compare_param "Elasticity Multiplier" '.elasticity_multiplier'
compare_param "Enable Height" '.enable_height'
compare_param "Base Fee" '.base_fee'
compare_param "Min Gas Price" '.min_gas_price'
compare_param "Min Gas Multiplier" '.min_gas_multiplier'

echo ""
echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"

# Count differences
CURRENT_JSON=$(echo "$CURRENT_PARAMS" | jq -S '.params')
PROPOSED_JSON=$(echo "$PROPOSED_PARAMS" | jq -S '.')

if [ "$CURRENT_JSON" == "$PROPOSED_JSON" ]; then
    echo -e "${GREEN}✓ All parameters match - no changes will be made${NC}"
else
    echo -e "${YELLOW}⚠ Some parameters differ - proposal will make changes${NC}"
fi

echo ""
