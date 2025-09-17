<img
src="repo_header.png"
alt="Shardeum Cosmos - A plug-and-play solution that adds EVM compatibility and customizability to your chain"
/>

**Please note**: This repo is undergoing changes while the code is being audited and tested. For the time being we will
be making v0.x releases. Some breaking changes might occur. While the original evmOS repo is currently being used in
production on a few chains without fault, Shardeum will only mark the Shardeum Cosmos repository as stable with a v1
release after the audit, key stability features and benchmarking are completed.

**Visit the official documentation for Shardeum**: [docs.shardeum.org](https://docs.shardeum.org/)

## What is Shardeum Cosmos?

Shardeum Cosmos is a plug-and-play solution that adds EVM compatibility
and customizability to your Cosmos SDK chain.

- Build an app-chain with the control and extensibility of the Cosmos SDK
- With native support for EVM as VM and seamless EVM<>Cosmos wallet / token / user experience.
- Leverage IBC with EVM, native support of ERC20 on Cosmos, and more with extensions and precompiles.

Shardeum Cosmos is a fork of [evmOS](https://github.com/evmos/OS), maintained by Shardeum
after the latter funded Tharsis to open-source the original codebase.

**Shardeum Cosmos is fully open-source under the Apache 2.0 license.** With this open-sourced version, you can get:

- Full access to Shardeum Cosmos's modules and updates
- Smooth onboarding experience for an existing or new Cosmos chain
- Full access to product partnerships (block explorers, RPCs, indexers etc.)
- Continuous upgrades, access to product and engineering support

**Want to use Shardeum Cosmos but want to discuss it with an expert first? [Contact the Shardeum team](https://shardeum.org/contact).**

For live discussions or support regarding advisories, join the #cosmos-tech channel in Slack.
[Get a Slack invite here](https://forms.gle/A8jawLgB8zuL1FN36) or join the [Telegram Group](https://t.me/cosmostechstack)

## Plug-in Shardeum Cosmos into your chain

### Integration

Shardeum Cosmos can easily be integrated into your existing chain
or added during the development of your upcoming chain launch
by importing Shardeum Cosmos as a go module library.
The Shardeum team provides you with integration guides and core protocol support depending on your needs and configurations.
**Updated documentation will be releasing soon!**

### Configurations

Shardeum Cosmos solution is engineered to provide unique flexibility,
empowering you to tailor every aspect of your Ethereum Virtual Machine (EVM) environment.
Whether you're launching a new blockchain or optimizing an existing one,
Shardeum Cosmos offers a suite of features designed to meet the unique demands of your project.

#### Powerful defaults

Shardeum Cosmos's modules come out of the box with defaults that will get you up and running instantly.

When integrating all available modules you will get a *permissionless EVM-enabled* blockchain
that *exposes JSON-RPC* endpoints for connectivity with all EVM tooling
like wallets ([MetaMask](https://metamask.io/), [Rabby](https://rabby.io/), and others)
or block explorers ([Blockscout](https://docs.blockscout.com/) and others).
You will have access to *all of Shardeum Cosmos' extensions*,
which enable access to chain-native functionality
through [Solidity](https://docs.soliditylang.org/en/v0.8.26/) smart contracts.
Your chain provides a *seamless use of any IBC asset in the EVM*
without liquidity fragmentation between wrapped and unwrapped tokens.
Transaction surges are handled by the *self-regulating fee market mechanism* based on EIP-1559
and EIP-712 allows for *structured data si gning* for arbitrary messages.

*Everything* can be controlled by on-chain governance
to create alignment between chain teams and their communities.

#### Extensive customizations

Based on these powerful defaults, the feature set is easily and highly customizable:

- *Permissioned/Restricted EVM*

  Maintain control over your network with permissioned or restricted EVM capabilities.
  Implement customized access controls to either blacklist or whitelist individual addresses for calling
  and/or creating smart contracts on the network.

- *EVM Extensions*

  Extend the capabilities of your EVM!
  These EVM extensions allow functionality
  that is native to Cosmos SDK modules to be accessible from Solidity smart contracts.
  We provide a selection of plug-and-play EVM extensions that are ready to be used *today*.

  Push the boundaries of what’s possible with fully custom EVM extensions.
  Develop the  business logic that sets your chain apart from others with the mature tooling for the Go language
  and offer its functionality to the masses of Solidity smart contract developers
  to integrate in their dApps.

- *Single Token Representation v2 & ERC-20 Module*

  Simplify token management with Single Token Representation v2
  and our `x/erc20` module to elevate the user experience on your chain.
  Align IBC coins and ERC-20s and say goodbye to fragmented liquidity.
  One balance. In every tool.

- *EIP-1559 Fee Market Mechanism*

  Take control of transaction costs with our
  ready-to-use [EIP-1559 fee market](https://eips.ethereum.org/EIPS/eip-1559) solution.
  Tailor fee structures to suit your network’s specific needs,
  balancing user affordability with network sustainability.
  Or disable it altogether.

- *JSON-RPC Server*

  There is full control over the exposed namespaces and fine-grained control of the
  [JSON-RPC server](https://cosmos-docs.mintlify.app/docs/api-reference/ethereum-json-rpc).
  Adjust the configuration to your liking,
  including custom timeouts for EVM calls or HTTP requests,
  maximum block gas, the number of maximum open connections, and more.

- *EIP-712 Signing*

  You have the option to integrate our [EIP-712 signature](https://eips.ethereum.org/EIPS/eip-712) implementation,
  which allows Cosmos SDK messages to be signed with EVM wallets like MetaMask.

- *Custom Improvement Proposals (Opcodes)*

  Any Shardeum Cosmos user is provided the opportunity to customize bits of their EVM opcodes and add new ones.
  Read more on [custom operations here](https://cosmos-docs.mintlify.app/docs/documentation/smart-contracts/custom-improvement-proposals#custom-improvement-proposals).

### Forward-compatibility with Ethereum

Ethereum-equivalence describes any EVM solution,
that is identical in transaction execution to the Ethereum client.
It does not more, but also not less than that.
Ethereum-compatible means,
that the EVM can be set up to run every transaction that is valid on Ethereum,
while the handling of the transactions can diverge in e.g. result or cost.

We like to coin the term **forward-compatible**
as a description of our EVM solution,
meaning that any Shardeum Cosmos chain can run any valid smart contract
from Ethereum but can also implement new features that are
not (yet) available on the standard Ethereum VM,
thus moving the standard forward.

## Getting started

### Quick Start - Single Node

To run the Shardeum `shardeumd` chain, run the script using `./local_node.sh`
from the root folder of the repository.

```bash
./local_node.sh
```

This will:
- Build the `shardeumd` binary
- Initialize a local testnet with the chain ID `shardeum`
- Create funded dev accounts (dev0, dev1, dev2, dev3)
- Start the node with JSON-RPC APIs enabled

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

### Chain Configuration

The Shardeum testnet includes:
- **Native Token**: SHM (ashm)
- **Block Time**: ~1-2 seconds (optimized for development)
- **Gas Price**: 0 ashm (free transactions for development)
- **Max Block Gas**: 10,000,000
- **Active Precompiles**: Bank, Distribution, Staking, ERC20, Gov, ICS20, Slashing
- **Governance**: Fast proposal periods (30s voting, 15s expedited)

### Migrations

We provide upgrade guides [here](./docs/migrations) for upgrading your chain from various Shardeum Cosmos versions.

### Testing

All test scripts are found in `Makefile` in the root of the repository.
Listed below are the commands for various tests:

#### Unit Testing

```bash
make test-unit
```

#### Coverage Test

This generates a code coverage file `filtered_coverage.txt` and prints out the
covered code percentage for the working files.

```bash
make test-unit-cover
```

#### Fuzz Testing

```bash
make test-fuzz
```

#### Solidity Tests

```bash
make test-solidity
```

#### Benchmark Tests

```bash
make benchmark
```

## Contributing

We welcome open source contributions and discussions! For more on contributing, read our [guide](./CONTRIBUTING.md).

## Open-source License & Credits

Shardeum Cosmos is open-source under the Apache 2.0 license, an extension of the license of the original codebase (https://github.com/evmos/OS)
created by Tharsis and the evmOS team - who conducted the foundational work for EVM compatibility and
interoperability in Cosmos.

### Key Contributors

We at Shardeum want to thank our key contributors at [B-Harvest](https://bharvest.io/) and 
[Mantra](https://www.mantrachain.io/) for contributing to and helping us drive the development of Shardeum Cosmos.
