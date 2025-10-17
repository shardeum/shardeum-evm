# Network Data Verification Tool

A comprehensive verification script for validating Shardeum network data against genesis values after a network restart.

## Features

- **Account Balance Verification**: Compare network balances against genesis values for all accounts
- **Supply Verification**: Check total supply with configurable inflation tolerance
- **Async Batch Processing**: Efficient concurrent RPC queries with configurable concurrency
- **Split Genesis Support**: Handles mainnet/testnet genesis files split across multiple JSON files
- **Detailed Reporting**: Clear output showing matches, mismatches, and failures
- **Flexible Configuration**: Command-line options for RPC endpoints, tolerances, and performance tuning

## Prerequisites

### Python Dependencies

```bash
pip install aiohttp bech32
```

Or install all dev-scripts dependencies:

```bash
cd dev-scripts
pip install -r requirements.txt
```

### Network Requirements

- Access to a running Shardeum node with JSON-RPC enabled (default: http://localhost:8545)
- Genesis files in the expected location (see Genesis Files below)

## Installation

The script is ready to use:

```bash
chmod +x verify-network.py
```

## Usage

### Basic Usage

```bash
# Verify local network against local genesis
python verify-network.py \
  --genesis shardeum-evm/config/environments/local-genesis.genesis.json \
  --rpc http://localhost:8547

# Verify testnet
python verify-network.py \
  --genesis shardeum-evm/config/environments/testnet-genesis.genesis.json \
  --rpc http://testnet-rpc:8545

# Verify mainnet with custom tolerance
python verify-network.py \
  --genesis shardeum-evm/config/environments/mainnet-genesis.genesis.json \
  --rpc http://mainnet-rpc:8545 \
  --tolerance 1.0
```

### Advanced Options

```bash
# High-performance verification with increased concurrency
python verify-network.py \
  --genesis config/local-genesis.genesis.json \
  --rpc http://localhost:8545 \
  --concurrency 50 \
  --timeout 30

# Verbose mode with detailed mismatch reporting
python verify-network.py \
  --genesis config/mainnet-genesis.genesis.json \
  --rpc http://mainnet-rpc:8545 \
  --verbose

# Custom inflation tolerance for supply check
python verify-network.py \
  --genesis config/testnet-genesis.genesis.json \
  --rpc http://testnet-rpc:8545 \
  --tolerance 2.0  # Allow 2% supply difference
```

## Command-Line Arguments

| Argument | Required | Default | Description |
|----------|----------|---------|-------------|
| `--genesis` | Yes | - | Path to genesis JSON file |
| `--rpc` | No | `http://localhost:8545` | RPC URL for network queries |
| `--tolerance` | No | `0.5` | Inflation tolerance percentage |
| `--concurrency` | No | `20` | Number of concurrent RPC requests |
| `--timeout` | No | `10` | RPC request timeout in seconds |
| `--verbose` | No | `false` | Enable verbose output |
