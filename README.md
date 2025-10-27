# Shardeum EVM Blockchain

A high-performance EVM-compatible blockchain that provides seamless Ethereum compatibility with enhanced scalability and interoperability features.

## Getting started


### Multi-Node Testnet

For testing with multiple validators, you can use either the Makefile target or the script directly. You can also specify which network to use:

```bash
# Using Makefile (recommended)
make start-network                 # Start 4 nodes on local network (default)
make start-network NETWORK=testnet # Start 4 nodes on testnet

# Or directly using the script
./scripts/start_network.sh [number_of_nodes] [--network network_name]
```

This script will:
- Build the `shardeumd` binary 
- Initialize nodes with the specified network configuration
- Set up a bootstrap validator on node0
- Configure seed-based peer discovery (node0 acts as seed)
- Start all nodes with proper port assignments

Examples:
```bash
# Using Makefile
make start-network                          # Start 4 nodes on local network (default)
make start-network NODES=6                  # Start 6 nodes on local network
make start-network NETWORK=testnet          # Start 4 nodes on testnet
make start-network NODES=6 NETWORK=devnet   # Start 6 nodes on devnet

# Using script directly 
./scripts/start_network.sh                  # Start 4 nodes on local network (default)
./scripts/start_network.sh 6                # Start 6 nodes on local network
./scripts/start_network.sh 4 --network testnet  # Start 4 nodes on testnet
./scripts/start_network.sh 6 --network devnet   # Start 6 nodes on devnet
```

#### Adding Nodes to Running Network

You can dynamically add more nodes to an existing network with different node types:

```bash
# Using Makefile
make add-node NODE_ID=node4 [NODE_TYPE=validator] [SEED_RPC=http://localhost:26657] [NETWORK=local]

# Or directly using the script
./scripts/add_node.sh <node_id> [--node-type type] [--seed-rpc seed_endpoint] [--network network_name]
```

##### Node Types

- **validator**: Sets up validator infrastructure (default) - can be promoted to active validator
- **full-node**: Sets up a non-validator full node that syncs the blockchain

Examples:
```bash
# Using Makefile
make add-node NODE_ID=node4                                    # Add validator infrastructure to local network
make add-node NODE_ID=node5 NODE_TYPE=full-node NETWORK=testnet # Add full-node to testnet
make add-node NODE_ID=node6 SEED_RPC=http://localhost:26658    # Add validator infrastructure with specific seed

# Using script directly
./scripts/add_node.sh node4                                    # Add validator infrastructure to local network
./scripts/add_node.sh node5 --node-type full-node --network testnet  # Add full-node to testnet
./scripts/add_node.sh node6 --node-type validator --seed-rpc http://localhost:26658  # Add validator infrastructure with specific seed
./scripts/add_node.sh 6                                        # Add node6 validator infrastructure to local network
```

The script will:
- Initialize the new node with proper configuration
- Download genesis from existing network
- Configure seed-based peer discovery
- Assign available ports automatically
- Start the node and connect to the network

##### Creating Active Validators

After setting up validator infrastructure, you need to create and fund a validator account, then register it:

**Step 1: Create Validator Account**
```bash
# Create validator key (save the mnemonic!)
./build/shardeumd keys add validator-node5 --keyring-backend test --home .testnet/node5
```

**Step 2: Fund the Validator Account**

Option A - Using CLI with existing funded account:
```bash
./build/shardeumd tx bank send dev0 [validator_address] 3000000000000000000ashm \
  --keyring-backend test --chain-id shardeum_8119-1 --node http://localhost:26657 \
  --from dev0 --yes
```

Option B - Using Keplr Wallet:
1. Add the Shardeum network to Keplr (see Keplr Integration section)
2. Send SHM tokens to the validator address via Keplr interface

**Step 3: Create the Validator**
```bash
# Using Makefile
make create-validator NODE_ID=node5 VALIDATOR_KEY=my-validator AMOUNT=2000000000000000000

# Using script directly
./scripts/create_validator.sh node5 --validator-key my-validator --amount 2000000000000000000
```

**Validator Requirements:**
- Minimum stake: 1 SHM (1000000000000000000ashm)
- Recommended: 3+ SHM for fees and operations
- Account must be funded before validator creation

**Check Validator Status:**
```bash
./build/shardeumd query staking validators --node tcp://localhost:26657
```

#### Stopping Nodes

To stop all nodes:
```bash
pkill -f 'shardeumd.*shardeum_8119-1'
# Or press Ctrl+C if running in foreground
```

To stop a specific added node:
```bash
# Find the process ID and kill it, or use Ctrl+C in the add_node terminal
```

### Network Endpoints

#### Single Node
- **RPC**: http://localhost:26657
- **API**: http://localhost:1317  
- **JSON-RPC**: http://localhost:8545

#### Multi-Node
Each node runs on different ports:
- **Node 0**: RPC 26657, GRPC 9090, API 1317, JSON-RPC 8545, WebSocket 8546
- **Node 1**: RPC 26658, GRPC 9091, API 1318, JSON-RPC 8547, WebSocket 8548
- **Node N**: RPC 26657+N, GRPC 9090+N, API 1317+N, JSON-RPC 8545+N*2, WebSocket 8546+N*2

## Keplr Wallet Integration

### Adding Shardeum to Keplr

1. **Start your local node** (single or multi-node)
2. **Serve the add-to-keplr page**:
   ```bash
   # Using Python
   python -m http.server 8000
   
   # Or using Node.js
   npx serve .
   ```
3. **Open the page**: http://localhost:8000/add-to-keplr.html
4. **Click "Add Shardeum Local to Keplr"** - this will add the network configuration to your Keplr wallet

### Network Configuration
- **Chain ID**: `shardeum_8119-1` (ethermint format for EVM compatibility)
- **Currency**: SHM (ashm)
- **Decimals**: 18
- **RPC**: http://127.0.0.1:26657
- **REST**: http://127.0.0.1:1317

### Staking with Keplr

After adding the network, you can use the staking interface:
1. Open: http://localhost:8000/keplr-staking.html
2. Connect your Keplr wallet
3. Stake tokens with the default validator

## Development Features

### Pre-funded Development Accounts

The local node comes with 4 pre-funded dev accounts:

| Account | Address (Ethereum) | Address (Cosmos) |
|---------|-------------------|------------------|
| dev0 | 0xC6Fe5D33615a1C52c08018c47E8Bc53646A0E101 | cosmos1cml96vmptgw99syqrrz8az79xer2pcgp84pdun |
| dev1 | 0x963EBDf2e1f8DB8707D05FC75bfeFFBa1B5BaC17 | cosmos1jcltmuhplrdcwp7stlr4hlhlhgd4htqh3a79sq |
| dev2 | 0x40a0cb1C63e026A81B55EE1308586E21eec1eFa9 | cosmos1gzsvk8rruqn2sx64acfsskrwy8hvrmafqkaze8 |
| dev3 | 0x498B5AeC5D439b733dC2F58AB489783A23FB26dA | cosmos1fx944mzagwdhx0wz7k9tfztc8g3lkfk6rrgv6l |

Each account is funded with 1,000,000 SHM tokens.

### Additional Development Options

#### Creating Extra Dev Accounts
```bash
./local_node.sh --additional-users 5  # Creates dev4, dev5, dev6, dev7, dev8
```

#### Custom Mnemonics
```bash
# Create a YAML file with your mnemonics
echo "mnemonics:" > my_mnemonics.yaml
echo '  - "your first mnemonic phrase here"' >> my_mnemonics.yaml
echo '  - "your second mnemonic phrase here"' >> my_mnemonics.yaml

./local_node.sh --mnemonics-input my_mnemonics.yaml
```

#### Development with Remote Debugging
```bash
./local_node.sh --remote-debugging
```

## Network Configuration System

The Shardeum Cosmos fork supports multiple network configurations through JSON files and environment variables, allowing you to easily deploy to different networks without manual intervention.

### Available Networks

The following networks are preconfigured:

| Network | Chain ID | EVM Chain ID | Description |
|---------|----------|--------------|-------------|
| local | shardeum_8119-1 | 8119 | Local development |
| testnet | shardeum_8119-2 | 8119 | Public testnet |
| devnet | shardeum_8119-3 | 8119 | Development network |
| mainnet | shardeum_8118-1 | 8118 | Production mainnet |

### Network Configuration Usage

#### Using Environment Variables

Set the network using environment variables:

```bash
# Set environment for testnet
export SHARDEUM_NETWORK=testnet

# Now run commands without --network flag
./scripts/start_network.sh 4
./scripts/add_node.sh node4
```

Or override specific parameters:

```bash
# Use testnet config but with custom chain ID (use ethermint format for EVM compatibility)
export SHARDEUM_NETWORK=testnet
export SHARDEUM_CHAIN_ID=mychain_8119-1
./scripts/start_network.sh 4
```

#### Environment Variables Reference

All network parameters can be overridden with environment variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `SHARDEUM_NETWORK` | Network name | `testnet` |
| `SHARDEUM_CHAIN_ID` | Cosmos chain ID | `shardeum_8119-1` (local), `shardeum_8119-2` (testnet), etc. |
| `SHARDEUM_EVM_CHAIN_ID` | EVM chain ID | `8119` |
| `SHARDEUM_BASE_DENOM` | Base denomination | `ashm` |
| `SHARDEUM_DISPLAY_DENOM` | Display denomination | `shm` |
| `SHARDEUM_RPC_PORT` | RPC port | `26657` |
| `SHARDEUM_REST_PORT` | REST API port | `1317` |
| `SHARDEUM_JSON_RPC_PORT` | JSON-RPC port | `8545` |
| `SHARDEUM_WEBSOCKET_PORT` | WebSocket port | `8546` |
| `SHARDEUM_GRPC_PORT` | gRPC port | `9090` |

### Creating Custom Networks

1. Create a new configuration file in `configs/`:

```bash
cp configs/testnet.json configs/mynetwork.json
```

2. Edit the configuration:

```json
{
  "name": "mynetwork",
  "chain_id": "mynetwork_9999-1",
  "evm_chain_id": 9999,
  "base_denom": "mycoin",
  "display_denom": "my",
  "decimals": 18,
  "bech32_prefix": "my",
  "ports": {
    "rpc": "26657",
    "rest": "1317", 
    "json_rpc": "8545",
    "websocket": "8546",
    "grpc": "9090"
  },
  "genesis_file": "mynetwork-genesis.json"
}
```

3. Use your custom network:

```bash
./scripts/start_network.sh 4 --network mynetwork
make start-network NETWORK=mynetwork
```

### Genesis Files

Each network has its own genesis file in the `config/environments/` directory:

- `config/environments/mainnet-genesis.json` - Mainnet genesis
- `config/environments/testnet-genesis.json` - Testnet genesis  
- `config/environments/devnet-genesis.json` - Devnet genesis
- `config/environments/local-genesis.json` - Local genesis

The system requires network-specific genesis files for each environment.
