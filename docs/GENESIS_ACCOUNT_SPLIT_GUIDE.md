# Genesis Account Splitter

This feature allows splitting large genesis files into smaller chunks to avoid issues with merge tools and file comparisons, as well as merging them back together.

## File Structure

When split, genesis files follow this pattern:
```
<network>.genesis.json                    # Main genesis (without accounts)
<network>.genesis.accounts.1.json         # First batch of accounts
<network>.genesis.accounts.2.json         # Second batch of accounts
<network>.genesis.accounts.<n>.json       # Nth batch of accounts
```

For example:
```
mainnet.genesis.json                      # Main structure
mainnet.genesis.accounts.1.json           # Accounts 1-5000
mainnet.genesis.accounts.2.json           # Accounts 5001-10000
...
```

## Usage

The `genesis_account_split.sh` script provides a unified interface for all genesis account operations.

### Splitting Genesis Files

Split a genesis file into smaller chunks:

```bash
# Split by network name
./scripts/genesis_account_split.sh split --network mainnet --accounts 10000

# Split a specific file
./scripts/genesis_account_split.sh split --file ./custom-genesis.json --accounts 5000

# Specify output directory
./scripts/genesis_account_split.sh split --network testnet --output-dir ./split-genesis
```

### Merging Split Files

Merge split account files back into a genesis:

```bash
# Merge and save to new file
./scripts/genesis_account_split.sh merge --file ./mainnet.genesis.json --output ./complete-genesis.json

# Merge from specific directory
./scripts/genesis_account_split.sh merge --file ./genesis.json --accounts-dir ./split-files
```

### Loading Genesis (Auto-merge)

Load a genesis file with automatic account merging if split files exist:

```bash
# Load and output to file
./scripts/genesis_account_split.sh load --file ./mainnet.genesis.json --output ./merged.json

# Load and output to stdout (for piping)
./scripts/genesis_account_split.sh load --file ./mainnet.genesis.json
```

### Checking for Split Files

Check if a genesis has split account files:

```bash
# Check by network
./scripts/genesis_account_split.sh check --network mainnet

# Check specific file
./scripts/genesis_account_split.sh check --file ./genesis.json
```

### Counting Accounts

Count total accounts (including those in split files):

```bash
# Count by network
./scripts/genesis_account_split.sh count --network mainnet

# Count in specific file
./scripts/genesis_account_split.sh count --file ./genesis.json
```

## Command Options

**Split Options:**
- `--network <name>`: Network to split (mainnet, testnet, devnet, local)
- `--file <path>`: Specific genesis file to split
- `--accounts <num>`: Number of accounts per file (default: 5000)
- `--output-dir <path>`: Output directory (default: same as genesis file)

**Merge Options:**
- `--file <path>`: Main genesis file to merge accounts into
- `--accounts-dir <path>`: Directory containing account files
- `--output <path>`: Output path for merged genesis

**Load Options:**
- `--file <path>`: Genesis file to load
- `--output <path>`: Output path for complete genesis (optional)

**Check/Count Options:**
- `--network <name>`: Network to check
- `--file <path>`: Genesis file to check

## Automatic Detection

The network startup scripts automatically detect and merge split account files:

```bash
# Start network - will auto-detect split files
make start-network NETWORK=mainnet

# Or using scripts directly
./scripts/start_network.sh 4 --network mainnet
```

No changes needed - the scripts automatically:
1. Detect if split account files exist
2. Merge them when loading genesis
3. Use the complete genesis for node initialization

## Configuration

Default settings:
- **Accounts per file**: 5000 (configurable)
- **File naming**: `<network>.genesis.accounts.<index>.json`

## Example Workflow

1. Split default mainnet genesis (49,544 accounts):
```bash
./scripts/genesis_account_split.sh split --network mainnet --accounts 10000
```

Creates:
- `mainnet.genesis.json` (structure only)
- `mainnet.genesis.accounts.1.json` (10,000 accounts)
- `mainnet.genesis.accounts.2.json` (10,000 accounts)
- `mainnet.genesis.accounts.3.json` (10,000 accounts)
- `mainnet.genesis.accounts.4.json` (10,000 accounts)
- `mainnet.genesis.accounts.5.json` (9,544 accounts)

2. Start network (auto-merges):
```bash
make start-network NETWORK=mainnet
```

3. Network starts normally with all 49,544 accounts loaded
