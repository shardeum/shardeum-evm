# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Cosmos EVM fork (Shardeum-Cosmos) - a plug-and-play solution that adds EVM compatibility to Cosmos SDK chains. It's based on evmOS and maintained as an Apache 2.0 open-source project.

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
make start-network            # Start 4 nodes (default)
./scripts/start_network.sh 6  # Start 6 nodes
make add-node NODE_ID=node4  # Add a new node to running network
```

### Testing Commands
```bash
# Unit tests
make test                     # Run all unit tests
make test-unit               # Unit tests only
make test-race               # Race condition tests
make test-evmd               # Test evmd module specifically
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

## Architecture Overview

### Module Structure
The chain is organized into several key modules under `x/`:
- `x/vm` - Core EVM implementation
- `x/erc20` - Single token representation for IBC/ERC-20 alignment
- `x/feemarket` - EIP-1559 fee market mechanism
- `x/ibc` - IBC callbacks and middleware
- `x/precisebank` - Precise bank operations

### Key Components
- **evmd**: Example chain binary in `evmd/` directory demonstrating Cosmos EVM integration
- **Precompiles**: Native Cosmos functionality accessible from Solidity (`precompiles/` directory)
- **EVM Extensions**: Bridge between Cosmos SDK modules and Solidity contracts
- **JSON-RPC**: Ethereum-compatible RPC endpoints in `rpc/` directory
- **Ante Handlers**: Transaction validation and fee processing in `ante/` directory

### Development Configuration
- **Chain ID**: `shardeum` (local), `cosmos_262144-1` (evmd example)
- **Native Token**: SHM (ashm, 18 decimals)
- **Default Ports**:
  - RPC: 26657
  - REST API: 1317
  - JSON-RPC: 8545
  - WebSocket: 8546
- **Pre-funded Dev Accounts**: 4 accounts (dev0-dev3) with both Ethereum and Cosmos addresses

### Testing Infrastructure
- Integration tests in `tests/integration/` - reusable test harnesses for precompiles
- System tests in `tests/systemtests/` - end-to-end testing
- EVM compatibility tests in `tests/evm-tools-compatibility/` - Hardhat, Foundry, Truffle support
- JSON-RPC tests in `tests/jsonrpc/`