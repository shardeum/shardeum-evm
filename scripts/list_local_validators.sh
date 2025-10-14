#!/usr/bin/env bash
# list_local_validators.sh
# Enumerate local node directories (.local/node*) and print:
#   node index | key name | account (delegator) address | valoper address (if key exists)
# Helps map a known valoper (e.g., shardeumvaloper1...) back to the account holding commission withdrawals.

set -euo pipefail

REPO_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")"/.. && pwd)"
LOCAL_DIR="$REPO_ROOT/.local"
BINARY=${BINARY:-"$REPO_ROOT/build/shardeumd"}
KEYRING_BACKEND=${KEYRING_BACKEND:-test}

if [[ ! -x "$BINARY" ]]; then
  if command -v shardeumd >/dev/null 2>&1; then
    BINARY=$(command -v shardeumd)
  else
    echo "shardeumd binary not found at $BINARY (build with 'make build')." >&2
    exit 1
  fi
fi

printf "%5s | %-20s | %-44s | %-60s\n" NODE KEY ACCOUNT VALOPER
printf -- "%.0s-" {1..140}; echo

shopt -s nullglob
for dir in "$LOCAL_DIR"/node*; do
  [[ -d "$dir" ]] || continue
  base=$(basename "$dir")
  idx=${base#node}
  if [[ "$idx" == "0" ]]; then
    key="validator"
  else
    key="validator-node$idx"
  fi
  set +e
  ACC=$($BINARY keys show "$key" --keyring-backend "$KEYRING_BACKEND" --home "$dir" -a 2>/dev/null)
  rc=$?
  set -e
  if [[ $rc -ne 0 || -z "$ACC" ]]; then
    continue
  fi
  # Derive valoper using CLI bech switch (if supported). Fallback to python decode/encode if needed.
  set +e
  VALOPER=$($BINARY keys show "$key" --keyring-backend "$KEYRING_BACKEND" --home "$dir" --bech val -a 2>/dev/null)
  if [[ -z "$VALOPER" ]]; then
    # Minimal bech32 re-encode fallback using local_testomatic's logic (extract via python if necessary)
    VALOPER=$(python - "$ACC" <<'PY'
import sys
HRP_VALOPER = 'shardeumvaloper'
BECH32_CHARSET = 'qpzry9x8gf2tvdw0s3jn54khce6mua7l'
def bech32_polymod(values):
    GENERATORS = [0x3b6a57b2,0x26508e6d,0x1ea119fa,0x3d4233dd,0x2a1462b3]
    chk = 1
    for v in values:
        b = (chk >> 25)
        chk = (chk & 0x1ffffff) << 5 ^ v
        for i in range(5):
            if (b >> i) & 1:
                chk ^= GENERATORS[i]
    return chk
def bech32_hrp_expand(s):
    return [ord(x) >> 5 for x in s] + [0] + [ord(x) & 31 for x in s]
def bech32_create_checksum(hrp, data):
    values = bech32_hrp_expand(hrp) + data
    polymod = bech32_polymod(values + [0,0,0,0,0,0]) ^ 1
    return [(polymod >> 5 * (5 - i)) & 31 for i in range(6)]
def bech32_decode(bech):
    if any(ord(x) < 33 or ord(x) > 126 for x in bech):
        return None, None
    bech = bech.lower()
    pos = bech.rfind('1')
    if pos < 1 or pos + 7 > len(bech):
        return None, None
    hrp = bech[:pos]
    data = [BECH32_CHARSET.find(c) for c in bech[pos+1:]]
    if any(x == -1 for x in data):
        return None, None
    return hrp, data[:-6]
def bech32_encode(hrp, data):
    combined = data + bech32_create_checksum(hrp, data)
    return hrp + '1' + ''.join(BECH32_CHARSET[d] for d in combined)
def convertbits(data, frombits, tobits, pad=True):
    acc = 0; bits = 0; ret = []
    maxv = (1 << tobits) - 1
    for value in data:
        if value < 0 or value >> frombits:
            return None
        acc = (acc << frombits) | value
        bits += frombits
        while bits >= tobits:
            bits -= tobits
            ret.append((acc >> bits) & maxv)
    if pad and bits:
        ret.append((acc << (tobits - bits)) & maxv)
    elif bits >= frombits or ((acc << (tobits - bits)) & maxv):
        return None
    return ret
acc=sys.argv[1].strip()
hrp,data = bech32_decode(acc)
if hrp is None:
    print('')
    sys.exit(0)
five_bits = data
print(bech32_encode(HRP_VALOPER, five_bits))
PY
    )
  fi
  set -e
  printf "%5s | %-20s | %-44s | %-60s\n" "node$idx" "$key" "$ACC" "$VALOPER"
done

echo
echo "Tip: To find the account for a given valoper, match the valoper column. Commission withdrawals pay into the ACCOUNT column." 
