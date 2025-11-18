# Multisig Validator Setup Using add_node.sh

This guide walks you through creating a validator with a multisig operator key using the `add_node.sh` script. The script automatically creates the multisig operator key during node setup, simplifying the process.

## Overview

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  Alice WS   │    │   Bob WS    │    │  Carol WS   │
│  (Private)  │    │  (Private)  │    │  (Private)  │
└──────┬──────┘    └──────┬──────┘    └──────┬──────┘
       │                  │                  │
       │  Public Keys     │  Public Keys     │  Public Keys
       └──────────────────┼──────────────────┘
                          │
                          ▼
                  ┌───────────────┐
                  │  add_node.sh  │
                  │  Creates      │
                  │  Multisig Key │
                  └───────┬───────┘
                          │
                          ▼
                  ┌───────────────┐
                  │ Validator Node│
                  │ (Public Keys) │
                  └───────────────┘
```

## Prerequisites

- `shardeumd` binary in PATH or set `BINARY` environment variable
- `jq` installed (`brew install jq` on macOS, `apt-get install jq` on Linux)
- A running network (seed node must be accessible)
- Signer public keys (from hardware wallets or separate machines)

## Step 1: Prepare Signer Public Keys

You need to collect public keys from all signers. Each signer should provide their public key in the following format.

### Option A: Get Public Key from Existing Key

If a signer has a key in their keyring:

```bash
# Signer runs this on their machine
shardeumd keys show alice --pubkey --keyring-backend test
```

Output example:
```json
{
  "@type": "/ethermint.crypto.v1.ethsecp256k1.PubKey",
  "key": "Agq507WWHGJCk8mf1NVkDMt6XvRtVsfFiO79lHlsInpT"
}
```

### Option B: Get Public Key from Hardware Wallet (Ledger)

For hardware wallet users, the easiest method is to import the public key (not private key) from the hardware wallet:

```bash
# Signer runs this on their machine
# Connect your Ledger device and open the Ethereum app

# Add the hardware wallet key (this only imports the public key, not the private key)
shardeumd keys add alice --ledger --keyring-backend test

# Export the public key in the format we need
shardeumd keys show alice --pubkey --keyring-backend test
```

**Note:** The `--ledger` flag only imports the public key from the hardware wallet. The private key never leaves the device.

### Option C: Convert Hex Public Key to JSON Format

If a signer has a hex public key (e.g., from a hardware wallet export or other tool), you can convert it using the helper script:

```bash
# From the shardeum-evm directory
bash scripts/convert_pubkey_to_json.sh <hex_pubkey>

# Example:
bash scripts/convert_pubkey_to_json.sh 0x04a1b2c3d4e5f6...
# or for compressed keys (33 bytes):
bash scripts/convert_pubkey_to_json.sh a1b2c3... true
```

**Important:** You **cannot** derive a public key from just a bech32 address. The address is a hash of the public key, so it's a one-way operation. You need the actual public key (hex or from hardware wallet).

### Create Signers JSON File

Create a file (e.g., `multisig-signers.json`) with all signer public keys:

```json
{
  "signers": [
    {
      "name": "alice",
      "pubkey": {
        "@type": "/ethermint.crypto.v1.ethsecp256k1.PubKey",
        "key": "Agq507WWHGJCk8mf1NVkDMt6XvRtVsfFiO79lHlsInpT"
      }
    },
    {
      "name": "bob",
      "pubkey": {
        "@type": "/ethermint.crypto.v1.ethsecp256k1.PubKey",
        "key": "AyCYDGhSOg0k7LvZ7X/qW5vdPiClbLEaAsFhh/54Tpu7"
      }
    },
    {
      "name": "carol",
      "pubkey": {
        "@type": "/ethermint.crypto.v1.ethsecp256k1.PubKey",
        "key": "AtIhR7qYTWKVywP8VJh9Ez2Ehl0VV9DdeDbkvuwHipar"
      }
    }
  ]
}
```

**Important Notes:**
- Each signer **must** have a unique `name`
- The `pubkey` must be a JSON object with `@type` and `key` fields
- The `@type` should be `/ethermint.crypto.v1.ethsecp256k1.PubKey` for Shardeum EVM
- The `key` is a base64-encoded public key

## Step 2: Set Environment Variables

```bash
# Set network configuration
export SHARDEUM_NETWORK=local  # or mainnet, testnet, devnet
export SHARDEUM_CONFIG_DIR=/path/to/shardeum-evm/config

# Set chain ID (optional, will be read from config if not set)
export SHARDEUM_CHAIN_ID=shardeum_8117-1

# Set transaction fee (must meet minimum global fee requirement)
# Check minimum fee: shardeumd query feemarket params --node tcp://localhost:26657
export FEE=1024065140194520500000ashm  # Adjust based on network requirements

# Set stake amount (MUST include denomination)
export STAKE_AMOUNT=1000000000000000000000ashm  # 1000 SHM

# Set chain ID for transactions
export CHAIN_ID=shardeum_8117-1  # Adjust based on your network
```

**Note for Testnet/Remote Networks:**
- The chain ID may differ from the config file. For example, testnet may use `shardeum-testnet` or `shardeum_8119-2` depending on the deployment. Always verify the actual chain ID from your genesis file or contact ops for the current network configuration.
- For fee estimation on remote networks, query the appropriate RPC endpoint (e.g., `shardeumd query feemarket params --node https://rpc-mezame.shardeum.org` for testnet).

## Step 3: Add Node with Multisig Operator Key

Run `add_node.sh` with the multisig options:

```bash
# From the shardeum-evm directory
bash scripts/add_node.sh node2 \
  --network local \
  --node-type validator \
  --multisig-keys-file ./multisig-signers.json \
  --multisig-threshold 2
```

**Parameters:**
- `node2`: Unique node identifier
- `--network local`: Network to join
- `--node-type validator`: Create a validator node
- `--multisig-keys-file`: Path to the signers JSON file created in Step 1
- `--multisig-threshold 2`: Require 2 out of 3 signatures (adjust based on your needs)

**What the script does:**
1. Initializes the node
2. Validates all signer public keys
3. Imports signer public keys as offline keys (no private keys)
4. Creates `validator-operator` multisig key
5. Generates `multisig_info.json` with all relevant information

**Expected Output:**
```
✅ Multisig operator key created successfully!
Multisig Address: shardeum1cr7ugf6hc6fjgy9qdzmle78jxv9lq6s0v9dnsh
Validator Operator Address: shardeumvaloper1cr7ugf6hc6fjgy9qdzmle78jxv9lq6s0tupnp0
Threshold: 2 of 3
Multisig info saved to: .local/node2/multisig_info.json

📝 Next steps:
1. Fund the multisig address: shardeum1cr7ugf6hc6fjgy9qdzmle78jxv9lq6s0v9dnsh
2. Create unsigned validator transaction
3. Share with signers for signatures (threshold: 2)
4. Merge signatures and broadcast
```

## Step 4: Fund the Multisig Address

The multisig address needs to be funded with enough tokens to cover:
- Stake amount
- Transaction fees

### Option A: Fund via Metamask/Keplr

1. Get the multisig address from `multisig_info.json`:
   ```bash
   cat .local/node2/multisig_info.json | jq -r '.multisig_address'
   ```

2. Send tokens to this address using Metamask or Keplr wallet

3. Verify balance:
   ```bash
   MULTISIG_ADDR=$(cat .local/node2/multisig_info.json | jq -r '.multisig_address')
   shardeumd query bank balances "$MULTISIG_ADDR" --node tcp://localhost:26657
   ```

### Option B: Fund from Another Account

If you have a funded account in your keyring:

```bash
# Load multisig address
MULTISIG_ADDR=$(cat .local/node2/multisig_info.json | jq -r '.multisig_address')

# Calculate required amount (stake + fee buffer)
STAKE_NUM=$(echo "$STAKE_AMOUNT" | sed 's/ashm$//')
FEE_BUFFER_NUM=$(echo "$FEE" | sed 's/ashm$//')
REQUIRED_NUM=$(echo "$STAKE_NUM + $FEE_BUFFER_NUM" | bc)
REQUIRED="${REQUIRED_NUM}ashm"

# Send funds
shardeumd tx bank send <funded-account> "$MULTISIG_ADDR" "$REQUIRED" \
  --chain-id "$CHAIN_ID" \
  --fees "$FEE" \
  --keyring-backend test \
  --home .local/node0 \
  -y

# Wait for confirmation
sleep 5

# Verify balance
shardeumd query bank balances "$MULTISIG_ADDR" --node tcp://localhost:26657
```

## Step 5: Get Account Number and Sequence

Before creating the unsigned transaction, you need the account number and sequence. Use the REST API (CLI has proto issues with multisig accounts):

```bash
# Load multisig address
MULTISIG_ADDR=$(cat .local/node2/multisig_info.json | jq -r '.multisig_address')

# Get account info via REST API
ACCOUNT_INFO=$(curl -s "http://localhost:1317/cosmos/auth/v1beta1/accounts/$MULTISIG_ADDR")

# Extract account number and sequence
ACCOUNT_NUMBER=$(echo "$ACCOUNT_INFO" | jq -r '.account.account_number // .account.value.account_number')
SEQUENCE=$(echo "$ACCOUNT_INFO" | jq -r '.account.sequence // .account.value.sequence // "0"')

echo "Account Number: $ACCOUNT_NUMBER"
echo "Sequence: $SEQUENCE"
```

## Step 6: Create Unsigned Validator Transaction

Create the validator configuration file:

```bash
# Load node directory and addresses
NODE_DIR=".local/node2"
MULTISIG_ADDR=$(cat "$NODE_DIR/multisig_info.json" | jq -r '.multisig_address')
VALIDATOR_OPERATOR_ADDR=$(cat "$NODE_DIR/multisig_info.json" | jq -r '.validator_operator_address')

# Get validator consensus public key
VALIDATOR_PUBKEY=$(shardeumd comet show-validator --home "$NODE_DIR")

# Create validator JSON
cat > "$NODE_DIR/validator-create.json" << EOF
{
  "pubkey": $VALIDATOR_PUBKEY,
  "amount": "$STAKE_AMOUNT",
  "moniker": "MultisigValidator",
  "identity": "",
  "website": "",
  "security": "",
  "details": "Validator operated by multisig (2-of-3)",
  "commission-rate": "0.10",
  "commission-max-rate": "0.20",
  "commission-max-change-rate": "0.01",
  "min-self-delegation": "1"
}
EOF
```

Generate the unsigned transaction:

```bash
shardeumd tx staking create-validator "$NODE_DIR/validator-create.json" \
  --from validator-operator \
  --chain-id "$CHAIN_ID" \
  --fees "$FEE" \
  --gas 500000 \
  --generate-only \
  --keyring-backend test \
  --home "$NODE_DIR" > "$NODE_DIR/create-validator-unsigned.json"

echo "✅ Unsigned transaction created: $NODE_DIR/create-validator-unsigned.json"
```

## Step 7: Prepare Signing Information for Signers

Before sharing with signers, extract all the information they'll need:

```bash
NODE_DIR=".local/node2"

# Get account number and sequence (if not already set from Step 5)
MULTISIG_ADDR=$(jq -r '.multisig_address' "$NODE_DIR/multisig_info.json")
ACCOUNT_INFO=$(curl -s "http://localhost:1317/cosmos/auth/v1beta1/accounts/$MULTISIG_ADDR")
ACCOUNT_NUMBER=$(echo "$ACCOUNT_INFO" | jq -r '.account.account_number // .account.value.account_number')
SEQUENCE=$(echo "$ACCOUNT_INFO" | jq -r '.account.sequence // .account.value.sequence // "0"')

# Get the multisig public key
MULTISIG_PUBKEY=$(shardeumd keys show validator-operator --pubkey --keyring-backend test --home "$NODE_DIR")

# Display all values
echo "=== Signing Information ==="
echo "Chain ID: $CHAIN_ID"
echo "Account Number: $ACCOUNT_NUMBER"
echo "Sequence: $SEQUENCE"
echo "Multisig Address: $MULTISIG_ADDR"
echo "Multisig Public Key: $MULTISIG_PUBKEY"
echo "Threshold: $(jq -r '.threshold' "$NODE_DIR/multisig_info.json") of $(jq -r '.total_signers' "$NODE_DIR/multisig_info.json")"
```

**Save these values** - you'll need to provide them to signers.

## Step 8: Share Transaction with Signers

Send the unsigned transaction file to each signer along with the signing information. 

**Important:** Replace all example values below with the actual values from Step 7.

Prepare a message with actual values:

**Files to share:**
- `create-validator-unsigned.json`
- `multisig_info.json` (optional, for reference)

**Message to send to signers (replace values with actual numbers from Step 7):**

```
Transaction Details:
- Chain ID: shardeum_8117-1
- Account Number: 50
- Sequence: 0
- Multisig Address: shardeum1cr7ugf6hc6fjgy9qdzmle78jxv9lq6s0v9dnsh
- Threshold: 2 of 3
- Files: create-validator-unsigned.json, multisig_info.json

Multisig Public Key (for import):
{"@type":"/cosmos.crypto.multisig.LegacyAminoPubKey","threshold":2,"public_keys":[...]}

Signing Steps (Option 1 — Using shardeumd CLI):

1. Create a temporary keyring directory:
   ```bash
   TEMP_KEYRING=$(mktemp -d)
   ```

2. Import your private key (from mnemonic):
   ```bash
   shardeumd keys add <your-name> --recover --keyring-backend test --keyring-dir "$TEMP_KEYRING"
   # Enter your mnemonic when prompted
   ```

3. Import multisig public key (use the value provided above):
   ```bash
   MULTISIG_PUBKEY='{"@type":"/cosmos.crypto.multisig.LegacyAminoPubKey","threshold":2,"public_keys":[...]}'
   shardeumd keys add validator-operator --pubkey "$MULTISIG_PUBKEY" --keyring-backend test --keyring-dir "$TEMP_KEYRING"
   ```

4. Sign the transaction (replace values with actual numbers from Transaction Details above):
   ```bash
   shardeumd tx sign create-validator-unsigned.json \
     --from <your-name> \
     --multisig validator-operator \
     --chain-id shardeum_8117-1 \
     --account-number 50 \
     --sequence 0 \
     --keyring-backend test \
     --keyring-dir "$TEMP_KEYRING" \
     --offline \
     --output-document <your-name>-sig.json
   ```

   **Note:**
   - The `--offline` flag is required when signing on a machine without a local node running (common in remote/testnet setups).
   - For remote signing workflows or production deployments, contact ops for specific signing procedures and security requirements.

5. Delete the temporary keyring:
   ```bash
   rm -rf "$TEMP_KEYRING"
   ```

6. Return `<your-name>-sig.json` to the operator.

Signing Steps (Option 2 — Using Standalone Script):
- Use `scripts/sign-transaction-standalone.js` or `scripts/sign-transaction-standalone.py`
- Provide your mnemonic, the unsigned transaction JSON, chain ID, account number, and sequence (from Transaction Details above)
- Return the generated signature JSON file to the operator
```

## Step 9: Collect Signatures

Each signer should return a signature file (e.g., `alice-sig.json`, `bob-sig.json`). You need at least the threshold number of signatures (2 in this example).

Place all signature files in a directory:

```bash
mkdir -p signatures
# Copy signature files from signers
# cp alice-sig.json signatures/
# cp bob-sig.json signatures/
```

## Step 10: Merge Signatures

Combine the signatures into a single signed transaction:

### For Local Network Setup:
```bash
NODE_DIR=".local/node2"

# Merge signatures (need at least threshold number)
shardeumd tx multisign "$NODE_DIR/create-validator-unsigned.json" \
  validator-operator \
  signatures/alice-sig.json \
  signatures/bob-sig.json \
  --chain-id "$CHAIN_ID" \
  --keyring-backend test \
  --home "$NODE_DIR" \
  --output-document "$NODE_DIR/create-validator-signed.json"

echo "✅ Merged signatures: $NODE_DIR/create-validator-signed.json"
```

### For Remote/Testnet Setup:
```bash
NODE_DIR=".local/node2"

# Get account number and sequence (from Step 5)
ACCOUNT_NUMBER=<from_step_5>
SEQUENCE=<from_step_5>

# Merge signatures with offline mode
shardeumd tx multisign "$NODE_DIR/create-validator-unsigned.json" \
  validator-operator \
  signatures/alice-sig.json \
  signatures/bob-sig.json \
  --chain-id "$CHAIN_ID" \
  --keyring-backend test \
  --home "$NODE_DIR" \
  --offline \
  --account-number $ACCOUNT_NUMBER \
  --sequence $SEQUENCE \
  --output-document "$NODE_DIR/create-validator-signed.json"

echo "✅ Merged signatures: $NODE_DIR/create-validator-signed.json"
```

**Note:**
- If you have more signatures than the threshold, you can include them all. The multisig will use the first `threshold` valid signatures.
- For remote/testnet setups, the `--offline`, `--account-number`, and `--sequence` flags are required when merging signatures on a machine without a local node running.
- For production or remote signing workflows, contact ops for guidance on the signing and merging process.

## Step 11: Broadcast Transaction

Broadcast the signed transaction:

### For Local Network:
```bash
NODE_DIR=".local/node2"

shardeumd tx broadcast "$NODE_DIR/create-validator-signed.json" \
  --chain-id "$CHAIN_ID" \
  --node tcp://localhost:26657

# Wait a moment for confirmation
sleep 5

# Verify validator was created
VALIDATOR_OPERATOR_ADDR=$(cat "$NODE_DIR/multisig_info.json" | jq -r '.validator_operator_address')
shardeumd query staking validator "$VALIDATOR_OPERATOR_ADDR" --node tcp://localhost:26657
```

### For Remote/Testnet:
```bash
NODE_DIR=".local/node2"

# Broadcast to remote RPC endpoint
shardeumd tx broadcast "$NODE_DIR/create-validator-signed.json" \
  --chain-id "$CHAIN_ID" \
  --node https://rpc-mezame.shardeum.org  # Replace with your testnet RPC endpoint

# Wait a moment for confirmation
sleep 5

# Verify validator was created
VALIDATOR_OPERATOR_ADDR=$(cat "$NODE_DIR/multisig_info.json" | jq -r '.validator_operator_address')
shardeumd query staking validator "$VALIDATOR_OPERATOR_ADDR" --node https://rpc-mezame.shardeum.org
```

**Expected Output:**
```
code: 0
codespace: ""
txhash: <transaction-hash>
...
```

**Note:** For remote/testnet setups, replace `tcp://localhost:26657` with the appropriate RPC endpoint for your network (e.g., `https://rpc-mezame.shardeum.org` for testnet).

## Step 12: Verify Validator Status

Check that your validator is active:

```bash
VALIDATOR_OPERATOR_ADDR=$(cat .local/node2/multisig_info.json | jq -r '.validator_operator_address')

# Query validator
shardeumd query staking validator "$VALIDATOR_OPERATOR_ADDR" --node tcp://localhost:26657

# Check validator in active set
shardeumd query staking validators --node tcp://localhost:26657 | grep -A 10 "$VALIDATOR_OPERATOR_ADDR"
```

## Troubleshooting

### Error: "Invalid public key format"
- Ensure each `pubkey` in the JSON file is a complete JSON object with `@type` and `key` fields
- Verify the `@type` is `/ethermint.crypto.v1.ethsecp256k1.PubKey`

### Error: "Threshold cannot be greater than number of signers"
- Ensure `--multisig-threshold` is less than or equal to the number of signers in the JSON file

### Error: "Signer at index X is missing 'name' field"
- Every signer must have a `name` field
- Signer names must be unique

### Error: "cannot marshal response account"
- Use the REST API workaround for account queries (see Step 5)
- This is a known issue with multisig accounts in the CLI

### Error: "insufficient fee"
- Check the minimum global fee: `shardeumd query feemarket params --node tcp://localhost:26657`
- Increase the `FEE` environment variable accordingly

### Error: "signature verification failed" during broadcast
- Ensure you collected at least `threshold` number of signatures
- Verify all signatures are from valid signers (names match the multisig)
- Check that account number and sequence match the unsigned transaction

## Next Steps

After your validator is created, you can perform validator operations (withdraw commission, edit validator, etc.) using the same multisig workflow:

1. Generate unsigned transaction
2. Share with signers
3. Collect signatures (threshold number)
4. Merge signatures
5. Broadcast

See `docs/MULTISIG_VALIDATOR.md` for detailed instructions on validator operations.

## Security Notes

- **Private keys never enter the validator node** - only public keys are imported
- **Signers keep their private keys secure** - on separate machines or hardware wallets
- **Multisig threshold enforces security** - requires multiple approvals for all operations
- **Validator node cannot sign transactions** - it only has public keys, not private keys

## Summary

This workflow provides:
- ✅ Automated multisig key creation during node setup
- ✅ Clear separation between consensus key (node) and operator key (multisig)
- ✅ Secure signing process (private keys stay with signers)
- ✅ Flexible funding options (Metamask/Keplr or CLI)
- ✅ Step-by-step guide for the entire process

The validator node is now ready and secured with a multisig operator key!

