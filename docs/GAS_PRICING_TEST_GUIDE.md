# Gas Pricing System Test Guide

## Overview

This guide provides an overview of testing the dual-layer gas pricing system in Shardeum EVM, which allows runtime parameter changes through governance proposals without network restarts.

## System Architecture

### Dual-Layer Gas Pricing
The system implements two levels of gas price validation:

1. **Node-level minimum**: Configured in `app.toml` (`minimum-gas-prices`)
2. **Global minimum**: Managed via governance through feemarket module parameters

All transactions must satisfy **both** requirements. The ante handler at `ante/evm/mono_decorator.go:122-137` performs this validation using the `CheckDualLayerFee` function.

### Runtime Parameter Updates
The `x/feemarket` module supports governance-based parameter updates, enabling:
- Dynamic adjustment of minimum gas prices
- Base fee modifications
- Fee market configuration changes
- No network restart required

## Local Testing Configuration

For development and testing purposes, the local genesis configuration includes several modifications:

### Modified Governance Parameters
- **Voting period**: Reduced to 60 seconds (from 48 hours)
- **Quorum**: Lowered to 0.000001 (from 33.4%)
- **Deposit requirement**: 10 SHM minimum

### Pre-funded Development Accounts
- 4 development accounts (dev0-dev3) with sufficient balances
- Validator account with genesis delegation
- Authority account for governance proposals

### Authority Configuration
- Governance authority: `shardeum10d07y265gmmuvt4z0w9aw880jnsr700jzj92zh`
- Used for module parameter update messages

## Testing Workflow Overview

### 1. Network Setup
Start a local 4-node network using the configured genesis file with testing optimizations.

```bash
./scripts/start_network.sh 4 --network local
```

### 2. Voting Power Preparation
Development accounts need voting power to participate in governance:

```bash
# First get the current validator address
VALIDATOR=$(shardeumd query staking validators --output json | jq -r '.validators[0].operator_address')

# Delegate from validator to give dev accounts voting power
shardeumd tx staking delegate $VALIDATOR 1000000000000000000ashm \
  --from dev0 --chain-id shardeum-local --fees 20000ashm -y

shardeumd tx staking delegate $VALIDATOR 1000000000000000000ashm \
  --from dev1 --chain-id shardeum-local --fees 20000ashm -y

shardeumd tx staking delegate $VALIDATOR 1000000000000000000ashm \
  --from dev2 --chain-id shardeum-local --fees 20000ashm -y

shardeumd tx staking delegate $VALIDATOR 1000000000000000000ashm \
  --from dev3 --chain-id shardeum-local --fees 20000ashm -y
```

Wait for delegation to be processed (~6 seconds).

### 3. Governance Proposal Workflow
Submit a parameter change proposal to update minimum gas price:

```bash
# Check current parameters
shardeumd query feemarket params

# Submit governance proposal
shardeumd tx gov submit-proposal docs/sample-governance-proposal.json \
  --from dev0 --chain-id shardeum-local --fees 20000ashm --gas 300000 -y

# Vote on proposal from all accounts
shardeumd tx gov vote 1 yes --from dev0 --chain-id shardeum-local --fees 20000ashm -y
shardeumd tx gov vote 1 yes --from dev1 --chain-id shardeum-local --fees 20000ashm -y
shardeumd tx gov vote 1 yes --from dev2 --chain-id shardeum-local --fees 20000ashm -y
shardeumd tx gov vote 1 yes --from dev3 --chain-id shardeum-local --fees 20000ashm -y
```

Wait for voting period to complete (60 seconds).

### 4. Runtime Verification
Confirm parameter changes took effect:

```bash
# Check proposal status (should show PROPOSAL_STATUS_PASSED)
shardeumd query gov proposal 1

# Verify parameter update (min_gas_price should now show "5000000000.000000000000000000")
shardeumd query feemarket params
```

### 5. Transaction Validation Testing
Test the dual-layer gas pricing enforcement:

```bash
# Test low gas price transaction (should fail with "insufficient fee" error)
shardeumd tx evm send dev0 0x1234567890123456789012345678901234567890 1ashm \
  --from dev0 --chain-id shardeum-local --gas-prices 1ashm -y

# Test adequate gas price transaction (should succeed with code: 0)
shardeumd tx evm send dev0 0x1234567890123456789012345678901234567890 1ashm \
  --from dev0 --chain-id shardeum-local --gas-prices 5000000000ashm -y
```

## Key Implementation Files

- `ante/evm/mono_decorator.go`: Dual-layer fee validation logic
- `x/feemarket/types/feemarket.pb.go`: Parameter type definitions
- `config/local-genesis.json`: Local testing configuration
- `docs/sample-governance-proposal.json`: Example proposal format

## Expected Results

A successful test demonstrates:
1. ✅ Governance proposal passes with sufficient votes
2. ✅ Parameters update at runtime without restart
3. ✅ Transaction validation enforces new gas requirements
4. ✅ System maintains dual-layer validation integrity

## Production Considerations

When deploying to production networks:
- Restore standard governance parameters (48-hour voting, 33.4% quorum)
- Remove testing-specific authority configurations
- Use appropriate deposit requirements for proposal costs
- Plan parameter changes with adequate notice to validators

This testing framework validates that the gas pricing system correctly implements runtime parameter updates while maintaining transaction fee enforcement at both node and global levels.