# Standalone Node Operation Guide

This guide covers how to operate Shardeum nodes independently using the `shardeumd` binary, both with our provided scripts and manual commands.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Using Shardeum Scripts](#using-shardeum-scripts)
3. [Manual Node Operation](#manual-node-operation)
4. [Network Connection Methods](#network-connection-methods)
5. [Validator Operations](#validator-operations)
6. [Troubleshooting](#troubleshooting)

## Prerequisites

### Binary Requirements
- `shardeumd` binary (build with `make build` or download release)
- Network configuration files in `configs/` directory
- Genesis file for target network

### Network Information
- Chain ID (e.g., `shardeum_8119-1` - ethermint format for EVM compatibility)
- Seed node addresses for peer discovery
- RPC endpoints for existing network

## Using Shardeum Scripts

### Script-Based Node Setup

Our scripts provide automated node setup with proper configuration:

#### 1. Adding Validator Infrastructure

```bash
# Add validator infrastructure to local network
./scripts/add_node.sh node5 --node-type validator --network local

# Add to testnet with custom seed
./scripts/add_node.sh node6 --node-type validator --network testnet --seed-rpc http://testnet-seed:26657

# Add full-node (non-validator)
./scripts/add_node.sh node7 --node-type full-node --network local
```

#### 2. Creating Active Validators

```bash
# Step 1: Create validator account key
./build/shardeumd keys add validator-node5 --keyring-backend test --home .testnet/node5

# Step 2: Fund the account (example with dev account)
./build/shardeumd tx bank send dev0 [validator_address] 3000000000000000000ashm \
  --keyring-backend test --chain-id shardeum_8119-1 --node http://localhost:26657 \
  --from dev0 --yes

# Step 3: Create validator
./scripts/create_validator.sh node5 --validator-key validator-node5 --amount 2000000000000000000
```

### Script Benefits
- Automatic port assignment
- Proper peer discovery configuration
- Network-specific genesis download
- Standard directory structure
- Integrated keyring management

## Manual Node Operation

### Manual Node Setup

For advanced users or custom deployments, you can operate nodes manually:

#### 1. Initialize Node

```bash
# Set variables
NODE_ID="my-node"
CHAIN_ID="shardeum_8119-1"
HOME_DIR="$HOME/.shardeum-$NODE_ID"

# Initialize node
./shardeumd init $NODE_ID --chain-id $CHAIN_ID --home $HOME_DIR
```

#### 2. Configure Network

```bash
# Download genesis (if joining existing network)
curl -s http://seed-node:26657/genesis | jq -r '.result.genesis' > $HOME_DIR/config/genesis.json

# Or copy local genesis
cp config/local-genesis.json $HOME_DIR/config/genesis.json
```

#### 3. Configure Peers

Edit `$HOME_DIR/config/config.toml`:

```toml
# Set seed nodes
seeds = "node_id@ip:port,node_id2@ip:port"

# Set persistent peers (optional)
persistent_peers = "node_id@ip:port,node_id2@ip:port"

# Allow duplicate IPs for local testing
allow_duplicate_ip = true
```

#### 4. Configure Ports

Edit `$HOME_DIR/config/config.toml` and `$HOME_DIR/config/app.toml`:

```toml
# config.toml
[rpc]
laddr = "tcp://127.0.0.1:26657"

[p2p]
laddr = "tcp://0.0.0.0:26656"

# app.toml
[grpc]
address = "0.0.0.0:9090"

[api]
address = "tcp://0.0.0.0:1317"

[json-rpc]
address = "127.0.0.1:8545"
ws-address = "127.0.0.1:8546"
```

#### 5. Start Node

```bash
# Start validator node
./shardeumd start \
  --home $HOME_DIR \
  --chain-id $CHAIN_ID \
  --minimum-gas-prices="0.000006ashm" \
  --json-rpc.enable \
  --json-rpc.api eth,txpool,personal,net,debug,web3 \
  --pruning nothing

# Start full-node (non-validator)
./shardeumd start \
  --home $HOME_DIR \
  --chain-id $CHAIN_ID \
  --minimum-gas-prices="0.000006ashm" \
  --json-rpc.enable \
  --json-rpc.api eth,txpool,personal,net,debug,web3 \
  --non-validator \
  --pruning nothing
```

## Network Connection Methods

### Seed-Based Discovery (Recommended)

Configure seed nodes in `config.toml`:

```toml
seeds = "seed_node_id@seed_ip:26656"
```

Benefits:
- Automatic peer discovery
- No need to know all network peers
- Resilient to network changes

### Static Peer Configuration

Configure persistent peers in `config.toml`:

```toml
persistent_peers = "peer1_id@ip1:26656,peer2_id@ip2:26656"
```

Use when:
- Private networks with known peers
- Need guaranteed connections to specific nodes

### Getting Peer Information

```bash
# Get node ID
./shardeumd comet show-node-id --home $HOME_DIR

# Get node status and peer info
curl -s http://localhost:26657/status
curl -s http://localhost:26657/net_info
```

## Validator Operations

### Creating Validator Account

```bash
# Create new account
./shardeumd keys add my-validator --keyring-backend test --home $HOME_DIR

# Import existing account
./shardeumd keys add my-validator --recover --keyring-backend test --home $HOME_DIR
```

### Funding Validator Account

```bash
# Check balance
./shardeumd query bank balances [validator_address] --node tcp://localhost:26657

# Send funds (from existing funded account)
./shardeumd tx bank send [sender] [validator_address] [amount]ashm \
  --keyring-backend test \
  --chain-id $CHAIN_ID \
  --node tcp://localhost:26657 \
  --from [sender] \
  --yes
```

### Creating Validator

```bash
# Get consensus public key
VALIDATOR_PUBKEY=$(./shardeumd comet show-validator --home $HOME_DIR)

# Create validator transaction
./shardeumd tx staking create-validator \
  --amount="1000000000000000000ashm" \
  --pubkey="$VALIDATOR_PUBKEY" \
  --moniker="My Validator" \
  --commission-rate="0.10" \
  --commission-max-rate="0.20" \
  --commission-max-change-rate="0.01" \
  --min-self-delegation="1" \
  --from=my-validator \
  --keyring-backend test \
  --home $HOME_DIR \
  --chain-id=$CHAIN_ID \
  --node=tcp://localhost:26657 \
  --gas=500000 \
  --fees=400000000000000000000ashm \
  --yes
```

### Validator Management

```bash
# Check validator status
./shardeumd query staking validator [validator_operator_address] --node tcp://localhost:26657

# List all validators
./shardeumd query staking validators --node tcp://localhost:26657

# Delegate to validator
./shardeumd tx staking delegate [validator_operator_address] [amount]ashm \
  --from [delegator] --keyring-backend test --chain-id $CHAIN_ID --node tcp://localhost:26657

# Unjail validator (if jailed)
./shardeumd tx slashing unjail --from my-validator --keyring-backend test \
  --chain-id $CHAIN_ID --node tcp://localhost:26657
```

## Troubleshooting

### Common Issues

#### Node Won't Start

```bash
# Check logs
tail -f $HOME_DIR/logs/node.log

# Verify genesis file
./shardeumd validate-genesis --home $HOME_DIR

# Check port conflicts
lsof -i :26657
```

#### Peer Discovery Issues

```bash
# Check peer connections
curl -s http://localhost:26657/net_info | jq '.result.peers'

# Test seed connectivity
nc -zv [seed_ip] 26656

# Reset addrbook
rm $HOME_DIR/config/addrbook.json
```

#### Validator Issues

```bash
# Check if validator is in active set
./shardeumd query staking validators --node tcp://localhost:26657 | grep -A5 -B5 [moniker]

# Check validator signing status
./shardeumd query slashing signing-info [validator_consensus_address] --node tcp://localhost:26657

# Check if validator is jailed
./shardeumd query staking validator [validator_operator_address] --node tcp://localhost:26657 | grep jailed
```

### Performance Optimization

#### Pruning Configuration

```toml
# app.toml
pruning = "custom"
pruning-keep-recent = "100"
pruning-interval = "10"
```

#### State Sync (For Fast Sync)

```toml
# config.toml
[statesync]
enable = true
rpc_servers = "node1:26657,node2:26657"
trust_height = [recent_height]
trust_hash = "[block_hash]"
```

### Monitoring

```bash
# Node status
curl -s http://localhost:26657/status

# Consensus state
curl -s http://localhost:26657/consensus_state

# Block info
curl -s http://localhost:26657/block

# Network info
curl -s http://localhost:26657/net_info
```

## Advanced Configuration

### Custom Network Configuration

Create custom network config in `configs/custom.json`:

```json
{
  "name": "custom",
  "chain_id": "my-custom-chain",
  "evm_chain_id": 9999,
  "base_denom": "acustom",
  "display_denom": "custom",
  "decimals": 18,
  "bech32_prefix": "custom",
  "ports": {
    "rpc": "26657",
    "rest": "1317",
    "json_rpc": "8545",
    "websocket": "8546",
    "grpc": "9090"
  },
  "genesis_file": "custom-genesis.json"
}
```

### Environment Variables

```bash
# Set network configuration directory
export SHARDEUM_CONFIG_DIR="/path/to/configs"

# Override network parameters
export SHARDEUM_NETWORK=testnet
export SHARDEUM_CHAIN_ID=my-custom-chain

# Use with scripts
./scripts/add_node.sh node5 --network custom
```

This guide provides both automated script-based and manual approaches for operating Shardeum nodes, giving operators flexibility based on their requirements and expertise level.