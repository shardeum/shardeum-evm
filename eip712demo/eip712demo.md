# EIP-712 Demo and Testing Tools

This directory contains a prototype implementation and testing tools for EIP-712 signature support in Shardeum. These tools demonstrate how MetaMask users can sign Cosmos transactions using EIP-712 typed data instead of requiring a Cosmos wallet.

**Note:** These files are intended to be migrated to a separate repository but can be tested here for now.

## Directory Structure

```
eip712demo/
├── broadcast-eip712/          # CLI tool to broadcast EIP-712 signed transactions
│   ├── main.go               # Go implementation
│   └── README.md             # Tool-specific documentation
├── tools/                    # Debugging and analysis utilities
│   ├── debug_suite.go        # Signature verification test suite
│   └── log_report.go         # Log analyzer for EIP-712 transactions
├── metamask-eip712-delegate.html  # Web demo for MetaMask signing
├── eip712-helpers.js         # JavaScript helper library
├── signed-tx1.json           # Example signed transaction
└── README.md                 # Overview documentation
```

## Building the Tools

### Build EIP-712 Broadcast Tool Only

```bash
make eip712
# or
make build-eip712
```

This builds `broadcast-eip712` binary to `./build/broadcast-eip712`

### Build Everything (Node + EIP-712 Tool)

```bash
make build
```

This builds both `shardeumd` and `broadcast-eip712` binaries.

## Components

### 1. MetaMask Web Demo (`metamask-eip712-delegate.html`)

**Purpose:** Interactive web interface for signing Cosmos delegation transactions with MetaMask using EIP-712.

**Usage:**

1. Start a local HTTP server:
   ```bash
   python3 -m http.server 8000
   ```

2. Open browser to: `http://localhost:8000/eip712demo/metamask-eip712-delegate.html`

3. Connect MetaMask wallet (ensure it's on the correct network)

4. Enter delegation details:
   - **Validator Address**: Shardeum validator address (e.g., `shardeumvaloper1...`)
   - **Amount**: Amount to delegate in SHM
   - **Account Number**: Your account number (query from chain)
   - **Sequence**: Current sequence number (starts at 0)

5. Click "Sign with MetaMask" - this will:
   - Generate EIP-712 typed data from the delegation message
   - Prompt MetaMask to sign the structured data
   - Display the signature and typed data
   - Save to `signed-tx1.json` for use with broadcast tool

**Features:**
- Real-time EIP-712 typed data preview
- Automatic signature formatting (R||S||V)
- JSON export for broadcasting
- Address derivation from signature

**JavaScript Helpers (`eip712-helpers.js`):**
- `createMsgDelegate()`: Constructs delegation message
- `createEIP712TypedData()`: Generates EIP-712 structure
- `signWithMetaMask()`: Handles MetaMask signing flow
- Address conversion utilities

### 2. Broadcast Tool (`broadcast-eip712/`)

**Purpose:** CLI tool to submit EIP-712 signed transactions to the Shardeum chain via REST API.

**Build:**
```bash
make eip712
```

**Usage:**

```bash
./build/broadcast-eip712 signed-tx1.json \
  --broadcast \
  --node http://localhost:1317 \
  --cosmos-chain-id shardeum_8117-1 \
  --evm-chain-id 8117
```

**Flags:**
- `--broadcast`: Actually submit to network (omit for dry-run)
- `--node`: REST API endpoint (default: http://localhost:1317)
- `--cosmos-chain-id`: Cosmos chain ID (default: shardeum_8117-1)
- `--evm-chain-id`: EVM chain ID for signature verification (default: 8117)

**Process:**
1. Parses signed transaction JSON
2. Recovers public key from EIP-712 signature
3. Queries account info from chain
4. Builds Cosmos transaction with ExtensionOptionsWeb3Tx
5. Broadcasts to chain via `/cosmos/tx/v1beta1/txs` endpoint

**Output:**
- ✅ Step-by-step progress indicators
- 🔍 Recovered address verification
- 📤 Transaction hash on success
- ❌ Detailed error messages on failure

### 3. Debug Tools (`tools/`)

#### `debug_suite.go`

**Purpose:** Comprehensive signature verification test suite for EIP-712 transactions.

**Usage:**
```bash
go run ./eip712demo/tools/debug_suite.go -signed signed-tx1.json
```

**Optional Flags:**
- `-typed`: Print reconstructed typed data JSON

**What it checks:**
- ✅ Signature length and format (65 bytes R||S||V)
- ✅ Recovery ID validity (0 or 1)
- ✅ R and S values within curve order
- ✅ S in lower half (malleability check)
- ✅ Public key recovery from signature
- ✅ `secp256k1.VerifySignature` (compressed pubkey)
- ✅ `secp256k1.VerifySignature` (uncompressed pubkey)
- ✅ `crypto.VerifySignature` (go-ethereum)
- ✅ `ecdsa.Verify` (standard library)
- ✅ EIP-712 hash calculation (decimal chainId)
- ✅ Address derivation (Ethereum & Cosmos formats)
- ✅ Typed data structure validation
- ✅ Message content verification

**Output:**
```
Loaded signed-tx1.json
Standard domain hash: a04f8864db36d01...
Decimal domain hash:  30786130346638...
Custom final hash:   9c7e065be31d96bd...
secp256k1.VerifySignature (compressed pubkey): true
ecdsa.Verify: true
Recovered Shardeum address: shardeum1jhfxwuu23vt839fcu5fxk4a5ww4hyx8pxqty3s

=== Check Summary ===
Total: 40, Passed: 40, Failed: 0
[PASS] Signature length is 65 bytes
[PASS] ecdsa.Verify passes
[PASS] Recovered shardeum address matches signed
...
```

#### `log_report.go`

**Purpose:** Scans node logs for EIP-712 transaction attempts and extracts key debugging information.

**Usage:**
```bash
go run ./eip712demo/tools/log_report.go -log .local/node0/node.log
```

**Optional Flags:**
- `-latest`: Only show most recent transaction attempt (default: true)

**What it extracts:**
- 🔵 ANTE HANDLER entry point (with BUILD version)
- 🟡 Reconstructed typed data (chain-side)
- 🔴 Calculated EIP-712 hash
- 🟣 Fee payer signature from extension
- 🟢 Verification inputs (pubkeys, hash, signature)
- ✅/❌ Verification result

**Output:**
```
=== TX Attempt #1 ===
ANTE HANDLER at line 232
TYPED DATA at line 248
HASH at line 255
SIGNATURE at line 262
VERIFY INPUTS at line 271
RESULT at line 281: ✅✅✅ EIP-712 SIGNATURE VERIFICATION SUCCEEDED! ✅✅✅
--- Log Snippet ---
🔵🔵🔵 ANTE HANDLER ENTRY (BUILD v3) 🔵🔵🔵
🟡 CHAIN RECONSTRUCTED TYPED DATA (with decimal chainId)
🔴 CHAIN CALCULATED HASH
...
```

**Use Cases:**
- Debugging signature verification failures
- Comparing client-side vs chain-side hash calculation
- Tracking multiple transaction attempts
- Identifying which code version is running (BUILD markers)

## Testing Workflow

### 1. Start Local Network

```bash
# Start single node
./scripts/start_network.sh 1 --network local

# Verify node is running
curl http://localhost:1317/cosmos/base/tendermint/v1beta1/node_info
```

### 2. Create Signed Transaction

**Option A: Using Web Demo**
1. Open `metamask-eip712-delegate.html` in browser
2. Connect MetaMask
3. Fill in delegation details
4. Sign and download `signed-tx1.json`

**Option B: Manual JSON (for testing)**
```json
{
  "typedData": { ... },
  "signature": "0x...",
  "ethAddress": "0x...",
  "bech32Address": "shardeum1...",
  "validatorAddress": "shardeumvaloper1...",
  "accountNumber": "40",
  "sequence": "0"
}
```

### 3. Verify Signature Locally

```bash
go run ./eip712demo/tools/debug_suite.go -signed signed-tx1.json
```

Expected: All checks pass (✅)

### 4. Broadcast Transaction

```bash
./build/broadcast-eip712 signed-tx1.json \
  --broadcast \
  --node http://localhost:1317 \
  --cosmos-chain-id shardeum_8117-1 \
  --evm-chain-id 8117
```

Expected output:
```
✅ Public key recovered and verified
✅ Transaction encoded: 480 bytes
📤 Broadcasting to network...
✅ Transaction successful! Hash: ABC123...
```

### 5. Verify Transaction

```bash
# Check account sequence incremented
curl -s http://localhost:1317/cosmos/auth/v1beta1/accounts/shardeum1jhfxwuu23vt839fcu5fxk4a5ww4hyx8pxqty3s | jq '.account.sequence'
# Should show: "1" (was "0")

# Check delegation created
curl -s "http://localhost:1317/cosmos/staking/v1beta1/delegations/shardeum1jhfxwuu23vt839fcu5fxk4a5ww4hyx8pxqty3s" | jq '.delegation_responses[0]'
```

### 6. Analyze Logs (if issues)

```bash
go run ./eip712demo/tools/log_report.go -log .local/node0/node.log
```

Look for:
- ❌ Signature verification failures
- Hash mismatches
- Pubkey format issues

## Technical Details

### EIP-712 Signature Flow

1. **Client Side (MetaMask)**
   - User initiates Cosmos transaction (e.g., delegate)
   - Transaction converted to EIP-712 typed data
   - User signs with MetaMask (secp256k1 signature)
   - Signature format: 65 bytes (R||S||V)

2. **Chain Side (Ante Handlers)**
   - Transaction arrives with `ExtensionOptionsWeb3Tx`
   - Custom ante decorator extracts EIP-712 signature
   - Reconstructs typed data from transaction
   - Calculates hash with decimal chainId
   - Recovers public key from signature
   - Verifies signature using **compressed** public key
   - Sets context flag to skip standard Cosmos sig verification

3. **Key Fixes Implemented**
   - **ChainId Format**: Use decimal string (`"8117"`) not hex (`"0x1fb5"`)
   - **Public Key Format**: Use compressed (33 bytes) for verification
   - **Signature Bypass**: Skip Cosmos signature check for EIP-712 txs

### Chain ID Formats

- **Cosmos Chain ID**: `shardeum_8117-1` (full format)
- **EVM Chain ID (decimal)**: `8117` (for EIP-712)
- **EVM Chain ID (hex)**: `0x1fb5` (MetaMask display only)

### Address Formats

- **Ethereum**: `0x95D267738A8B16789538e5126B57b473AB7218E1`
- **Cosmos (generic)**: `cosmos1jhfxwuu23vt839fcu5fxk4a5ww4hyx8p7sat95`
- **Shardeum**: `shardeum1jhfxwuu23vt839fcu5fxk4a5ww4hyx8pxqty3s`

All formats derived from same secp256k1 public key.

## Troubleshooting

### Signature Verification Failed

**Symptoms:**
```
Error: signature verification failed; please verify account number (40) and chain-id
```

**Debug Steps:**
1. Run debug suite: `go run ./eip712demo/tools/debug_suite.go -signed signed-tx1.json`
2. Check if all verification methods pass
3. Analyze node logs: `go run ./eip712demo/tools/log_report.go -log .local/node0/node.log`
4. Compare client hash vs chain hash

**Common Causes:**
- ChainId mismatch (decimal vs hex)
- Wrong account number or sequence
- Invalid validator address
- Network restart (validator addresses change)

### MetaMask Connection Issues

**Symptoms:**
- "MetaMask not detected"
- Wrong network

**Solutions:**
1. Install MetaMask extension
2. Add custom network:
   - Network Name: Shardeum Local
   - RPC URL: http://localhost:8545
   - Chain ID: 8117
   - Currency: SHM

### Build Errors

**Symptoms:**
```
Error: cannot find module
```

**Solutions:**
```bash
# Update dependencies
go mod tidy

# Clean build
rm -rf build/
make build
make eip712
```

## Future Migration

These tools are prototypes and will be migrated to a separate repository for:
- Easier distribution
- Independent versioning
- Standalone documentation
- npm package for JavaScript helpers
- Docker images for testing

## Related Documentation

- `../docs/onboard6.md` - Latest debugging session
- `../docs/EIP712_SUPPORT.md` - Integration guide
- `broadcast-eip712/README.md` - CLI tool details
- `README.md` - Quick start guide
