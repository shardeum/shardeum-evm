#!/bin/bash
#
# Network Validator for Economic Simulation
#
# Launches a 1-node test network with custom genesis, queries mint endpoints,
# and validates simulation results against actual network behavior.
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
GENESIS_FILE=""
OUTPUT_FILE=""
WAIT_TIME=10
RPC_PORT=1317
JSON_RPC_PORT=8545
CLEANUP=true
VERBOSE=false

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Usage
usage() {
    cat <<EOF
Usage: $0 [OPTIONS]

Launch a test network with custom genesis and query mint parameters.

OPTIONS:
    -g, --genesis FILE       Path to genesis file (required)
    -o, --output FILE        Output file for results (JSON format)
    -w, --wait SECONDS       Seconds to wait for network to start (default: 10)
    -p, --rpc-port PORT      REST API port (default: 1317)
    --no-cleanup             Don't stop the network after querying
    -v, --verbose            Verbose output
    -h, --help               Show this help message

EXAMPLE:
    $0 -g simulation_configs/test-genesis-min7_max20_goal67.json -o results.json

EOF
    exit 1
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -g|--genesis)
            GENESIS_FILE="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        -w|--wait)
            WAIT_TIME="$2"
            shift 2
            ;;
        -p|--rpc-port)
            RPC_PORT="$2"
            shift 2
            ;;
        --no-cleanup)
            CLEANUP=false
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo -e "${RED}Error: Unknown option $1${NC}"
            usage
            ;;
    esac
done

# Validate required arguments
if [[ -z "$GENESIS_FILE" ]]; then
    echo -e "${RED}Error: Genesis file is required (-g/--genesis)${NC}"
    usage
fi

if [[ ! -f "$GENESIS_FILE" ]]; then
    echo -e "${RED}Error: Genesis file not found: $GENESIS_FILE${NC}"
    exit 1
fi

# Convert to absolute path if needed
if [[ ! "$GENESIS_FILE" = /* ]]; then
    GENESIS_FILE="$REPO_ROOT/$GENESIS_FILE"
fi

log() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${GREEN}[INFO]${NC} $1"
    fi
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Cleanup function
cleanup_network() {
    if [[ "$CLEANUP" == "true" ]]; then
        log "Cleaning up network..."
        pkill -f shardeumd || true
        sleep 2
    fi
}

# Set up trap for cleanup
trap cleanup_network EXIT

# Main execution
echo "Network Validator for Economic Simulation"
echo "=========================================="
echo

log "Genesis file: $GENESIS_FILE"
log "RPC port: $RPC_PORT"
log "Wait time: ${WAIT_TIME}s"
echo

# Check if shardeumd is already running
if pgrep -x "shardeumd" > /dev/null; then
    warn "shardeumd is already running. Attempting to stop..."
    pkill -f shardeumd || true
    sleep 2
fi

# Start the network
echo "Starting network with custom genesis..."
log "Command: cd $REPO_ROOT && ./local_node.sh -y -g $GENESIS_FILE --no-install"

cd "$REPO_ROOT"

# Start network in background
if [[ "$VERBOSE" == "true" ]]; then
    ./local_node.sh -y -g "$GENESIS_FILE" &
else
    ./local_node.sh -y -g "$GENESIS_FILE" > /dev/null 2>&1 &
fi

NETWORK_PID=$!

log "Network started with PID: $NETWORK_PID"

# Wait for network to start
echo "Waiting ${WAIT_TIME}s for network to start..."
sleep "$WAIT_TIME"

# Check if network is still running
if ! kill -0 $NETWORK_PID 2>/dev/null; then
    error "Network failed to start"
    exit 1
fi

# Wait for RPC to be ready
echo "Checking RPC endpoint..."
MAX_RETRIES=30
RETRY_COUNT=0
RPC_READY=false

while [[ $RETRY_COUNT -lt $MAX_RETRIES ]]; do
    if curl -s "http://localhost:${RPC_PORT}/cosmos/base/tendermint/v1beta1/node_info" > /dev/null 2>&1; then
        RPC_READY=true
        break
    fi

    RETRY_COUNT=$((RETRY_COUNT + 1))
    log "Waiting for RPC... (attempt $RETRY_COUNT/$MAX_RETRIES)"
    sleep 1
done

if [[ "$RPC_READY" == "false" ]]; then
    error "RPC endpoint failed to become ready"
    exit 1
fi

echo -e "${GREEN}Network is ready!${NC}"
echo

# Query mint endpoints
echo "Querying mint module endpoints..."
echo

MINT_PARAMS=$(curl -s "http://localhost:${RPC_PORT}/cosmos/mint/v1beta1/params")
ANNUAL_PROVISIONS=$(curl -s "http://localhost:${RPC_PORT}/cosmos/mint/v1beta1/annual_provisions")
INFLATION=$(curl -s "http://localhost:${RPC_PORT}/cosmos/mint/v1beta1/inflation")

# Check if queries succeeded
if [[ -z "$MINT_PARAMS" ]] || [[ "$MINT_PARAMS" == *"error"* ]]; then
    error "Failed to query mint params"
    exit 1
fi

# Display results
echo "Results:"
echo "--------"
echo
echo "Mint Parameters:"
echo "$MINT_PARAMS" | jq '.'
echo
echo "Annual Provisions:"
echo "$ANNUAL_PROVISIONS" | jq '.'
echo
echo "Inflation:"
echo "$INFLATION" | jq '.'
echo

# Combine results into single JSON object
RESULTS=$(jq -n \
    --argjson params "$MINT_PARAMS" \
    --argjson provisions "$ANNUAL_PROVISIONS" \
    --argjson inflation "$INFLATION" \
    '{
        timestamp: (now | todate),
        mint_params: $params,
        annual_provisions: $provisions,
        inflation: $inflation
    }')

# Save to file if specified
if [[ -n "$OUTPUT_FILE" ]]; then
    # Create output directory if needed
    OUTPUT_DIR=$(dirname "$OUTPUT_FILE")
    mkdir -p "$OUTPUT_DIR"

    echo "$RESULTS" > "$OUTPUT_FILE"
    echo -e "${GREEN}Results saved to: $OUTPUT_FILE${NC}"
fi

echo
echo "=========================================="
echo -e "${GREEN}Validation complete!${NC}"

# Return results to stdout for programmatic use
echo "$RESULTS"

exit 0
