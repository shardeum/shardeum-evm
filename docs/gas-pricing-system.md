# Gas Pricing System Documentation

This guide provides comprehensive documentation of the dual-layer gas pricing system in Shardeum EVM, including calculation logic, parameter configuration, and architectural details.

## Gas Pricing Architecture

Shardeum EVM implements a dual-layer gas pricing system that combines node-level spam protection with network-wide EIP-1559 fee markets.

### Layer 1: Node-Level Minimum Gas Price

Each validator can set their own minimum gas price floor for spam protection:

**Calculation:**

```
nodeRequiredFee = nodeMinGasPrice × gasLimit
```

**Configuration:**

- **Parameter**: `minimum-gas-prices` in `app.toml`
- **Control**: Individual validator control only
- **Change Method**: Edit `app.toml` and restart the specific node
- **Example**: `minimum-gas-prices = "0ashm"` (currently set to 0 in dev)
- **Governance**: Cannot be changed via governance (by design for security)

### Layer 2: Global Network Minimum Gas Price

Network-wide minimum enforced through on-chain governance:

**Calculation:**

```
globalRequiredFee = globalMinGasPrice × gasLimit
```

**Configuration:**

- **Parameter**: `min_gas_price` in feemarket module
- **Control**: Network-wide governance
- **Change Method**: Submit governance proposal for parameter change
- **Initial Value**: Set in genesis file (currently "0" in local genesis)
- **Runtime Updates**: Can be changed via governance without node restarts

### EIP-1559 Dynamic Fee Transactions

For transactions using EIP-1559 dynamic fees:

**Effective Gas Price Calculation:**

```
effectiveGasPrice = min(baseFee + tipCap, feeCap)
```

**Final Transaction Cost:**

```
txCost = value + (gasLimit × gasPrice)
```

**Base Fee Adjustment:**
Base fee adjusts automatically based on network congestion using the elasticity multiplier and base fee change denominator.

## Feemarket Parameters

The feemarket module in genesis contains these configurable parameters:

```json
"feemarket": {
  "params": {
    "no_base_fee": false,                    // Disables EIP-1559 base fee mechanism
    "base_fee_change_denominator": 8,        // Controls base fee adjustment rate
    "elasticity_multiplier": 2,              // Network congestion elasticity
    "enable_height": "0",                   // Height to enable fee market
    "base_fee": "1000000000.000000000000000000",     // Initial base fee
    "min_gas_price": "1000000000.000000000000000000", // Global minimum (globalMinGasPrice)
    "min_gas_multiplier": "0.500000000000000000"      // Minimum gas price multiplier
  }
}
```

### Parameter Mapping

- **globalMinGasPrice** ↔ `min_gas_price` (feemarket param)
- **nodeMinGasPrice** ↔ `minimum-gas-prices` (app.toml config)

## Transaction Validation Logic

Both layers must be satisfied for transaction acceptance:

1. **Node-Level Check**: Transaction gas price ≥ nodeMinGasPrice
2. **Global-Level Check**: Transaction gas price ≥ globalMinGasPrice
3. **Final Validation**: Transaction must pass both checks

**Implementation Location**: `ante/evm/mono_decorator.go:134` and `ante/evm/dual_layer_fee.go:32`

The core validation logic:

```go
// Layer 1: Node-level floor validation
nodeRequiredFee := nodeMinGasPrice.Mul(gasLimit)
if fee.LT(nodeRequiredFee) {
    return errorsmod.Wrapf(errortypes.ErrInsufficientFee, "fee below node minimum")
}

// Layer 2: Global minimum validation
globalRequiredFee := globalMinGasPrice.Mul(gasLimit)
if fee.LT(globalRequiredFee) {
    return errorsmod.Wrapf(errortypes.ErrInsufficientFee, "fee below global minimum")
}
```

## Parameter Control Methods

### Global Parameters (Network-wide)

**Who Controls**: Network governance (all validators with voting power)
**Change Method**:

1. Submit governance proposal with new feemarket parameters
2. Validators vote on proposal
3. If passed, parameters update immediately across all nodes
4. No node restarts required

**Initial Setup**:

- Set in genesis file during network bootstrap
- Single validator networks can change via governance if they have voting power

### Node Parameters (Individual Validator)

**Who Controls**: Individual validator operators
**Change Method**:

1. Edit `minimum-gas-prices` in node's `app.toml`
2. Restart the specific validator node
3. Only affects that validator's transaction acceptance

**Security Design**:

- Cannot be controlled via governance to prevent centralized manipulation
- Each validator maintains sovereign control over their minimum requirements

## Multi-Node Scenarios

**Question**: If nodes have different `minimum-gas-prices` values, what determines the global behavior?

**Answer**:

- Each validator enforces their own `minimum-gas-prices` locally
- The global `min_gas_price` (from feemarket) applies to ALL validators
- A transaction must satisfy BOTH the individual validator's minimum AND the global minimum
- The governance-set global value remains consistent regardless of individual node configurations

**Example**:

- Node A: `minimum-gas-prices = "5ashm"`
- Node B: `minimum-gas-prices = "10ashm"`
- Node C: `minimum-gas-prices = "2ashm"`
- Global: `min_gas_price = "8ashm"`

Result: Node A requires max(5, 8) = 8, Node B requires max(10, 8) = 10, Node C requires max(2, 8) = 8

## Testing and Implementation

For practical testing procedures and step-by-step commands to validate the gas pricing system, see [`GAS_PRICING_TEST_GUIDE.md`](./GAS_PRICING_TEST_GUIDE.md).

## Key Implementation Details

- **Dual-layer validation**: Transactions must meet both node-level `minimum-gas-prices` (from app.toml) and global `min_gas_price` (from feemarket params)
- **Runtime updates**: Global parameters update immediately when governance proposals pass
- **Ante handler**: Located in `ante/evm/mono_decorator.go:134`, validates gas pricing for all EVM transactions using `CheckDualLayerFee`
- **No restart required**: Parameter changes take effect immediately without network restart
- **Implementation files**:
  - `ante/evm/dual_layer_fee.go`: Core dual-layer validation logic
  - `ante/evm/mono_decorator.go`: Integration with transaction processing
  - `x/feemarket/`: Module handling global parameter governance
