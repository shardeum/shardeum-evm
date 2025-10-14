#!/usr/bin/env bash
# list_local_rewards.sh
# Enumerate local validator keys and display:
#   node index
#   key name
#   account (delegator) address
#   valoper address
#   bank balance (ashm)
#   un-withdrawn commission (ashm)
#   un-withdrawn delegator rewards (ashm) for (account, valoper)
#
# Requirements: jq
# Optional env vars:
#   BINARY (default: ./build/shardeumd or shardeumd from PATH)
#   KEYRING_BACKEND (default: test)
#   LOCAL_DIR (default: repo_root/.local)
#   DENOM (default: ashm)
#   DECIMALS (default: 18) for human-readable conversion with --human
#
# Usage:
#   ./scripts/list_local_rewards.sh          # raw base unit amounts
#   ./scripts/list_local_rewards.sh --human  # adds human token columns
#
set -euo pipefail

HUMAN=0
for arg in "$@"; do
  case "$arg" in
    --human) HUMAN=1 ; shift ;;
    *) echo "Unknown arg: $arg" >&2; exit 1 ;;
  esac
done

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required" >&2
  exit 1
fi

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
LOCAL_DIR=${LOCAL_DIR:-"$REPO_ROOT/.local"}
BAL_QUERY_REF_NODE=${REF_NODE:-0}
DENOM=${DENOM:-ashm}
DECIMALS=${DECIMALS:-18}
BINARY=${BINARY:-"$REPO_ROOT/build/shardeumd"}
KEYRING_BACKEND=${KEYRING_BACKEND:-test}

if [[ ! -x "$BINARY" ]]; then
  if command -v shardeumd >/dev/null 2>&1; then
    BINARY=$(command -v shardeumd)
  else
    echo "shardeumd binary not found (expected at $BINARY). Build with 'make build'." >&2
    exit 1
  fi
fi

convert_human() {
  # $1 = base units integer
  local v="$1"
  if [[ $HUMAN -eq 0 ]]; then
    echo "$v"
    return 0
  fi
  # Use awk for decimal scaling
  awk -v n="$v" -v d="$DECIMALS" 'BEGIN { if (n==0) {print 0; exit} scale=d; int_part = substr(n, 1, length(n)-d); if (int_part == "") int_part = 0; frac = substr(n, length(n)-d+1); if (length(frac)<d) frac=sprintf("%0"d"d", frac); # trim trailing zeros
    sub(/0+$/, "", frac); if (frac == "") printf "%s", int_part; else printf "%s.%s", int_part, frac }'
}

sum_coin_array() {
  # Read JSON coin array (objects or strings). Output total (integer base units) for target denom.
  # Accept forms:
  #   {"denom":"ashm","amount":"123"}
  #   "123ashm"
  #   "123.000000000000000000ashm" (strip decimals, truncate)
  jq -r --arg DEN "$DENOM" '
    [.[] | if type=="object" then select(.denom==$DEN) | .amount
            elif type=="string" then (capture("(?<amt>[0-9]+)(?:\\.[0-9]+)?(?<denom>[a-zA-Z0-9/]+)") | select(.denom==$DEN) | .amt)
            else empty end
    | tonumber] | add // 0' 2>/dev/null
}

extract_commission() {
  local valoper=$1
  local query_home=$2
  local json
  if ! json=$($BINARY query distribution commission "$valoper" --home "$query_home" -o json 2>/dev/null); then
    echo 0; return
  fi
  echo "$json" | jq -c '.commission.commission // []' | sum_coin_array
}

extract_rewards() {
  local acc=$1
  local valoper=$2
  local query_home=$3
  local json
  if ! json=$($BINARY query distribution rewards "$acc" "$valoper" --home "$query_home" -o json 2>/dev/null); then
    echo 0; return
  fi
  echo "$json" | jq -c '.rewards // []' | sum_coin_array
}

extract_balance() {
  local acc=$1
  local query_home=$2
  local json
  if ! json=$($BINARY query bank balances "$acc" --home "$query_home" -o json 2>/dev/null); then
    echo 0; return
  fi
  echo "$json" | jq -r --arg DEN "$DENOM" '.balances[]? | select(.denom==$DEN) | .amount' | awk '{s+=$1} END { if (s=="") s=0; print s }'
}

printf "%-6s | %-18s | %-44s | %-60s | %-18s | %-18s | %-18s" NODE KEY ACCOUNT VALOPER BAL_${DENOM} COMM_${DENOM} REW_${DENOM}
if [[ $HUMAN -eq 1 ]]; then
  printf " | %-14s | %-14s | %-14s" BAL_TOK COMM_TOK REW_TOK
fi
echo
printf -- "%.0s-" {1..200}; echo

shopt -s nullglob
REF_HOME="$LOCAL_DIR/node${BAL_QUERY_REF_NODE}"
if [[ ! -d "$REF_HOME" ]]; then
  echo "Reference node home $REF_HOME not found; set REF_NODE env var correctly." >&2
  exit 1
fi

for node_home in "$LOCAL_DIR"/node*; do
  [[ -d "$node_home" ]] || continue
  node_base=$(basename "$node_home")
  idx=${node_base#node}
  if [[ "$idx" == "0" ]]; then
    key="validator"
  else
    key="validator-node$idx"
  fi
  set +e
  ACC=$($BINARY keys show "$key" --keyring-backend "$KEYRING_BACKEND" --home "$node_home" -a 2>/dev/null)
  rc=$?
  set -e
  if [[ $rc -ne 0 || -z "$ACC" ]]; then
    continue
  fi
  set +e
  VALOPER=$($BINARY keys show "$key" --keyring-backend "$KEYRING_BACKEND" --home "$node_home" --bech val -a 2>/dev/null)
  set -e
  if [[ -z "$VALOPER" ]]; then
    # Derive valoper by replacing hrp checksum via CLI parse (simpler: attempt keys parse output forms)
    # Fallback: leave blank
    VALOPER="(unknown)"
  fi
  BAL=$(extract_balance "$ACC" "$REF_HOME")
  COMM=$(extract_commission "$VALOPER" "$REF_HOME")
  REW=$(extract_rewards "$ACC" "$VALOPER" "$REF_HOME")
  printf "%-6s | %-18s | %-44s | %-60s | %-18s | %-18s | %-18s" "node$idx" "$key" "$ACC" "$VALOPER" "$BAL" "$COMM" "$REW"
  if [[ $HUMAN -eq 1 ]]; then
    printf " | %-14s | %-14s | %-14s" "$(convert_human "$BAL")" "$(convert_human "$COMM")" "$(convert_human "$REW")"
  fi
  echo
done

echo
echo "Notes:"
echo "  BAL = current bank balance (liquid)."
echo "  COMM = unwithdrawn commission (withdraw with: shardeumd tx distribution withdraw-commission <valoper> --from <key> ...)."
echo "  REW = unwithdrawn delegator rewards for the self-delegation (withdraw with: shardeumd tx distribution withdraw-rewards <valoper> --from <key> ...)."
if [[ $HUMAN -eq 1 ]]; then
  echo "  *_TOK columns are scaled by 10^$DECIMALS for human readability."
fi
