# Shardeum Cosmos Minting & Distribution Analysis

## Overview

This document provides a comprehensive analysis of the Shardeum Cosmos blockchain's minting and reward distribution mechanisms, covering configuration parameters, reward calculations, voting power influence, and implementation strategies for achieving different reward distribution models.

## 🔧 Current Mint Module Configuration

### Default Parameters (from genesis)

```json
"mint": {
  "minter": {
    "inflation": "0.130000000000000000",  // 13% starting inflation
    "annual_provisions": "0.000000000000000000"
  },
  "params": {
    "mint_denom": "ashm",                    // Base denomination
    "inflation_rate_change": "0.130000000000000000",  // 13% adjustment rate per year
    "inflation_max": "0.200000000000000000",          // 20% maximum inflation
    "inflation_min": "0.070000000000000000",          // 7% minimum inflation
    "goal_bonded": "0.670000000000000000",            // 67% target staking ratio
    "blocks_per_year": "6311520"                      // ~5 second blocks
  }
}
```

### Key Configuration Files

- **Genesis Function**: `shardeumd/genesis.go` - `NewMintGenesisState()`
- **Chain Denomination**: `shardeumd/cmd/shardeumd/config/shardeumd_config.go`
- **Module Integration**: `shardeumd/app.go` (lines 357-365)
- **Distribution Precompile**: `precompiles/distribution/distribution.go`

## 📊 Reward Calculation Mechanics

### Inflation Adjustment Formula

The system dynamically adjusts inflation based on staking participation:

```javascript
// Under-staked scenario (currentBonded < goalBonded)
if (currentBonded < goalBonded) {
  gap = goalBonded - currentBonded
  perBlockAdjustment = (inflationRateChange × gap) / blocksPerYear
  newInflation = min(currentInflation + perBlockAdjustment, inflationMax)
}

// Over-staked scenario (currentBonded > goalBonded)
else if (currentBonded > goalBonded) {
  gap = currentBonded - goalBonded
  perBlockAdjustment = (inflationRateChange × gap) / blocksPerYear
  newInflation = max(currentInflation - perBlockAdjustment, inflationMin)
}
```

### Annual Provisions Calculation

```javascript
// Total tokens minted per year
totalMinted = currentInflation × totalSupply

// Community pool allocation (2% default)
communityPool = totalMinted × communityTax

// Validator rewards (remaining 98%)
validatorRewards = totalMinted - communityPool

// Per-block distribution
perBlockProvision = validatorRewards / blocksPerYear
```

### APR Calculation

```javascript
totalStaked = totalSupply × currentBondedRatio
baseAPR = validatorRewards / totalStaked
delegatorAPR = baseAPR × (1 - validatorCommission)
```

## ⚖️ Distribution Module Configuration

### Current Distribution Parameters

```json
"distribution": {
  "params": {
    "community_tax": "0.020000000000000000",        // 2% to community pool
    "base_proposer_reward": "0.000000000000000000",  // 0% base proposer bonus
    "bonus_proposer_reward": "0.000000000000000000", // 0% bonus proposer reward
    "withdraw_addr_enabled": true
  }
}
```

### Staking Module Configuration

```json
"staking": {
  "params": {
    "unbonding_time": "1814400s",        // 21 days unbonding period
    "max_validators": 100,               // Maximum 100 active validators
    "max_entries": 7,
    "historical_entries": 10000,
    "bond_denom": "ashm",
    "min_commission_rate": "0.000000000000000000"  // 0% minimum commission
  }
}
```

## 🎯 Voting Power and Reward Distribution

### Current System (Proportional Distribution)

**Rewards are distributed proportionally to voting power (stake amount):**

1. **Validator Share**: Proportional to total delegated stake
2. **Commission Structure**: Validators earn commission on delegator rewards
3. **Self-Delegation**: Validators earn full rewards on their own stake
4. **Delegator Share**: Remaining rewards after commission

### Voting Power Impact Formula

```javascript
validatorTotalRewards = (validatorStake / totalStaked) × totalRewards
validatorCommission = validatorTotalRewards × commissionRate
delegatorRewards = validatorTotalRewards - validatorCommission
```

## 🌐 Cosmos Ecosystem Comparison

### Standard Practices Across Major Chains

| Chain | Inflation Range | APR Range | Distribution Model | Community Tax |
|-------|----------------|-----------|-------------------|---------------|
| **Cosmos Hub** | 7-20% | 13.7% | Proportional | 5% |
| **Osmosis** | 9.83% | 1.45-3.78% | Proportional | 5% |
| **Juno** | 1-40% (decreasing) | 21-26% | Proportional | 2% |
| **Evmos** | Reduced in v16.0.0 | 5.91-10.93% | Proportional | 2% |
| **Injective** | 10-13.23% | 11.64-15% | Proportional | 5% |

### Key Findings

- **No Equal Distribution**: No major Cosmos chain implements equal validator rewards
- **Proportional Standard**: All use stake-weighted reward distribution
- **Inflation Evolution**: Most chains reduce inflation over time
- **Community Pool**: 2-5% allocation is standard

### Implementation Options for Equal Rewards

#### Option 1: Custom Distribution Logic

**File**: `precompiles/distribution/distribution.go`

```go
// Modify the distribution precompile to implement equal rewards
func (p Precompile) DistributeEqualRewards(ctx sdk.Context, rewards sdk.Coins) error {
    validators := p.stakingKeeper.GetAllValidators(ctx)
    activeValidators := 0

    // Count active validators
    for _, validator := range validators {
        if validator.IsBonded() {
            activeValidators++
        }
    }

    // Distribute equally
    rewardPerValidator := rewards.QuoInt(sdk.NewInt(int64(activeValidators)))

    for _, validator := range validators {
        if validator.IsBonded() {
            // Distribute equal rewards regardless of stake
            p.distributionKeeper.AllocateTokensToValidator(ctx, validator, rewardPerValidator)
        }
    }

    return nil
}
```

#### Option 2: Community Pool Redistribution

**Governance Proposal Example**:

```json
{
  "@type": "/cosmos.distribution.v1beta1.MsgUpdateParams",
  "authority": "shardeum10d07y265gmmuvt4z0w9aw880jnsr700jzj92zh",
  "params": {
    "community_tax": "0.980000000000000000",  // 98% to community pool
    "base_proposer_reward": "0.000000000000000000",
    "bonus_proposer_reward": "0.000000000000000000",
    "withdraw_addr_enabled": true
  }
}
```

Then implement custom logic to redistribute community pool funds equally.

## 📈 Reward Rate Configuration Strategies

### Increasing Overall Rewards

```json
// Higher inflation bounds
"inflation_max": "0.300000000000000000",  // 30% max (vs 20%)
"inflation_min": "0.100000000000000000",  // 10% min (vs 7%)
"inflation_rate_change": "0.200000000000000000"  // Faster adjustment
```

### Decreasing Rewards for Sustainability

```json
// Lower, sustainable inflation
"inflation_max": "0.100000000000000000",  // 10% max
"inflation_min": "0.030000000000000000",  // 3% min
"goal_bonded": "0.800000000000000000"     // Higher staking target
```

## 🎛️ Parameter Adjustment Methods

### 1. Genesis Configuration

**File**: `shardeumd/genesis.go`

```go
func NewMintGenesisState() *minttypes.GenesisState {
    mintGenState := minttypes.DefaultGenesisState()
    mintGenState.Params.MintDenom = config.ShardeumChainDenom

    // Custom parameters
    mintGenState.Params.InflationMax = sdk.MustNewDecFromStr("0.25")      // 25%
    mintGenState.Params.InflationMin = sdk.MustNewDecFromStr("0.05")      // 5%
    mintGenState.Params.GoalBonded = sdk.MustNewDecFromStr("0.75")        // 75%

    return mintGenState
}
```

### 2. Governance Proposals

```bash
# Submit parameter change proposal
shardeumd tx gov submit-proposal param-change proposal.json \
  --from validator \
  --chain-id shardeum_8082-1 \
  --gas auto \
  --gas-prices 0.025ashm
```

## 📚 Additional Resources

- **Cosmos SDK Documentation**: [https://docs.cosmos.network](https://docs.cosmos.network)
- **Distribution Module**: [x/distribution](https://docs.cosmos.network/main/modules/distribution)
- **Mint Module**: [x/mint](https://docs.cosmos.network/main/modules/mint)
- **Governance**: [x/gov](https://docs.cosmos.network/main/modules/gov)

## 💻 Cosmos Rewards Calculator App Analysis

The codebase includes a comprehensive educational tool at `cosmos-rewards-app/` that provides interactive visualization and code references for understanding the reward system.

### 🏗️ Application Structure

#### Core Components
- **Server**: Node.js/Express app (`server.js`) serving 3 main pages
- **Calculator**: Interactive parameter adjustment tool (`views/calculator.ejs`) [ Work in Progress ]
- **Code References**: Detailed Cosmos SDK implementation (`views/code-references.ejs`)
- **Overview**: Educational content explaining concepts (`views/overview.ejs`)

#### JavaScript Implementation (`public/js/calculator.js`)

The calculator implements the exact Cosmos SDK formulas in JavaScript:

```javascript
// Core calculation function matching Cosmos SDK NextInflationRate
function calculateCurrentInflation(params) {
    let currentInflation = params.initialInflation;

    if (params.currentBonded < params.goalBonded) {
        // Under-staked: inflation increases
        const gap = params.goalBonded - params.currentBonded;
        const perBlockAdjustment = (params.inflationRateChange * gap) / params.blocksPerYear;
        currentInflation = Math.min(currentInflation + perBlockAdjustment, params.inflationMax);
    } else if (params.currentBonded > params.goalBonded) {
        // Over-staked: inflation decreases
        const gap = params.currentBonded - params.goalBonded;
        const perBlockAdjustment = (params.inflationRateChange * gap) / params.blocksPerYear;
        currentInflation = Math.max(currentInflation - perBlockAdjustment, params.inflationMin);
    }

    return currentInflation;
}
```

### 🔧 Important Parameters for Reward Control

Based on the calculator analysis, these parameters directly control reward rates:

#### Primary Inflation Control
1. **`inflation_rate_change`** (13% default)
   - Controls speed of inflation adjustment
   - Higher = faster response to staking changes
   - Formula impact: `(inflationRateChange × gap) ÷ blocksPerYear`

2. **`inflation_max`** (20% default)
   - Maximum inflation rate when under-staked
   - Directly caps maximum APR available to attract stakers

3. **`inflation_min`** (7% default)
   - Minimum inflation rate when over-staked
   - Sets baseline reward level

4. **`goal_bonded`** (67% default)
   - Target staking ratio for equilibrium
   - Determines when inflation increases vs decreases

#### Secondary Parameters
5. **`blocks_per_year`** (6,311,520 default)
   - Defines precision of per-block adjustments
   - Affects speed of inflation changes

6. **`community_tax`** (2% default)
   - Percentage of rewards going to community pool
   - Directly reduces validator/delegator rewards

### 📊 Real-Time Calculation Process

The app demonstrates the exact Cosmos SDK execution flow:

#### Every Block (~5 seconds):
1. **BeginBlocker** executes in mint module
2. **BondedRatio** calculated: `stakedTokens ÷ totalSupply`
3. **NextInflationRate** adjusts inflation:
   ```javascript
   perBlockChange = (inflationRateChange × gap) ÷ blocksPerYear
   newInflation = currentInflation ± perBlockChange
   ```
4. **AnnualProvisions** calculated: `inflation × totalSupply`
5. **BlockProvision** minted: `annualProvisions ÷ blocksPerYear`

#### Distribution Flow:
```javascript
totalMinted = currentInflation × totalSupply
communityPool = totalMinted × communityTax  // 2%
validatorRewards = totalMinted - communityPool  // 98%
```

### 🎯 Interactive Features

#### Parameter Calculator
- Real-time adjustment of all mint parameters [ Work in Progress, complicated since reward calculation is dependent on realtime bonded stake percentage, need to provide continuous values]
- Immediate calculation of resulting APR and rewards
- Visual feedback showing inflation direction and magnitude

#### Timeline Projections
The calculator shows inflation changes over time:
- **Block 1**: Immediate effect of parameter changes
- **Day 1**: 17,280 blocks later (`17280 × perBlockChange`)
- **Week 1**: 120,960 blocks later
- **Month 1**: 525,960 blocks later
- **Year 1**: Full year projection (theoretical maximum)

#### Preset Scenarios
1. **Under-staked** (40% bonded): Shows high inflation response
2. **Target** (67% bonded): Shows stable equilibrium
3. **Over-staked** (85% bonded): Shows decreasing inflation

### 🧮 Mathematical Formulas (from Code References)

#### Core Cosmos SDK Implementation

**File**: `cosmos-sdk/x/mint/types/minter.go`
```go
// NextInflationRate returns the new inflation rate for the next block
func (m Minter) NextInflationRate(params Params, bondedRatio math.LegacyDec) math.LegacyDec {
    if bondedRatio.LT(bondedRatioTarget) {
        // Under-staked: increase inflation
        inflationRate := m.Inflation.Add(
            inflationRateChangePerYear.Mul(
                bondedRatioTarget.Sub(bondedRatio)
            ).Quo(math.LegacyNewDec(int64(params.BlocksPerYear)))
        )

        if inflationRate.GT(inflationMax) {
            inflationRate = inflationMax
        }
        return inflationRate
    } else {
        // Over-staked: decrease inflation
        inflationRate := m.Inflation.Sub(
            inflationRateChangePerYear.Mul(
                bondedRatio.Sub(bondedRatioTarget)
            ).Quo(math.LegacyNewDec(int64(params.BlocksPerYear)))
        )

        if inflationRate.LT(inflationMin) {
            inflationRate = inflationMin
        }
        return inflationRate
    }
}
```

#### APR Calculations
```javascript
// Base APR calculation
totalStaked = totalSupply × currentBonded
baseAPR = validatorRewards ÷ totalStaked
delegatorAPR = baseAPR × (1 - validatorCommission)
validatorAPR = baseAPR  // Full rate on own stake + commission from delegators
```

### 🔄 Execution Flow Analysis

#### BeginBlocker Order (`shardeumd/app.go:629-630`)
```go
app.ModuleManager.SetOrderBeginBlockers(
    minttypes.ModuleName,  // Runs FIRST every block
    // ... other modules
)
```

#### Per-Block Process (`cosmos-sdk/x/mint/keeper/abci.go`)
```go
func (k Keeper) BeginBlocker(ctx context.Context) error {
    // 1. Get current bonding ratio
    bondedRatio, err := k.BondedRatio(ctx)

    // 2. Get current minter state
    minter, err := k.Minter.Get(ctx)
    params, err := k.Params.Get(ctx)

    // 3. ADJUST INFLATION EVERY BLOCK
    minter.Inflation = minter.NextInflationRate(params, bondedRatio)
    minter.AnnualProvisions = minter.NextAnnualProvisions(params, k.TokenSupply(ctx))

    // 4. Store updated state
    k.Minter.Set(ctx, minter)

    // 5. Mint tokens for this block
    blockProvision := minter.BlockProvision(params)
    k.MintCoins(ctx, sdk.NewCoins(blockProvision))
    k.AddCollectedFees(ctx, sdk.NewCoins(blockProvision))
}
```

### 🎛️ Parameter Impact Matrix

| Parameter | Current Value | Increase Effect | Decrease Effect | Reward Impact |
|-----------|---------------|-----------------|-----------------|---------------|
| `inflation_rate_change` | 13% | Faster adjustment | Slower adjustment | Response speed |
| `inflation_max` | 20% | Higher max rewards | Lower max rewards | Peak APR |
| `inflation_min` | 7% | Higher base rewards | Lower base rewards | Minimum APR |
| `goal_bonded` | 67% | More staking target | Less staking target | Equilibrium point |
| `blocks_per_year` | 6,311,520 | Slower adjustments | Faster adjustments | Precision |
| `community_tax` | 2% | Less validator rewards | More validator rewards | Net APR |

### 🚀 Running the Calculator

```bash
cd cosmos-rewards-app/
npm install
npm start
# Open http://localhost:3000
```

*Analysis based on Cosmos SDK v0.53.4 implementation and cosmos-rewards-app calculator*