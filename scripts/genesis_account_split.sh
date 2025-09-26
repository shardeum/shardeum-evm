#!/bin/bash
# Unified Genesis Account Management Script
# Handles splitting, merging, and loading of genesis accounts

set -e

# Default configuration
ACCOUNTS_PER_FILE=5000
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# -----------------------------
# Helper Functions
# -----------------------------

print_header() {
  echo "========================================="
  echo "$1"
  echo "========================================="
}

print_error() {
  echo -e "${RED}Error: $1${NC}" >&2
}

print_success() {
  echo -e "${GREEN}$1${NC}"
}

print_info() {
  echo -e "${YELLOW}$1${NC}"
}

usage() {
  echo "Usage: $0 <command> [options]"
  echo ""
  echo "Commands:"
  echo "  split    Split genesis accounts into multiple files"
  echo "  merge    Merge split account files into genesis"
  echo "  load     Load genesis with automatic account merging"
  echo "  check    Check if genesis has split account files"
  echo "  count    Count total accounts (including split files)"
  echo ""
  echo "Split Options:"
  echo "  -n, --network <name>         Network to split (mainnet, testnet, devnet, local)"
  echo "  -f, --file <path>            Genesis file to split (alternative to --network)"
  echo "  -a, --accounts <num>         Accounts per file (default: 5000)"
  echo "  -o, --output-dir <path>      Output directory (default: same as genesis file)"
  echo ""
  echo "Merge Options:"
  echo "  -f, --file <path>            Main genesis file to merge accounts into"
  echo "  -d, --accounts-dir <path>    Directory containing account files"
  echo "  -o, --output <path>          Output path for merged genesis"
  echo ""
  echo "Load Options:"
  echo "  -f, --file <path>            Genesis file to load"
  echo "  -o, --output <path>          Output path for complete genesis (optional)"
  echo ""
  echo "Check/Count Options:"
  echo "  -n, --network <name>         Network to check"
  echo "  -f, --file <path>            Genesis file to check"
  echo ""
  echo "Global Options:"
  echo "  -h, --help                   Show this help message"
  echo ""
  echo "Examples:"
  echo "  # Split mainnet genesis into 10000-account chunks"
  echo "  $0 split --network mainnet --accounts 10000"
  echo ""
  echo "  # Split custom genesis file"
  echo "  $0 split --file ./custom-genesis.json --accounts 1000"
  echo ""
  echo "  # Merge split accounts back into genesis"
  echo "  $0 merge --file ./mainnet.genesis.json --output ./complete-genesis.json"
  echo ""
  echo "  # Load genesis (auto-merges if needed)"
  echo "  $0 load --file ./mainnet.genesis.json"
  echo ""
  echo "  # Check if testnet has split files"
  echo "  $0 check --network testnet"
  echo ""
  echo "  # Count total accounts in mainnet"
  echo "  $0 count --network mainnet"
  exit 1
}

# -----------------------------
# Find genesis file for network
# -----------------------------
find_network_genesis() {
  local NETWORK="$1"
  
  # Check for split pattern first
  local GENESIS_FILE="$REPO_ROOT/config/environments/${NETWORK}.genesis.json"
  if [ ! -f "$GENESIS_FILE" ]; then
    # Try with hyphen
    GENESIS_FILE="$REPO_ROOT/config/environments/${NETWORK}-genesis.json"
  fi
  
  if [ ! -f "$GENESIS_FILE" ]; then
    print_error "Genesis file not found for network: $NETWORK"
    echo "Looked for:"
    echo "  - $REPO_ROOT/config/environments/${NETWORK}.genesis.json"
    echo "  - $REPO_ROOT/config/environments/${NETWORK}-genesis.json"
    return 1
  fi
  
  echo "$GENESIS_FILE"
  return 0
}

# -----------------------------
# Python implementation embedded
# -----------------------------
run_python_cmd() {
  local CMD="$1"
  python3 -c "$CMD"
}

# -----------------------------
# Split Command
# -----------------------------
cmd_split() {
  local NETWORK=""
  local GENESIS_FILE=""
  local OUTPUT_DIR=""
  
  while [[ $# -gt 0 ]]; do
    case $1 in
      -n|--network)
        NETWORK="$2"
        shift 2
        ;;
      -f|--file)
        GENESIS_FILE="$2"
        shift 2
        ;;
      -a|--accounts)
        ACCOUNTS_PER_FILE="$2"
        shift 2
        ;;
      -o|--output-dir)
        OUTPUT_DIR="$2"
        shift 2
        ;;
      *)
        print_error "Unknown option for split: $1"
        usage
        ;;
    esac
  done
  
  # Determine genesis file
  if [ -n "$GENESIS_FILE" ]; then
    if [ ! -f "$GENESIS_FILE" ]; then
      print_error "Genesis file not found: $GENESIS_FILE"
      exit 1
    fi
  elif [ -n "$NETWORK" ]; then
    GENESIS_FILE=$(find_network_genesis "$NETWORK") || exit 1
  else
    print_error "Must specify either --network or --file"
    usage
  fi
  
  # Determine output directory
  if [ -z "$OUTPUT_DIR" ]; then
    OUTPUT_DIR="$(dirname "$GENESIS_FILE")"
  fi
  
  # Create output directory if needed
  mkdir -p "$OUTPUT_DIR"
  
  print_header "Genesis Account Splitter"
  echo "Genesis file:      $GENESIS_FILE"
  echo "Output directory:  $OUTPUT_DIR"
  echo "Accounts per file: $ACCOUNTS_PER_FILE"
  echo ""
  
  # Count existing accounts
  ACCOUNT_COUNT=$(jq '.app_state.auth.accounts | length' "$GENESIS_FILE" 2>/dev/null || echo 0)
  BALANCE_COUNT=$(jq '.app_state.bank.balances | length' "$GENESIS_FILE" 2>/dev/null || echo 0)
  
  echo "Current genesis stats:"
  echo "  Accounts: $ACCOUNT_COUNT"
  echo "  Balances: $BALANCE_COUNT"
  echo ""
  
  if [ "$ACCOUNT_COUNT" -eq 0 ]; then
    print_info "No accounts found in genesis file"
    exit 0
  fi
  
  # Calculate number of files needed
  FILES_NEEDED=$(( (ACCOUNT_COUNT + ACCOUNTS_PER_FILE - 1) / ACCOUNTS_PER_FILE ))
  echo "Will create $FILES_NEEDED account files"
  echo ""
  
  # Confirm with user
  read -p "Continue with splitting? (y/n) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted"
    exit 0
  fi
  
  # Run the split using embedded Python
  echo ""
  print_info "Splitting genesis..."
  
  run_python_cmd "
import json
import os
from pathlib import Path

genesis_path = Path('$GENESIS_FILE')
output_dir = Path('$OUTPUT_DIR')
accounts_per_file = $ACCOUNTS_PER_FILE

# Load genesis file
with open(genesis_path, 'r') as f:
    genesis = json.load(f)

# Extract accounts and balances
accounts = genesis.get('app_state', {}).get('auth', {}).get('accounts', [])
balances = genesis.get('app_state', {}).get('bank', {}).get('balances', [])

# Determine base name
base_name = genesis_path.stem.replace('.genesis', '')
if base_name == 'genesis':
    base_name = 'network'

# Create a mapping of addresses to balances
balance_map = {bal['address']: bal for bal in balances}

# Save the main genesis without accounts/balances
main_genesis = genesis.copy()
main_genesis['app_state']['auth']['accounts'] = []
main_genesis['app_state']['bank']['balances'] = []

# Save main genesis file
main_path = output_dir / f'{base_name}.genesis.json'
with open(main_path, 'w') as f:
    json.dump(main_genesis, f, indent=2)

print(f'Created main genesis: {main_path}')

# Split and save accounts
files_created = 0
for i in range(0, len(accounts), accounts_per_file):
    chunk_idx = (i // accounts_per_file) + 1
    chunk_accounts = accounts[i:i + accounts_per_file]
    
    # Get corresponding balances
    chunk_balances = []
    for account in chunk_accounts:
        addr = account.get('address')
        if addr in balance_map:
            chunk_balances.append(balance_map[addr])
    
    # Create account chunk file
    chunk_data = {
        'accounts': chunk_accounts,
        'balances': chunk_balances
    }
    
    chunk_path = output_dir / f'{base_name}.genesis.accounts.{chunk_idx}.json'
    with open(chunk_path, 'w') as f:
        json.dump(chunk_data, f, indent=2)
    
    files_created += 1
    print(f'Created account file {chunk_idx}: {len(chunk_accounts)} accounts, {len(chunk_balances)} balances')

print(f'\\nSplit complete:')
print(f'  Total accounts: {len(accounts)}')
print(f'  Total balances: {len(balances)}')
print(f'  Files created: {files_created}')
"
  
  if [ $? -eq 0 ]; then
    echo ""
    print_header "Split completed successfully!"
    echo "Files created in: $OUTPUT_DIR"
    echo ""
    echo "The network scripts will automatically detect and use"
    echo "these split files when starting the network."
  else
    print_error "Failed to split genesis"
    exit 1
  fi
}

# -----------------------------
# Merge Command
# -----------------------------
cmd_merge() {
  local GENESIS_FILE=""
  local ACCOUNTS_DIR=""
  local OUTPUT=""
  
  while [[ $# -gt 0 ]]; do
    case $1 in
      -f|--file)
        GENESIS_FILE="$2"
        shift 2
        ;;
      -d|--accounts-dir)
        ACCOUNTS_DIR="$2"
        shift 2
        ;;
      -o|--output)
        OUTPUT="$2"
        shift 2
        ;;
      *)
        print_error "Unknown option for merge: $1"
        usage
        ;;
    esac
  done
  
  if [ -z "$GENESIS_FILE" ]; then
    print_error "Must specify --file for merge"
    usage
  fi
  
  if [ ! -f "$GENESIS_FILE" ]; then
    print_error "Genesis file not found: $GENESIS_FILE"
    exit 1
  fi
  
  if [ -z "$OUTPUT" ]; then
    OUTPUT="$GENESIS_FILE"
  fi
  
  if [ -z "$ACCOUNTS_DIR" ]; then
    ACCOUNTS_DIR="$(dirname "$GENESIS_FILE")"
  fi
  
  print_header "Genesis Account Merger"
  echo "Main genesis file: $GENESIS_FILE"
  echo "Accounts directory: $ACCOUNTS_DIR"
  echo "Output file: $OUTPUT"
  echo ""
  
  print_info "Merging accounts..."
  
  run_python_cmd "
import json
import glob
from pathlib import Path

genesis_path = Path('$GENESIS_FILE')
accounts_dir = Path('$ACCOUNTS_DIR')
output_path = Path('$OUTPUT')

# Load main genesis
with open(genesis_path, 'r') as f:
    genesis = json.load(f)

# Determine base name
base_name = genesis_path.stem.replace('.genesis', '')
if base_name == 'genesis':
    base_name = 'network'

# Find and sort account files
pattern = str(accounts_dir / f'{base_name}.genesis.accounts.*.json')
account_files = sorted(glob.glob(pattern), 
                      key=lambda x: int(Path(x).stem.split('.')[-1]))

if not account_files:
    print(f'No account files found matching pattern: {pattern}')
    import sys
    sys.exit(0)

# Merge accounts and balances
all_accounts = []
all_balances = []

for account_file in account_files:
    with open(account_file, 'r') as f:
        chunk_data = json.load(f)
    
    all_accounts.extend(chunk_data.get('accounts', []))
    all_balances.extend(chunk_data.get('balances', []))
    
    print(f'Loaded {len(chunk_data.get(\"accounts\", []))} accounts from {Path(account_file).name}')

# Update genesis with merged data
if 'app_state' not in genesis:
    genesis['app_state'] = {}
if 'auth' not in genesis['app_state']:
    genesis['app_state']['auth'] = {}
if 'bank' not in genesis['app_state']:
    genesis['app_state']['bank'] = {}

genesis['app_state']['auth']['accounts'] = all_accounts
genesis['app_state']['bank']['balances'] = all_balances

# Save merged genesis
with open(output_path, 'w') as f:
    json.dump(genesis, f, indent=2)

print(f'\\nTotal merged: {len(all_accounts)} accounts, {len(all_balances)} balances')
print(f'Merged genesis saved to: {output_path}')
"
  
  if [ $? -eq 0 ]; then
    print_success "Merge completed successfully!"
  else
    print_error "Failed to merge genesis"
    exit 1
  fi
}

# -----------------------------
# Load Command
# -----------------------------
cmd_load() {
  local GENESIS_FILE=""
  local OUTPUT=""
  
  while [[ $# -gt 0 ]]; do
    case $1 in
      -f|--file)
        GENESIS_FILE="$2"
        shift 2
        ;;
      -o|--output)
        OUTPUT="$2"
        shift 2
        ;;
      *)
        print_error "Unknown option for load: $1"
        usage
        ;;
    esac
  done
  
  if [ -z "$GENESIS_FILE" ]; then
    print_error "Must specify --file for load"
    usage
  fi
  
  if [ ! -f "$GENESIS_FILE" ]; then
    print_error "Genesis file not found: $GENESIS_FILE"
    exit 1
  fi
  
  print_info "Loading genesis with automatic account merging..."
  
  # Check if accounts already exist in genesis
  ACCOUNT_COUNT=$(jq '.app_state.auth.accounts | length' "$GENESIS_FILE" 2>/dev/null || echo 0)
  
  if [ "$ACCOUNT_COUNT" -gt 0 ]; then
    print_info "Genesis already has $ACCOUNT_COUNT accounts"
    if [ -n "$OUTPUT" ] && [ "$OUTPUT" != "$GENESIS_FILE" ]; then
      cp "$GENESIS_FILE" "$OUTPUT"
      print_success "Genesis copied to: $OUTPUT"
    fi
    exit 0
  fi
  
  # Try to merge split accounts
  local GENESIS_DIR="$(dirname "$GENESIS_FILE")"
  local BASE_NAME="$(basename "$GENESIS_FILE" .genesis.json)"
  if [ "$BASE_NAME" = "$(basename "$GENESIS_FILE")" ]; then
    BASE_NAME="$(basename "$GENESIS_FILE" .json)"
  fi
  
  local ACCOUNT_FILES=$(ls "${GENESIS_DIR}/${BASE_NAME}.genesis.accounts."*.json 2>/dev/null | head -n 1)
  
  if [ -z "$ACCOUNT_FILES" ]; then
    print_info "No split account files found"
    if [ -n "$OUTPUT" ] && [ "$OUTPUT" != "$GENESIS_FILE" ]; then
      cp "$GENESIS_FILE" "$OUTPUT"
    fi
    exit 0
  fi
  
  # Merge the accounts
  if [ -z "$OUTPUT" ]; then
    OUTPUT="/tmp/merged_genesis_$$.json"
    TEMP_OUTPUT=true
  fi
  
  cmd_merge -f "$GENESIS_FILE" -o "$OUTPUT"
  
  if [ "$TEMP_OUTPUT" = true ]; then
    cat "$OUTPUT"
    rm -f "$OUTPUT"
  else
    print_success "Complete genesis saved to: $OUTPUT"
  fi
}

# -----------------------------
# Check Command
# -----------------------------
cmd_check() {
  local NETWORK=""
  local GENESIS_FILE=""
  
  while [[ $# -gt 0 ]]; do
    case $1 in
      -n|--network)
        NETWORK="$2"
        shift 2
        ;;
      -f|--file)
        GENESIS_FILE="$2"
        shift 2
        ;;
      *)
        print_error "Unknown option for check: $1"
        usage
        ;;
    esac
  done
  
  # Determine genesis file
  if [ -n "$GENESIS_FILE" ]; then
    if [ ! -f "$GENESIS_FILE" ]; then
      print_error "Genesis file not found: $GENESIS_FILE"
      exit 1
    fi
  elif [ -n "$NETWORK" ]; then
    GENESIS_FILE=$(find_network_genesis "$NETWORK") || exit 1
  else
    print_error "Must specify either --network or --file"
    usage
  fi
  
  local GENESIS_DIR="$(dirname "$GENESIS_FILE")"
  local GENESIS_NAME="$(basename "$GENESIS_FILE")"
  local BASE_NAME="${GENESIS_NAME%.genesis.json}"
  
  # Handle already split pattern
  if [[ "$BASE_NAME" == *".genesis" ]]; then
    BASE_NAME="${BASE_NAME%.genesis}"
  fi
  
  # Handle regular pattern
  if [ "$BASE_NAME" = "$GENESIS_NAME" ]; then
    BASE_NAME="${GENESIS_NAME%.json}"
  fi
  
  local ACCOUNT_PATTERN="${GENESIS_DIR}/${BASE_NAME}.genesis.accounts.*.json"
  local ACCOUNT_FILES=$(ls $ACCOUNT_PATTERN 2>/dev/null)
  
  if [ -n "$ACCOUNT_FILES" ]; then
    print_success "Genesis has split account files:"
    local COUNT=0
    for FILE in $ACCOUNT_FILES; do
      COUNT=$((COUNT + 1))
      local ACCOUNTS=$(jq '.accounts | length' "$FILE" 2>/dev/null || echo 0)
      local BALANCES=$(jq '.balances | length' "$FILE" 2>/dev/null || echo 0)
      echo "  $(basename "$FILE"): $ACCOUNTS accounts, $BALANCES balances"
    done
    echo "Total files: $COUNT"
  else
    print_info "No split account files found for genesis"
    # Check if accounts are in main file
    local ACCOUNTS=$(jq '.app_state.auth.accounts | length' "$GENESIS_FILE" 2>/dev/null || echo 0)
    if [ "$ACCOUNTS" -gt 0 ]; then
      echo "Main genesis contains $ACCOUNTS accounts"
    fi
  fi
}

# -----------------------------
# Count Command
# -----------------------------
cmd_count() {
  local NETWORK=""
  local GENESIS_FILE=""
  
  while [[ $# -gt 0 ]]; do
    case $1 in
      -n|--network)
        NETWORK="$2"
        shift 2
        ;;
      -f|--file)
        GENESIS_FILE="$2"
        shift 2
        ;;
      *)
        print_error "Unknown option for count: $1"
        usage
        ;;
    esac
  done
  
  # Determine genesis file
  if [ -n "$GENESIS_FILE" ]; then
    if [ ! -f "$GENESIS_FILE" ]; then
      print_error "Genesis file not found: $GENESIS_FILE"
      exit 1
    fi
  elif [ -n "$NETWORK" ]; then
    GENESIS_FILE=$(find_network_genesis "$NETWORK") || exit 1
  else
    print_error "Must specify either --network or --file"
    usage
  fi
  
  local GENESIS_DIR="$(dirname "$GENESIS_FILE")"
  local GENESIS_NAME="$(basename "$GENESIS_FILE")"
  local BASE_NAME="${GENESIS_NAME%.genesis.json}"
  
  # Handle already split pattern
  if [[ "$BASE_NAME" == *".genesis" ]]; then
    BASE_NAME="${BASE_NAME%.genesis}"
  fi
  
  # Handle regular pattern
  if [ "$BASE_NAME" = "$GENESIS_NAME" ]; then
    BASE_NAME="${GENESIS_NAME%.json}"
  fi
  
  local TOTAL_ACCOUNTS=0
  local TOTAL_BALANCES=0
  
  # Check for split account files first
  local ACCOUNT_PATTERN="${GENESIS_DIR}/${BASE_NAME}.genesis.accounts.*.json"
  local ACCOUNT_FILES=$(ls $ACCOUNT_PATTERN 2>/dev/null)
  
  if [ -n "$ACCOUNT_FILES" ]; then
    # Count from split files
    print_info "Counting from split files..."
    for FILE in $ACCOUNT_FILES; do
      local ACCOUNTS=$(jq '.accounts | length' "$FILE" 2>/dev/null || echo 0)
      local BALANCES=$(jq '.balances | length' "$FILE" 2>/dev/null || echo 0)
      TOTAL_ACCOUNTS=$((TOTAL_ACCOUNTS + ACCOUNTS))
      TOTAL_BALANCES=$((TOTAL_BALANCES + BALANCES))
    done
    echo "Source: Split files"
  else
    # Count from main genesis
    TOTAL_ACCOUNTS=$(jq '.app_state.auth.accounts | length' "$GENESIS_FILE" 2>/dev/null || echo 0)
    TOTAL_BALANCES=$(jq '.app_state.bank.balances | length' "$GENESIS_FILE" 2>/dev/null || echo 0)
    echo "Source: Main genesis file"
  fi
  
  print_header "Genesis Account Count"
  echo "File: $GENESIS_FILE"
  echo "Total accounts: $TOTAL_ACCOUNTS"
  echo "Total balances: $TOTAL_BALANCES"
}

# -----------------------------
# Utility Functions (for sourcing)
# -----------------------------

# Load genesis with automatic account merging
# Usage: load_genesis_with_accounts <genesis_path> [temp_output_path]
# Returns: Path to the complete genesis file (either original or merged)
load_genesis_with_accounts() {
  local GENESIS_PATH="$1"
  local OUTPUT_PATH="${2:-}"
  
  if [ ! -f "$GENESIS_PATH" ]; then
    echo "Error: Genesis file not found: $GENESIS_PATH" >&2
    return 1
  fi
  
  # Get the directory and base name
  local GENESIS_DIR="$(dirname "$GENESIS_PATH")"
  local GENESIS_NAME="$(basename "$GENESIS_PATH")"
  local BASE_NAME="${GENESIS_NAME%.genesis.json}"
  
  # Handle already split pattern
  if [[ "$BASE_NAME" == *".genesis" ]]; then
    BASE_NAME="${BASE_NAME%.genesis}"
  fi
  
  # Handle regular pattern
  if [ "$BASE_NAME" = "$GENESIS_NAME" ]; then
    BASE_NAME="${GENESIS_NAME%.json}"
  fi
  
  # Check if account files exist
  local ACCOUNT_PATTERN="${GENESIS_DIR}/${BASE_NAME}.genesis.accounts.*.json"
  local ACCOUNT_FILES=$(ls $ACCOUNT_PATTERN 2>/dev/null | head -n 1)
  
  if [ -z "$ACCOUNT_FILES" ]; then
    # No split files, use original genesis
    echo "$GENESIS_PATH"
    return 0
  fi
  
  # Account files exist, need to merge
  echo "Found split account files for $BASE_NAME, merging..." >&2
  
  # Determine output path
  if [ -z "$OUTPUT_PATH" ]; then
    OUTPUT_PATH="/tmp/merged_genesis_$$.json"
  fi
  
  # Merge using embedded Python
  python3 -c "
import json
import glob
from pathlib import Path

genesis_path = Path('$GENESIS_PATH')
accounts_dir = Path('$GENESIS_DIR')
output_path = Path('$OUTPUT_PATH')

# Load main genesis
with open(genesis_path, 'r') as f:
    genesis = json.load(f)

# Find and sort account files
pattern = str(accounts_dir / '${BASE_NAME}.genesis.accounts.*.json')
account_files = sorted(glob.glob(pattern), 
                      key=lambda x: int(Path(x).stem.split('.')[-1]))

if account_files:
    # Merge accounts and balances
    all_accounts = []
    all_balances = []
    
    for account_file in account_files:
        with open(account_file, 'r') as f:
            chunk_data = json.load(f)
        all_accounts.extend(chunk_data.get('accounts', []))
        all_balances.extend(chunk_data.get('balances', []))
    
    # Update genesis
    genesis.setdefault('app_state', {}).setdefault('auth', {})['accounts'] = all_accounts
    genesis.setdefault('app_state', {}).setdefault('bank', {})['balances'] = all_balances

# Save merged genesis
with open(output_path, 'w') as f:
    json.dump(genesis, f, indent=2)
" >&2
  
  if [ $? -ne 0 ]; then
    echo "Error: Failed to merge genesis accounts" >&2
    return 1
  fi
  
  echo "$OUTPUT_PATH"
  return 0
}

# Split genesis accounts into multiple files
# Usage: split_genesis_accounts <genesis_path> [accounts_per_file] [output_dir]
split_genesis_accounts() {
  local GENESIS_PATH="$1"
  local ACCOUNTS_PER_FILE="${2:-5000}"
  local OUTPUT_DIR="${3:-$(dirname "$GENESIS_PATH")}"
  
  if [ ! -f "$GENESIS_PATH" ]; then
    echo "Error: Genesis file not found: $GENESIS_PATH" >&2
    return 1
  fi
  
  echo "Splitting genesis accounts from $GENESIS_PATH..." >&2
  
  # Use the split command internally
  cmd_split -f "$GENESIS_PATH" -a "$ACCOUNTS_PER_FILE" -o "$OUTPUT_DIR"
  
  return $?
}

# Check if genesis has split account files
# Usage: has_split_accounts <genesis_path>
# Returns: 0 if split files exist, 1 otherwise
has_split_accounts() {
  local GENESIS_PATH="$1"
  local GENESIS_DIR="$(dirname "$GENESIS_PATH")"
  local GENESIS_NAME="$(basename "$GENESIS_PATH")"
  local BASE_NAME="${GENESIS_NAME%.genesis.json}"
  
  # Handle already split pattern
  if [[ "$BASE_NAME" == *".genesis" ]]; then
    BASE_NAME="${BASE_NAME%.genesis}"
  fi
  
  # Handle regular pattern
  if [ "$BASE_NAME" = "$GENESIS_NAME" ]; then
    BASE_NAME="${GENESIS_NAME%.json}"
  fi
  
  local ACCOUNT_PATTERN="${GENESIS_DIR}/${BASE_NAME}.genesis.accounts.*.json"
  local ACCOUNT_FILES=$(ls $ACCOUNT_PATTERN 2>/dev/null | head -n 1)
  
  if [ -n "$ACCOUNT_FILES" ]; then
    return 0
  else
    return 1
  fi
}

# Count accounts in genesis (including split files)
# Usage: count_genesis_accounts <genesis_path>
count_genesis_accounts() {
  local GENESIS_PATH="$1"
  local GENESIS_DIR="$(dirname "$GENESIS_PATH")"
  local GENESIS_NAME="$(basename "$GENESIS_PATH")"
  local BASE_NAME="${GENESIS_NAME%.genesis.json}"
  
  # Handle already split pattern
  if [[ "$BASE_NAME" == *".genesis" ]]; then
    BASE_NAME="${BASE_NAME%.genesis}"
  fi
  
  # Handle regular pattern
  if [ "$BASE_NAME" = "$GENESIS_NAME" ]; then
    BASE_NAME="${GENESIS_NAME%.json}"
  fi
  
  local TOTAL=0
  
  # Check for split account files first
  local ACCOUNT_PATTERN="${GENESIS_DIR}/${BASE_NAME}.genesis.accounts.*.json"
  local ACCOUNT_FILES=$(ls $ACCOUNT_PATTERN 2>/dev/null)
  
  if [ -n "$ACCOUNT_FILES" ]; then
    # Count from split files
    for FILE in $ACCOUNT_FILES; do
      local COUNT=$(jq '.accounts | length' "$FILE" 2>/dev/null || echo 0)
      TOTAL=$((TOTAL + COUNT))
    done
  else
    # Count from main genesis
    TOTAL=$(jq '.app_state.auth.accounts | length' "$GENESIS_PATH" 2>/dev/null || echo 0)
  fi
  
  echo "$TOTAL"
}

# -----------------------------
# Main Entry Point
# -----------------------------

# If script is being sourced, don't execute main
if [ "${BASH_SOURCE[0]}" != "${0}" ]; then
  # Script is being sourced - just export functions
  return 0
fi

# Parse command
COMMAND="${1:-}"
shift || true

case "$COMMAND" in
  split)
    cmd_split "$@"
    ;;
  merge)
    cmd_merge "$@"
    ;;
  load)
    cmd_load "$@"
    ;;
  check)
    cmd_check "$@"
    ;;
  count)
    cmd_count "$@"
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    print_error "Unknown command: $COMMAND"
    usage
    ;;
esac