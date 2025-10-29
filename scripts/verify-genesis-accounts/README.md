# Network Verification Script

## Overview

`verify-network.py` is a tool for verifying Shardeum accounts across three data sources:

1. **Database (SQLite)** - The original Shardeum accounts.sqlite3 database
2. **Genesis Files** - Generated genesis.json (monolithic or split format)
3. **Live Network** - On-chain state via REST API

This script ensures data consistency across the entire pipeline: from snapshot database → genesis generation → network deployment.

## Features

✅ Multi-source verification (DB, Genesis, Network)
✅ Account type filtering (EOA, Contracts, or both)
✅ Balance and nonce/sequence verification
✅ Unclaimed rewards consideration
✅ Support for split genesis files
✅ Concurrent network queries for performance
✅ Detailed mismatch reporting
✅ Flexible configuration (enable/disable any source)

## Installation

```bash
# Install required Python packages
pip install requirements.txt

# Make script executable
chmod +x verify-network.py
```

## Basic Usage

**Note**: All sources (DB, Genesis, Network) are optional. You can verify any combination.

### Genesis vs Network Only (Most Common)

```bash
./verify-network.py \
  --genesis ../../shardeum-evm/config/environments/mainnet-genesis.genesis.json \
  --rest http://localhost:1317
```

### Database vs Genesis Only

```bash
./verify-network.py \
  --db accounts.sqlite3 \
  --genesis genesis.json \
  --balance-multiplier 240
```

### All Three Sources

```bash
./verify-network.py \
  --db ../../helper/latest-accounts.sqlite3 \
  --genesis ../../shardeum-evm/config/environments/mainnet-genesis.genesis.json \
  --rest http://localhost:1317 \
  --balance-multiplier 240
```

### Include Contract Accounts (Type 1)

```bash
./verify-network.py \
  --db accounts.sqlite3 \
  --genesis genesis.json \
  --rest http://localhost:1317 \
  --account-types 0 1
```

### With Nonce Verification

```bash
./verify-network.py \
  --db accounts.sqlite3 \
  --genesis genesis.json \
  --rest http://localhost:1317 \
  --check-nonce
```

### With Unclaimed Rewards File

```bash
./verify-network.py \
  --db accounts.sqlite3 \
  --genesis genesis.json \
  --rest http://localhost:1317 \
  --unclaimed-rewards ../../dev-scripts/staking-data-extraction/genesis-addresses.json
```

### With Balance Multiplier

If genesis was generated with a balance multiplier (e.g., 240x), specify it for accurate comparison:

```bash
./verify-network.py \
  --db accounts.sqlite3 \
  --genesis genesis.json \
  --rest http://localhost:1317 \
  --balance-multiplier 240
```

**Important**: The script applies the multiplier to DB balances during extraction, then compares against genesis/network balances which already have the multiplier baked in.

### Save Output to File

To save the verification report to a file instead of printing to terminal:

```bash
./verify-network.py \
  --genesis genesis.json \
  --rest http://localhost:1317 \
  --output verification-report.txt

# Or use the short form
./verify-network.py \
  --genesis genesis.json \
  --rest http://localhost:1317 \
  -o verification-report.txt
```

**Benefits of using `--output`**:
- Progress messages still appear on screen (sent to stderr)
- Full report with **all accounts** (no trimming) is saved to the file
- Unlike terminal output which limits to 5-10 items per section, file output shows everything
- Useful for detailed analysis and record-keeping

## Command-Line Arguments

### Sources (at least one must be provided)

- `--db <path>` - Path to accounts.sqlite3 database (optional)
- `--genesis <path>` - Path to genesis.json file (optional)
- `--rest <url>` - REST API URL, e.g., http://localhost:1317 (optional)

**Note**: You must provide at least one source. The script will only compare sources that are provided.

### Optional

- `--unclaimed-rewards <path>` - Path to genesis-addresses.json with unclaimed rewards
- `--manual-transfers <path>` - Path to manual transfers JSON file
- `--output <path>` / `-o <path>` - Save verification report to file (default: print to stdout)
- `--account-types <types...>` - Account types to check (default: `0`)
  - `0` = EOA (Externally Owned Account)
  - `1` = Contract Account
  - `0 1` = Both types
- `--check-nonce` - Also verify nonces/sequences (default: false)
- `--balance-multiplier <n>` - Balance multiplier for verification (default: 1)
- `--concurrency <n>` - Concurrent network requests (default: 20)
- `--timeout <n>` - Network request timeout in seconds (default: 10)
- `--verbose` / `-v` - Verbose output with detailed error messages and all accounts (no trimming)
- `--check-db <true|false>` - Enable/disable database checking (default: true)
- `--check-genesis <true|false>` - Enable/disable genesis checking (default: true)
- `--check-network <true|false>` - Enable/disable network checking (default: true)

## Account Types

The script supports filtering by account type from the database:

- **Type 0**: EOA (Externally Owned Accounts) - Regular user accounts
- **Type 1**: Contract Accounts - Smart contracts deployed on the network

By default, only EOA accounts (type 0) are checked. Use `--account-types 0 1` to include both.

## Output Report

The script generates a clear, focused report with these sections:

### 1. Configuration
- Which sources are being verified (Database, Genesis, Network)
- Balance multiplier (if applied)
- Nonce verification status

### 2. Account Counts by Source
- Number of accounts in each source

### 3. Total Supply by Source
- Total supply in SHM for each source

### 4. Missing Accounts
Shows accounts that exist in one source but not another:
- Only displays relevant comparisons based on sources provided
- Clearly labeled (e.g., "In Genesis but NOT on Network")
- Includes explanation for expected cases (e.g., accounts created after genesis)
- **Terminal output**: Shows first 10 accounts, then "... and X more"
- **File output** (`--output` flag): Shows **all** accounts (no trimming)
- **Verbose mode** (`--verbose` flag): Shows **all** accounts (no trimming)

### 5. Balance Mismatches (if any)
- Grouped by comparison type (db_vs_genesis, genesis_vs_network, etc.)
- Includes unclaimed rewards information when applicable
- **Terminal output**: Shows first 5 examples with full details
- **File output** (`--output` flag): Shows **all** mismatches (no trimming)
- **Verbose mode** (`--verbose` flag): Shows **all** mismatches (no trimming)

### 6. Nonce Mismatches (if enabled)
- **Terminal output**: Shows first 5 examples with details
- **File output** (`--output` flag): Shows **all** mismatches (no trimming)
- **Verbose mode** (`--verbose` flag): Shows **all** mismatches (no trimming)

### 7. Verification Summary
Clear summary showing:
- **Accounts matched** - Balance and nonce match perfectly
- **Missing accounts** - Exist in one source but not another
- **Balance mismatches** - Exist in both but balances differ
- **Nonce mismatches** - (if checked) Nonces differ

Example:
```
Database ↔ Genesis:
  - Accounts matched: 71,936
  - Missing accounts: 0
  - Balance mismatches: 8
  - Nonce mismatches: 0

Genesis ↔ Network:
  - Accounts matched: 20,550
  - Missing accounts: 9
  - Balance mismatches: 51,394
```

### 8. Overall Status
- ✅ Success if no critical issues
- ⚠️  Warning with explanation if mismatches found
- Note explaining that Genesis↔Network mismatches are normal due to network activity

## How It Works

### 1. Database Extraction

Queries the SQLite database for accounts matching the specified types:

```sql
SELECT accountId, accountType, balance, nonce
FROM accounts
WHERE accountType IN (0, 1, ...)
```

Converts Ethereum addresses to Cosmos bech32 format using the same algorithm as `verify_genesis_accounts.py`.

### 2. Genesis Loading

Supports both monolithic and split genesis formats:

- **Monolithic**: Reads from `app_state.auth.accounts` and `app_state.bank.balances`
- **Split**: Automatically detects and loads `*.genesis.accounts.*.json` files

### 3. Network Querying

Uses REST API endpoints to fetch on-chain data:

1. Fetches all addresses: `GET /cosmos/auth/v1beta1/accounts`
2. For each address:
   - Account info: `GET /cosmos/auth/v1beta1/accounts/{address}`
   - Balance info: `GET /cosmos/bank/v1beta1/balances/{address}`

Uses concurrent requests (default: 20) for performance.

### 4. Unclaimed Rewards

If provided, loads the `genesis-addresses.json` file to track which accounts received additional balance from unclaimed staking rewards. This helps explain balance differences between sources.

### 5. Comparison & Analysis

Compares data across all enabled sources and identifies:
- Accounts present in some sources but not others
- Balance mismatches (with differences calculated)
- Nonce mismatches (if enabled)
- Supply differences

## Example Scenarios

### Scenario 1: Post-Genesis Network Launch

Verify that all genesis accounts made it onto the network:

```bash
./verify-network.py \
  --genesis mainnet-genesis.genesis.json \
  --rest https://sphinx.shardeum.org \
  --check-db false
```

### Scenario 2: Genesis Generation Validation

Verify genesis was correctly generated from database:

```bash
./verify-network.py \
  --db accounts.sqlite3 \
  --genesis genesis.json \
  --check-network false \
  --check-nonce
```

### Scenario 3: Full Pipeline Verification

Verify entire pipeline with contracts and nonces:

```bash
./verify-network.py \
  --db accounts.sqlite3 \
  --genesis genesis.json \
  --rest http://localhost:1317 \
  --unclaimed-rewards genesis-addresses.json \
  --account-types 0 1 \
  --check-nonce \
  --verbose
```

### Scenario 4: Network Snapshot Comparison

Compare live network against original snapshot database:

```bash
./verify-network.py \
  --db snapshot-accounts.sqlite3 \
  --rest https://mainnet-api.shardeum.org \
  --check-genesis false \
  --account-types 0 1
```

## Performance Considerations

- **Concurrency**: Adjust `--concurrency` based on your network and API rate limits
  - Too high: May overwhelm the REST API
  - Too low: Verification takes longer
  - Recommended: 10-50 for local node, 5-20 for remote API

- **Timeout**: Increase `--timeout` for slow networks or large account queries

- **Account Types**: Checking only EOAs (type 0) is faster than including contracts

- **Network Checking**: Disable with `--check-network false` if you only need DB/Genesis comparison

## Exit Codes

- `0` - All checks passed, no issues found
- `1` - Issues found (mismatches or missing accounts)
- `1` - Error occurred during verification

## Troubleshooting

### "Database file not found"
Ensure the path to accounts.sqlite3 is correct and the file exists.

### "Genesis file not found"
Check the genesis file path. For split genesis, ensure the main file and `*.genesis.accounts.*.json` files are in the same directory.

### "Network connection failed"
- Verify the REST API URL is correct
- Check if the node is running and accessible
- Try increasing `--timeout`
- Reduce `--concurrency`

### "Balance mismatches found"
This may be expected if:
- **Balance multiplier not specified**: If genesis was created with `--balance-multiplier 240`, you MUST use `--balance-multiplier 240` when verifying
- Unclaimed rewards were added (provide `--unclaimed-rewards` file)
- Network has inflation enabled (balances increase over time)

**Example**: If DB shows 130 SHM and genesis/network show 31,200 SHM, that's a 240x multiplier. Use `--balance-multiplier 240`.

### "Missing accounts in network"
This could indicate:
- Genesis file wasn't fully applied during network initialization
- Accounts with zero balance weren't created (expected behavior in some cases)
- Network is still initializing

## License

Same as the Shardeum project.
