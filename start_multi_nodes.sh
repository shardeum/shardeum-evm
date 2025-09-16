#!/bin/bash

# Script to easily start multiple Shardeum nodes locally
# This creates separate processes for each node with different ports

set -e

# Default values
DEFAULT_NODES=4
CHAINID="${CHAIN_ID:-local-testnet}"
OUTPUT_DIR="./.multi-nodes"
KEYRING="test"
MIN_GAS_PRICES="0.000006ashm"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

usage() {
  cat <<EOF
Usage: $0 [options]

Options:
  -n, --nodes N        Number of nodes to start (default: 4)
  -c, --chain-id ID    Chain ID (default: local-testnet)
  -o, --output-dir DIR Output directory for node configs (default: ./.multi-nodes)
  --clean              Clean existing data before starting
  --no-build           Skip building the binary
  -h, --help           Show this help message

Examples:
  $0                   # Start 4 nodes with default settings
  $0 -n 3              # Start 3 nodes
  $0 -n 5 --clean      # Clean data and start 5 nodes
EOF
}

# Parse command line arguments
NODES=""
CLEAN=false
NO_BUILD=false

while [[ $# -gt 0 ]]; do
  case $1 in
    -n|--nodes)
      NODES="$2"
      shift 2
      ;;
    -c|--chain-id)
      CHAINID="$2"
      shift 2
      ;;
    -o|--output-dir)
      OUTPUT_DIR="$2"
      shift 2
      ;;
    --clean)
      CLEAN=true
      shift
      ;;
    --no-build)
      NO_BUILD=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo -e "${RED}Unknown option: $1${NC}"
      usage
      exit 1
      ;;
  esac
done

# Interactive prompt for number of nodes if not specified
if [[ -z "$NODES" ]]; then
  echo -e "${YELLOW}How many nodes do you want to start? (default: $DEFAULT_NODES)${NC}"
  read -r input
  NODES=${input:-$DEFAULT_NODES}
fi

# Validate nodes is a number and reasonable
if ! [[ "$NODES" =~ ^[0-9]+$ ]] || [[ "$NODES" -lt 1 ]] || [[ "$NODES" -gt 20 ]]; then
  echo -e "${RED}Error: Number of nodes must be between 1 and 20${NC}"
  exit 1
fi

echo -e "${GREEN}Starting $NODES nodes for chain: $CHAINID${NC}"

# Build the binary if requested
if [[ "$NO_BUILD" != "true" ]]; then
  echo -e "${YELLOW}Building evmd binary...${NC}"
  make install
fi

# Clean existing data if requested
if [[ "$CLEAN" == "true" ]] && [[ -d "$OUTPUT_DIR" ]]; then
  echo -e "${YELLOW}Cleaning existing data in $OUTPUT_DIR...${NC}"
  rm -rf "$OUTPUT_DIR"
fi

# Create testnet configuration files
echo -e "${YELLOW}Initializing $NODES node configurations...${NC}"
evmd testnet init-files \
  --validator-count "$NODES" \
  --output-dir "$OUTPUT_DIR" \
  --chain-id "$CHAINID" \
  --single-host \
  --keyring-backend "$KEYRING" \
  --minimum-gas-prices "$MIN_GAS_PRICES"

if [[ $? -ne 0 ]]; then
  echo -e "${RED}Failed to initialize testnet files${NC}"
  exit 1
fi

# Function to start a single node
start_node() {
  local node_num=$1
  local node_dir="$OUTPUT_DIR/node$node_num/evmd"
  
  echo -e "${GREEN}Starting node$node_num...${NC}"
  
  evmd start \
    --home "$node_dir" \
    --pruning nothing \
    --log_level info \
    --minimum-gas-prices="$MIN_GAS_PRICES" \
    --json-rpc.api eth,txpool,personal,net,debug,web3 \
    --chain-id "$CHAINID" \
    > "$OUTPUT_DIR/node$node_num.log" 2>&1 &
  
  local pid=$!
  echo "$pid" > "$OUTPUT_DIR/node$node_num.pid"
  echo -e "${GREEN}Node $node_num started with PID $pid${NC}"
}

# Start all nodes
echo -e "${YELLOW}Starting all nodes...${NC}"
for ((i=0; i<NODES; i++)); do
  start_node $i
  sleep 2  # Small delay between starting nodes
done

# Create stop script
cat > "$OUTPUT_DIR/stop_nodes.sh" << 'EOF'
#!/bin/bash
echo "Stopping all nodes..."
for pidfile in *.pid; do
  if [[ -f "$pidfile" ]]; then
    pid=$(cat "$pidfile")
    if kill -0 "$pid" 2>/dev/null; then
      echo "Stopping node with PID $pid"
      kill "$pid"
    fi
    rm -f "$pidfile"
  fi
done
echo "All nodes stopped"
EOF
chmod +x "$OUTPUT_DIR/stop_nodes.sh"

# Display connection info
echo
echo -e "${GREEN}===== MULTI-NODE TESTNET STARTED =====${NC}"
echo -e "${YELLOW}Chain ID:${NC} $CHAINID"
echo -e "${YELLOW}Number of nodes:${NC} $NODES"
echo -e "${YELLOW}Output directory:${NC} $OUTPUT_DIR"
echo
echo -e "${YELLOW}Node endpoints:${NC}"
for ((i=0; i<NODES; i++)); do
  rpc_port=$((26657 + i))
  api_port=$((1317 + i))
  grpc_port=$((9090 + i))
  jsonrpc_port=$((8545 + i * 10))
  
  echo -e "  ${GREEN}Node $i:${NC}"
  echo -e "    RPC:      http://localhost:$rpc_port"
  echo -e "    API:      http://localhost:$api_port"
  echo -e "    gRPC:     localhost:$grpc_port"
  echo -e "    JSON-RPC: http://localhost:$jsonrpc_port"
done

echo
echo -e "${YELLOW}Logs:${NC} $OUTPUT_DIR/node*.log"
echo -e "${YELLOW}To stop all nodes:${NC} $OUTPUT_DIR/stop_nodes.sh"
echo
echo -e "${GREEN}All nodes are starting up... Check the logs for any issues.${NC}"
echo -e "${YELLOW}The first block should be produced shortly.${NC}"