# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Shardeum Cosmos fork - a plug-and-play solution that adds EVM compatibility to Cosmos SDK chains. It's based on evmOS and maintained as an Apache 2.0 open-source project.

## Build and Development Commands

### Core Build Commands
```bash
# Build the shardeumd binary
make build                     # Build to ./build/shardeumd
make install                   # Install to $GOPATH/bin
make build-linux              # Cross-compile for Linux AMD64
```

### Running the Chain
```bash
# Single node development
./local_node.sh               # Start single node with default config
./local_node.sh -y            # Overwrite previous database
./local_node.sh --no-install  # Skip binary installation
./local_node.sh --remote-debugging  # Build for remote debugging

# Multi-node testnet
make start-network                    # Start 4 nodes on local network (default)
make start-network NETWORK=testnet   # Start 4 nodes on testnet network
make start-network NODES=6           # Start 6 nodes on local network
make start-network NETWORK=mainnet NODES=6  # Start 6 nodes on mainnet

# Add nodes to running network
make add-node NODE_ID=node4                          # Add node4 to local network (default)
make add-node NODE_ID=node5 NETWORK=testnet          # Add node5 to testnet network
make add-node NODE_ID=node6 SEED_RPC=http://localhost:26657  # Add node6 with custom seed RPC

# Direct script usage (alternative)
./scripts/start_network.sh 6 --network testnet       # Start 6 nodes on testnet
./scripts/add_node.sh node4 --network mainnet        # Add node4 to mainnet
```

### Testing Commands
```bash
# Unit tests
make test                     # Run all unit tests
make test-unit               # Unit tests only
make test-race               # Race condition tests
make test-shardeumd               # Test shardeumd module specifically
make test-unit-cover         # Generate coverage report

# Other tests
make test-fuzz               # Fuzz testing
make test-scripts            # Script tests
make test-solidity           # Solidity contract tests
```

### Code Quality Commands
```bash
# Linting
make lint                    # Run all linters
make lint-go                # Go code linting
make lint-python            # Python code linting
make lint-contracts         # Solidity contract linting
make lint-fix               # Auto-fix Go linting issues
make lint-fix-contracts     # Auto-fix Solidity linting issues

# Formatting
make format                 # Format all code
make format-go              # Format Go code
make format-python          # Format Python code
make format-shell           # Format shell scripts
```

### Protobuf Commands
```bash
make proto-all              # Format, lint and generate protos
make proto-gen              # Generate implementations from .proto files
make proto-format           # Format protobuf files
make proto-lint             # Lint protobuf files
make proto-check-breaking   # Check for breaking changes
```

### Smart Contract Commands
```bash
make contracts-all          # Compile contracts and clean up
make contracts-compile      # Compile Solidity contracts
make contracts-clean        # Clean compilation artifacts
```

## Available Networks

The following predefined networks are available for development and testing:

### Network Configurations
- **local**: Development network (`shardeum_8117-1`, EVM Chain ID: 8117)
- **testnet**: Test network (`shardeum_8119-2`, EVM Chain ID: 8119)
- **devnet**: Development staging network (`shardeum_8119-3`, EVM Chain ID: 8119)
- **mainnet**: Production network (`shardeum_8118-1`, EVM Chain ID: 8118)

**Note**: All networks use the ethermint chain ID format (`{name}_{evmChainId}-{version}`) for automatic EVM compatibility detection by wallets like Keplr and ping.pub. Each network has a unique version number or chain ID to prevent replay attacks across networks.

### Network Selection
Use the `NETWORK` parameter with make commands or `--network` flag with scripts:
- **Default**: `local` network is used when no network is specified
- **Makefile**: `make start-network NETWORK=testnet`
- **Scripts**: `./scripts/start_network.sh 4 --network mainnet`

### Environment Variables
You can also set the network using environment variables:
```bash
export SHARDEUM_NETWORK=testnet
./scripts/start_network.sh 4  # Will use testnet
```

## Architecture Overview

### Module Structure
The chain is organized into several key modules under `x/`:
- `x/vm` - Core EVM implementation
- `x/erc20` - Single token representation for IBC/ERC-20 alignment
- `x/feemarket` - EIP-1559 fee market mechanism
- `x/ibc` - IBC callbacks and middleware
- `x/precisebank` - Precise bank operations

### Key Components
- **shardeumd**: Shardeum chain binary in `shardeumd/` directory demonstrating Shardeum Cosmos integration
- **Precompiles**: Native Cosmos functionality accessible from Solidity (`precompiles/` directory)
- **EVM Extensions**: Bridge between Cosmos SDK modules and Solidity contracts
- **JSON-RPC**: Ethereum-compatible RPC endpoints in `rpc/` directory
- **Ante Handlers**: Transaction validation and fee processing in `ante/` directory

### Development Configuration
- **Chain IDs**: Automatically selected based on network (see Available Networks section)
- **Native Token**: SHM (ashm, 18 decimals)
- **Default Ports** (configurable per network):
  - RPC: 26657
  - REST API: 1317
  - JSON-RPC: 8545
  - WebSocket: 8546
  - GRPC: 9090
- **Pre-funded Dev Accounts**: 4 accounts (dev0-dev3) with both Ethereum and Cosmos addresses
- **Default Network**: `local` for development, configurable via `NETWORK` parameter

### Testing Infrastructure
- Integration tests in `tests/integration/` - reusable test harnesses for precompiles
- System tests in `tests/systemtests/` - end-to-end testing
- EVM compatibility tests in `tests/evm-tools-compatibility/` - Hardhat, Foundry, Truffle support
- JSON-RPC tests in `tests/jsonrpc/`