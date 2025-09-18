# 🌌 Cosmos Rewards Calculator

An interactive educational tool for understanding how inflation and rewards work in Cosmos SDK blockchains, specifically designed for the Shardeum Cosmos chain.

## 🚀 Quick Start

```bash
# Install dependencies
npm install

# Start the application
npm start
```

The application will automatically open in your browser at `http://localhost:3000`.

## 📚 What You'll Learn

### 🎯 Core Concepts
- How validator and delegator rewards are calculated
- The self-regulating inflation mechanism
- Economic incentives that maintain network security
- Real-time parameter impact on APR and rewards

### 🧮 Interactive Features
- **Parameter Calculator**: Adjust inflation settings and see immediate results
- **Timeline Projections**: See block-by-block inflation changes over time
- **Show Calculations**: Detailed mathematical breakdowns for every formula
- **Code References**: Actual Cosmos SDK implementation with line-by-line explanations

### 💻 Technical Implementation
- Built with Node.js, Express, and Bootstrap
- Real-time calculations using your Shardeum genesis parameters
- Interactive charts and visualizations
- Responsive design with custom color scheme (#3042FB, #FCFAEF)

## 📖 Application Structure

### Page 1: Educational Overview
- Complete explanation of how Cosmos rewards work
- Validator vs delegator roles and earnings
- Self-correcting economic mechanism
- Foundation steering strategies
- Real examples using Shardeum parameters

### Page 2: Interactive Calculator
- **Parameter Legend**: Always-visible explanations of each variable
- **Real-time Calculations**:
  - Inflation rate adjustments per block
  - Annual and block provisions
  - APR calculations for validators and delegators
  - Timeline projections showing inflation changes
- **Show Calculations Buttons**: Expandable sections with step-by-step math
- **Preset Scenarios**: Quick-load common staking situations
- **Visual Charts**: Interactive inflation vs bonding ratio graphs

### Page 3: Code References
- Actual Cosmos SDK source code with explanations
- BeginBlocker execution flow
- NextInflationRate implementation
- Mathematical formulas with real examples
- Key insights from our analysis
- Parameter impact summary table

## 🎛️ Key Parameters Explained

Based on your Shardeum genesis configuration:

| Parameter | Value | Description |
|-----------|-------|-------------|
| `inflation_rate_change` | 13% | Speed of inflation adjustment (fast response) |
| `inflation_max` | 20% | Maximum inflation when under-staked |
| `inflation_min` | 7% | Minimum inflation when over-staked |
| `goal_bonded` | 67% | Target staking ratio for equilibrium |
| `blocks_per_year` | 6,311,520 | ~5 second blocks for precise calculations |
| `community_tax` | 2% | Percentage going to governance treasury |

## 🔬 How It Works

### Every Block (~5 seconds):
1. **BeginBlocker** runs in the mint module
2. **Bonding ratio** is calculated (staked tokens / total supply)
3. **Inflation rate** adjusts based on distance from 67% target
4. **New tokens** are minted and distributed to validators/delegators

### Self-Correcting Mechanism:
- **Under-staked** (&lt;67%) → Inflation increases → Higher APR → Attracts stakers
- **Target** (67%) → Stable inflation → Balanced security & liquidity
- **Over-staked** (&gt;67%) → Inflation decreases → Lower APR → Encourages unstaking

## 💡 Educational Value

This tool helps you understand:
- Why inflation changes dynamically in Cosmos chains
- How economic incentives maintain network security
- The precise mathematical formulas behind reward calculations
- Real-world impact of parameter adjustments
- Foundation strategies for network governance

## 🛠️ Development

```bash
# Development mode with auto-restart
npm run dev

# Production mode
npm start
```

## 📊 Example Scenarios

### Under-staked Network (40% bonded):
- Inflation increases toward 20% maximum
- APR rises to ~30% to attract more stakers
- Network security improves as more tokens stake

### Target Staking (67% bonded):
- Inflation stable around 13%
- Balanced security and token liquidity
- APR around 19% maintains equilibrium

### Over-staked Network (85% bonded):
- Inflation decreases toward 7% minimum
- Lower APR encourages some unstaking
- More tokens available for DeFi and liquidity

## 🎯 Foundation Steering

Learn how chain foundations can influence rewards:
- **Governance proposals** to adjust inflation bounds
- **Target modifications** to change equilibrium points
- **Community tax adjustments** for development funding
- **Response speed tuning** for market stability

## 🔗 Cosmos SDK Integration

All calculations are based on the actual Cosmos SDK v0.53.4 implementation used in your Shardeum chain, with references to:
- `x/mint/keeper/abci.go` - BeginBlocker execution
- `x/mint/types/minter.go` - Inflation calculations
- `shardeumd/app.go` - Module execution order

---

**Built for the Shardeum Cosmos ecosystem** 🌟

Understanding blockchain economics has never been this interactive!