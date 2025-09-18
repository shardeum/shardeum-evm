# Shardeum EVM Blockchain

A high-performance EVM-compatible blockchain that provides seamless Ethereum compatibility with enhanced scalability and interoperability features.

## Getting started


### Multi-Node Testnet

For testing with multiple validators, you can use either the Makefile target or the script directly:

```bash
# Using Makefile (recommended)
make start-network

# Or directly using the script
./scripts/start_network.sh [number_of_nodes]
```

This script will:
- Build the `shardeumd` binary 
- Initialize nodes with the Shardeum testnet configuration
- Set up a bootstrap validator on node0
- Configure seed-based peer discovery (node0 acts as seed)
- Start all nodes with proper port assignments

Examples:
```bash
# Using Makefile
make start-network                 # Start 4 nodes (default)
make start-network NODES=6         # Start 6 nodes

# Using script directly 
./scripts/start_network.sh         # Start 4 nodes (default)
./scripts/start_network.sh 6       # Start 6 nodes
```

#### Adding Nodes to Running Network

You can dynamically add more nodes to an existing testnet:

```bash
# Using Makefile (recommended)
make add-node NODE_ID=node4 [SEED_RPC=http://localhost:26657]

# Or directly using the script
./scripts/add_node.sh <node_id> [seed_rpc_endpoint]
```

Examples:
```bash
# Using Makefile
make add-node NODE_ID=node4                           # Add node4, connect to default seed at localhost:26657
make add-node NODE_ID=node5 SEED_RPC=http://localhost:26658  # Add node5, connect to specific node as seed

# Using script directly
./scripts/add_node.sh node4                           # Add node4, connect to default seed at localhost:26657
./scripts/add_node.sh node5 http://localhost:26658    # Add node5, connect to specific node as seed
./scripts/add_node.sh 6                               # Add node6 (ID automatically prefixed)
```

The script will:
- Initialize the new node with proper configuration
- Download genesis from existing network
- Configure seed-based peer discovery
- Assign available ports automatically
- Start the node and connect to the network

#### Stopping Nodes

To stop all nodes:
```bash
pkill -f 'shardeumd.*shardeum-testnet'
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
- **Chain ID**: `shardeum`
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
