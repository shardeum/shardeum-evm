# Shardeum Cosmos Network Setup Guide

A complete guide for setting up and expanding a Shardeum Cosmos network from scratch, suitable for complete beginners.

## Prerequisites

Before starting, ensure you have:

1. **Environment Variables Set**:
   ```bash
   export SHARDEUM_CONFIG_DIR="/absolute/path/to/your/config/directory"
   export SHARDEUM_NETWORK="local"  # or testnet, devnet, mainnet
   ```

2. **Required Tools**:
   - `jq` (JSON processor)
   - `curl` (for HTTP requests)
   - `bc` (calculator for large numbers)

3. **Built Binary**:
   ```bash
   make build  # Creates shardeumd binary
   ```

## Step 1: Start Network with Single Node

### 1.1 Initialize the Network
Start with a single validator node (node0):

```bash
# Start a single-node network (default is 4 nodes, we specify 1)
./scripts/start_network.sh 1 --network local
```

This creates:
- **node0**: A validator node with voting power
- Pre-funded dev accounts (dev0-dev3) in the keyring
- Genesis file with initial validator setup
- Network running on default ports:
  - RPC: http://localhost:26657
  - API: http://localhost:1317

### 1.2 Verify Network is Running
Check network status:
```bash
curl -s http://localhost:26657/status | jq '.result.sync_info'
```

You should see the chain producing blocks.

## Step 2: Add a Second Node (Future Validator)

### 2.1 Add Node to Network
Add node1 which will become a validator:

```bash
# Add node1 to the existing network
./scripts/add_node.sh node1 --node-type validator --seed-rpc http://localhost:26657
```

This creates:
- **node1**: Initially a non-validating node
- Automatically discovers and connects to node0 via seed mechanism
- Assigned ports:
  - RPC: http://localhost:26658
  - API: http://localhost:1318


## Step 3: Create Validator Key and Fund It

### 3.1 Create Validator Key
Create a key for the new validator on node1:

```bash
# Access node1's keyring and create validator key
cd .local/node1
shardeumd keys add validator-node1 --keyring-backend test --home .
```

This outputs:
- **Address**: The account address (starts with `shardeum...`)
- **Mnemonic**: Backup phrase (save this securely!)

### 3.2 Get the Validator Address
```bash
# Get the address for funding
VALIDATOR_ADDR=$(shardeumd keys show validator-node1 -a --keyring-backend test --home .)
echo "Validator address: $VALIDATOR_ADDR"
```

### 3.3 Fund the Validator Account
Send tokens to the validator account using Keplr wallet:

1. **Connect Keplr** to your local network (http://localhost:8547 for JSON-RPC)
2. **Send tokens** from your funded dev account to the validator address
3. **Amount needed**: Send at least 2000 SHM (2000000000000000000000ashm) to cover:
   - Validator stake: 1000 SHM minimum
   - Transaction fees: ~1000 SHM buffer

```bash
# Copy this address to send tokens to via Keplr:
echo "Send tokens to: $VALIDATOR_ADDR"
```

### 3.4 Verify Funding
Check the validator account balance:
```bash
shardeumd query bank balances $VALIDATOR_ADDR --node tcp://localhost:26657
```

## Step 4: Create the Validator

### 4.1 Create Validator Transaction
Now promote node1 to be an active validator:

```bash
# Create validator for node1
./scripts/create_validator.sh node1 --validator-key validator-node1 --moniker "Node1-Validator"
```

This:
- Creates a validator with staked tokens
- Sets commission rates (10% default)
- Registers the validator in the staking module

### 4.2 Verify Validator Creation
Check that the validator is now active:
```bash
# Check validator set
shardeumd query staking validators --node tcp://localhost:26657


You should now see 2 validators in the active set, both with voting power.

## Step 5: Add Full Nodes (Non-Validators)

### 5.1 Add Full Node
Add a full node that participates in the network but doesn't validate:

```bash
# Add node2 as a full node (not validator)
./scripts/add_node.sh node2 --node-type full-node --seed-rpc http://localhost:26657
```

Full nodes:
- Have JSON-RPC enabled for DApp connections
- Sync all blockchain data
- Don't participate in consensus
- Assigned ports:
  - RPC: http://localhost:26659
  - API: http://localhost:1319
  - JSON-RPC: http://localhost:8549
  - WebSocket: ws://localhost:8550

### 5.2 Verify Full Node Operation
Check JSON-RPC functionality:
```bash
# Test JSON-RPC endpoint
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  http://localhost:8549
```

## Summary of Network Architecture

After completing all steps, you'll have:

### Network Topology:
- **node0**: Original validator (ports 26657, 1317)
- **node1**: Additional validator (ports 26658, 1318) 
- **node2**: Full node with JSON-RPC (ports 26659, 1319, 8549, 8550)

### Key Management:
- **node0**: Has "validator" account pre-funded with 100M tokens, plus dev accounts (dev0-dev3) in keyring
- **node1**: Has validator key (validator-node1) for staking
- **node2**: No keys needed (full node only)

### Port Allocation Pattern:
- **RPC**: 26657 + node_number
- **P2P**: 27656 + node_number  
- **API**: 1317 + node_number
- **gRPC**: 9090 + node_number
- **JSON-RPC**: 8545 + (node_number * 2)
- **WebSocket**: 8546 + (node_number * 2)

## Common Operations

### Check Network Status
```bash
# View all connected peers
curl -s http://localhost:26657/net_info | jq '.result.peers'

# Check latest block
curl -s http://localhost:26657/status | jq '.result.sync_info.latest_block_height'

# View validator set
./build/shardeumd query staking validators --node tcp://localhost:26657
```

### Stop the Network
```bash
# Stop all nodes
pkill -f "shardeumd.*shardeum-local"

# Or stop individual node (if you know the PID)
kill $(cat .local/node1/node.pid)
```

### View Logs
```bash
# View node logs
tail -f .local/node0/node.log
tail -f .local/node1/node.log
tail -f .local/node2/node.log
```

## Troubleshooting

Remember to set environment variables as you open up new terminals!

### Seed node won't start
- Try setting the environment variable BINARY to the absolute path of shardeumd  `export BINARY="<path to shardeumd... is in build folder>`
- You may need to add the go binary folder do your PATH.  this will vary but for "trudy" using the asdf package manager and 1.25.0 it may look like this:  `export PATH="/home/trudy/.asdf/installs/golang/1.25.0/bin:$PATH"`

### Node Won't Connect
- Verify the seed node (node0) is running: `curl -s http://localhost:26657/status`
- Check node logs for connection errors: `tail -f .local/nodeX/node.log`
- Ensure ports aren't blocked by firewall

### Validator Creation Fails
- Verify account has sufficient balance (>1.5 SHM for stake + fees)
- Check that node is fully synced before creating validator
- Ensure validator key exists: `./build/shardeumd keys list --keyring-backend test --home .local/nodeX`

### JSON-RPC Not Working
- Only full nodes have JSON-RPC enabled (not validators for security)
- Check the correct port: node1=8547, node2=8549, etc.
- Verify node is running: `curl -s http://localhost:26659/status`

This guide provides a complete foundation for understanding and operating a Shardeum Cosmos network, from initial setup through adding validators and full nodes.
