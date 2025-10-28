# Genesis Balance Transfer Script

Transfer balances between accounts in Shardeum genesis files.

## Input Files

The script reads and modifies genesis files in `config/environments/`:

| Network | Main Genesis                 | Split Account Files                          |
| ------- | ---------------------------- | -------------------------------------------- |
| mainnet | mainnet-genesis.genesis.json | mainnet-genesis.genesis.accounts.{1..8}.json |
| testnet | testnet-genesis.genesis.json | testnet-genesis.genesis.accounts.{1..8}.json |
| local   | local-genesis.genesis.json   | local-genesis.genesis.accounts.{1..2}.json   |

**Note:** If split files exist, the script uses them. Otherwise, it reads/writes to the main genesis file only.

## Usage

```bash
python3 scripts/transfer_genesis_balance.py <network> <source> <dest> <amount>
```

**Arguments:**

- `network` - Genesis network: `mainnet`, `testnet`, `local`
- `source` - Source address (bech32 or hex)
- `dest` - Destination address (bech32 or hex)
- `amount` - Amount in ashm (1 SHM = 10^18 ashm)

## Examples

### Basic Transfer

```bash
python3 scripts/transfer_genesis_balance.py local \
  shardeum1ak8cc38yx23f8fla0skvk4dvfat7a9hyqf3zwv \
  shardeum1zszzmpde0zel9xyehh65ryntur2s2usxhhfzvj \
  1000000000000000000000
```

### Using Hex Addresses

```bash
python3 scripts/transfer_genesis_balance.py testnet \
  0x1234567890abcdef1234567890abcdef12345678 \
  0xabcdefabcdefabcdefabcdefabcdefabcdef1234 \
  5000000000000000000000
```

### Mixed Format

```bash
python3 scripts/transfer_genesis_balance.py mainnet \
  shardeum1source1234567890123456789012345678901234 \
  0xabcdefabcdefabcdefabcdefabcdefabcdef1234 \
  2000000000000000000000
```

## Amount Conversion

| SHM    | Ashm                    |
| ------ | ----------------------- |
| 0.001  | 1000000000000000        |
| 1      | 1000000000000000000     |
| 100    | 100000000000000000000   |
| 1,000  | 1000000000000000000000  |
| 10,000 | 10000000000000000000000 |

**Quick conversion:**

```bash
# SHM to ashm
python3 -c "print(int(1000 * 10**18))"

# ashm to SHM
python3 -c "print(1000000000000000000000 / 10**18)"
```

## Features

- ✅ Minimal changes - only modifies affected accounts/files
- ✅ Supports both bech32 (`shardeum1...`) and hex (`0x...`) addresses
- ✅ Works with split (mainnet/testnet) and unified (local) genesis files
- ✅ Auto-creates destination account if it doesn't exist
- ✅ Validates source balance before transfer

## What Gets Changed

### New Destination Account

- Source file: 1 balance updated (decreased)
- Last file: 1 account + 1 balance added (new account)

### Existing Destination Account

- Source file: 1 balance updated (decreased)
- Dest file: 1 balance updated (increased)

### NOT Changed

- `evm.accounts` (only for smart contracts)
- `bank.supply` (stays constant)
- Account order (preserved)
- Unrelated files (skipped)

## Verification

### Check Balance

```bash
jq '.app_state.bank.balances[] | select(.address=="<address>")' \
  config/environments/<network>-genesis.genesis.json
```

### List All Accounts

```bash
jq '.app_state.auth.accounts[].address' \
  config/environments/<network>-genesis.genesis.json
```

## Common Use Cases

### Fund a Validator

```bash
python3 scripts/transfer_genesis_balance.py mainnet \
  <treasury_account> \
  <validator_account> \
  500000000000000000000000
```

### Create Multiple Test Accounts

```bash
SOURCE="shardeum1treasury..."
for i in {1..5}; do
  python3 scripts/transfer_genesis_balance.py local \
    $SOURCE \
    shardeum1test$(printf "%03d" $i)... \
    1000000000000000000000
done
```

## Troubleshooting

| Error                    | Solution                                 |
| ------------------------ | ---------------------------------------- |
| Source account not found | Verify address exists in genesis         |
| Insufficient balance     | Check balance with jq command            |
| Invalid address format   | Use bech32 (shardeum1...) or hex (0x...) |
| Genesis file not found   | Check network name                       |

## Technical Details

### Account Structure

New accounts are created with:

```json
{
  "@type": "/cosmos.auth.v1beta1.BaseAccount",
  "address": "shardeum1...",
  "pub_key": null,
  "sequence": "0"
}
```

- `sequence: "0"` means the account has sent 0 transactions (brand new)
- This is the transaction counter (like Ethereum's nonce)
- Increments with each transaction to prevent replay attacks

### Balance Structure

```json
{
  "address": "shardeum1...",
  "coins": [{ "denom": "ashm", "amount": "1000000000000000000000" }]
}
```

## Requirements

- Python 3.6+
- `jq` (optional, for verification)
- Read/write access to genesis files
