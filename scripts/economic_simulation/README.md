# Economic Simulation Tools

Comprehensive tools for simulating and analyzing economic parameters in the Shardeum Cosmos network.

## Overview

This simulation suite evaluates the impact of different mint module parameters (min/max inflation, bonding ratio targets) on network economics including rewards, APY, and token minting.

**Key Features:**
- **Interactive menu system** for non-technical users
- Fast offline simulation (9,400+ runs in seconds)
- Network validation against actual chain behavior
- Comprehensive visualizations and reporting
- Customizable parameter ranges
- HTML reports with interactive analysis

## Quick Start (Interactive Mode - RECOMMENDED)

### For Non-Technical Users

The easiest way to run simulations - **no command-line knowledge needed**:

```bash
cd /home/marc/evm/shardeum-evm
./scripts/economic_simulation/run_interactive.sh
```

**What happens:**
1. **Friendly menu** appears with preset options
2. **Select a mode** (Quick Test / Standard / Comprehensive / Current Settings)
3. **Confirm** your choice
4. **Wait** for simulation to complete
5. **Open** the HTML report in your browser - Done!

**Recommended for most users:** Select option `2) Standard Analysis`

### Interactive Mode Features

- ✅ **Plain language explanations** - no technical jargon
- ✅ **Preset configurations** - optimized for common use cases
- ✅ **Parameter validation** - prevents invalid inputs
- ✅ **Progress indicators** - see simulation status in real-time
- ✅ **Built-in help** - explains what each parameter means
- ✅ **Custom mode** - for advanced users who want full control

### Available Presets

**1) Quick Test** (~5 minutes)
- Fast test with ~125 scenarios
- Good for verifying setup
- Basic parameter exploration

**2) Standard Analysis** (~15 minutes) **[RECOMMENDED]**
- Comprehensive with ~1,000 scenarios
- Thorough parameter exploration
- Full HTML report with visualizations

**3) Comprehensive Study** (~45 minutes)
- Extensive with ~3,000+ scenarios
- Includes network validation
- Maximum accuracy

**4) Current Network Settings** (~2 minutes)
- Tests only current network parameters (min=7%, max=20%, goal=67%)
- Varies bonding ratios from 30% to 95%
- Quick validation of current settings

**5) Custom Parameters** (Advanced)
- Manually configure all parameters
- Full control over ranges and detail level

## Quick Start (Command-Line Mode)

### For Technical Users

Direct command-line execution without interactive menu:

```bash
cd /home/marc/evm/shardeum-evm
./scripts/economic_simulation/run_full_simulation.sh
```

This will:
1. Run parameter sweep with default ranges
2. Generate visualizations
3. Create HTML report

**Output:** `simulation_results/reports/economic_report_*.html`

### Custom Parameter Ranges

```bash
./scripts/economic_simulation/run_full_simulation.sh \
  --min-inflation-min 0.05 \
  --min-inflation-max 0.12 \
  --max-inflation-min 0.15 \
  --max-inflation-max 0.30 \
  --goal-bonded-min 0.50 \
  --goal-bonded-max 0.80 \
  --num-points 8
```

### With Network Validation

Run simulation and validate against actual networks:

```bash
./scripts/economic_simulation/run_full_simulation.sh --validate --num-validations 15
```

**Note:** Network validation requires launching test networks and is much slower.

## Individual Tools

### 1. Economic Simulator (`economic_simulator.py`)

Core simulation logic implementing Cosmos SDK mint module algorithm.

**Usage:**
```bash
python3 scripts/economic_simulation/economic_simulator.py
```

**Demo output:**
```
Bonded Ratio: 50%
  Current Inflation: 15.21%
  Annual Provisions: 9,126,000,000 SHM
  Validator APY: 30.42%
```

**Python API:**
```python
from economic_simulator import run_single_simulation

result = run_single_simulation(
    total_supply_shm=60_000_000_000,
    bonded_ratio=0.67,
    min_inflation=0.07,
    max_inflation=0.20,
    goal_bonded=0.67
)

print(f"APY: {result.validator_apy:.2%}")
print(f"New SHM/year: {result.annual_provisions_shm:,.0f}")
```

### 2. Parameter Sweep (`run_simulation.py`)

Generates and runs all parameter combinations.

**Usage:**
```bash
python3 scripts/economic_simulation/run_simulation.py \
  --min-inflation-min 0.03 \
  --min-inflation-max 0.15 \
  --max-inflation-min 0.10 \
  --max-inflation-max 0.40 \
  --goal-bonded-min 0.40 \
  --goal-bonded-max 0.90 \
  --num-points 10 \
  --num-bonding-ratios 10 \
  --output-name my_sweep
```

**Parameters:**
- `--num-points`: Number of values to sample per parameter (default: 10)
- `--num-bonding-ratios`: Number of bonding ratios to test (default: 10)
- `--output-dir`: Output directory (default: simulation_results)

**Output:**
- CSV: `simulation_results/my_sweep_*.csv`
- JSON: `simulation_results/my_sweep_*.json`
- Metadata: `simulation_results/my_sweep_*_metadata.json`

### 3. Genesis Generator (`generate_test_genesis.py`)

Creates custom genesis files with modified mint parameters.

**Usage:**
```bash
python3 scripts/economic_simulation/generate_test_genesis.py \
  --min-inflation 0.05 \
  --max-inflation 0.15 \
  --goal-bonded 0.70 \
  --output simulation_configs/test-genesis.json
```

**Launch network with custom genesis:**
```bash
./local_node.sh -y -g simulation_configs/test-genesis.json
```

### 4. Network Validator (`network_validator.sh`)

Launches test network and queries mint endpoints.

**Usage:**
```bash
./scripts/economic_simulation/network_validator.sh \
  -g simulation_configs/test-genesis.json \
  -o validation_results.json \
  -v
```

**Queries:**
- Mint parameters
- Annual provisions
- Current inflation

**Note:** Automatically starts and stops test network.

### 5. Validation Runner (`validate_simulation.py`)

Validates simulated results against actual networks.

**Usage:**
```bash
python3 scripts/economic_simulation/validate_simulation.py \
  --input simulation_results/full_sweep_*.csv \
  --num-samples 15 \
  --strategy diverse
```

**Strategies:**
- `diverse`: Corners + center + random samples (recommended)
- `corners`: Only corner cases of parameter space
- `random`: Purely random selection

**Output:** Comparison of simulated vs actual results with error percentages.

### 6. Report Generator (`generate_report.py`)

Creates visualizations and HTML report.

**Usage:**
```bash
python3 scripts/economic_simulation/generate_report.py \
  --input simulation_results/full_sweep_*.csv \
  --metadata simulation_results/full_sweep_*_metadata.json
```

**Requires:** `matplotlib`, `numpy`
```bash
pip install matplotlib numpy
```

**Output:**
- HTML report with summary statistics
- APY vs bonding ratio plots
- Annual provisions comparison
- Interactive analysis

## Understanding the Results

### Key Metrics

**Validator APY:**
- Return on staked tokens for validators/delegators
- Higher when fewer tokens are staked
- Formula: `annual_provisions / bonded_tokens`

**Annual Provisions:**
- New tokens minted per year
- Formula: `total_supply * current_inflation`
- With 60B supply at 13% inflation: ~7.8B SHM/year

**Current Inflation:**
- Dynamically adjusts based on bonding ratio
- Increases if bonded < goal (incentivize staking)
- Decreases if bonded > goal (reduce rewards)

### Inflation Algorithm

The Cosmos SDK mint module adjusts inflation:

```
if bonded_ratio < goal_bonded:
    inflation += inflation_rate_change * (goal_bonded - bonded_ratio)
else:
    inflation -= inflation_rate_change * (bonded_ratio - goal_bonded)

inflation = clamp(inflation, min_inflation, max_inflation)
```

**Example with current params:**
- Goal: 67% bonded
- Min: 7%, Max: 20%
- Rate change: 13%

At 50% bonded (below goal):
- Inflation increases to ~15.21%
- Higher rewards to encourage staking

At 85% bonded (above goal):
- Inflation decreases to ~10.66%
- Lower rewards as target exceeded

## Parameter Recommendations

Based on simulation results:

### For Higher Staking Incentives
- Lower min_inflation: 3-5%
- Higher max_inflation: 25-40%
- Lower goal_bonded: 40-55%

**Effect:** Larger inflation adjustments, higher rewards when bonding is low

### For Stable Economics
- Moderate range: min 5-8%, max 15-22%
- Moderate goal: 60-70%
- Moderate rate change: 10-15%

**Effect:** Balanced approach, gradual adjustments

### For Minimal Inflation
- Higher min_inflation: 8-12%
- Lower max_inflation: 12-18%
- Higher goal_bonded: 75-85%

**Effect:** Low token emission, predictable supply growth

## File Structure

```
scripts/economic_simulation/
├── README.md                          # This file
├── economic_simulator.py              # Core simulation logic
├── run_simulation.py                  # Parameter sweep runner
├── generate_test_genesis.py           # Genesis file generator
├── network_validator.sh               # Network testing harness
├── validate_simulation.py             # Validation runner
├── generate_report.py                 # Visualization & reporting
└── run_full_simulation.sh             # Master orchestrator

simulation_results/                    # Simulation outputs
├── full_sweep_*.csv                   # Simulation data
├── full_sweep_*.json                  # JSON format
├── full_sweep_*_metadata.json         # Simulation metadata
├── reports/
│   ├── economic_report_*.html         # HTML report
│   ├── apy_vs_bonding_ratio.png       # Visualizations
│   └── annual_provisions.png
└── validation/
    └── validation_results_*.json      # Network validation results

simulation_configs/                    # Generated genesis files
└── test-genesis-*.json                # Custom genesis configs
```

## Current Network Parameters

From `config/environments/*-genesis.genesis.json`:

```json
{
  "mint_denom": "ashm",
  "inflation_min": "0.070000000000000000",     // 7%
  "inflation_max": "0.200000000000000000",     // 20%
  "inflation_rate_change": "0.130000000000000000",  // 13%
  "goal_bonded": "0.670000000000000000",       // 67%
  "blocks_per_year": "6311520"
}
```

**Initial Supply:** ~60 billion SHM

## Example Workflows

### Interactive Mode Example: Finding Optimal Parameters

**Scenario:** You want to find parameters that give ~15-20% validator APY

**Steps:**
1. Launch interactive mode:
   ```bash
   ./scripts/economic_simulation/run_interactive.sh
   ```

2. Select `2) Standard Analysis` (recommended)

3. Wait ~15 minutes for simulation to complete

4. Open the HTML report (path shown at end)

5. In the report, look for:
   - APY vs Bonding Ratio charts
   - Find the parameter combinations that show 15-20% APY at your target bonding ratio
   - Check the "Summary Statistics" section

6. If you want to test specific parameters:
   - Run interactive mode again
   - Select `5) Custom Parameters`
   - Enter narrower ranges based on what worked well

**That's it!** No command-line knowledge needed.

### Command-Line Example: Finding Optimal Parameters for 15% Target APY

1. **Run simulation with wide range:**
```bash
./scripts/economic_simulation/run_full_simulation.sh \
  --min-inflation-min 0.03 \
  --min-inflation-max 0.15 \
  --max-inflation-min 0.10 \
  --max-inflation-max 0.35 \
  --num-points 12
```

2. **Analyze report:**
Open `simulation_results/reports/economic_report_*.html`
Look for parameter combinations yielding 15% APY at target bonding ratio

3. **Validate top candidates:**
```bash
python3 scripts/economic_simulation/validate_simulation.py \
  --input simulation_results/full_sweep_*.csv \
  --num-samples 20 \
  --strategy diverse
```

4. **Test specific configuration:**
```bash
# Generate genesis with chosen parameters
python3 scripts/economic_simulation/generate_test_genesis.py \
  --min-inflation 0.06 \
  --max-inflation 0.18 \
  --goal-bonded 0.65

# Launch test network
./local_node.sh -y -g simulation_configs/test-genesis-min6_max18_goal65.json

# Query results
curl -s "http://localhost:1317/cosmos/mint/v1beta1/annual_provisions" | jq
```

5. **Iterate if needed**

## Troubleshooting

### Simulation is slow
- Reduce `--num-points` (e.g., 5-7 instead of 10)
- Reduce `--num-bonding-ratios` (e.g., 5 instead of 10)

### Visualization errors
```bash
pip install matplotlib numpy
```

### Network validation fails
- Check if shardeumd is already running: `pkill shardeumd`
- Verify genesis file is valid JSON
- Check port 1317 is not in use
- Use `--verbose` flag for detailed output

### "Permission denied" errors
```bash
chmod +x scripts/*.sh
chmod +x scripts/*.py
```

## Advanced Usage

### Custom Bonding Ratio Range

Test different actual bonding scenarios:

```bash
python3 scripts/economic_simulation/run_simulation.py \
  --bonding-ratio-min 0.20 \
  --bonding-ratio-max 0.95 \
  --num-bonding-ratios 15
```

### Programmatic Access

```python
from economic_simulator import CosmosEconomicSimulator, MintParams, NetworkState

# Create simulator
params = MintParams(
    mint_denom="ashm",
    inflation_rate_change=0.13,
    inflation_max=0.20,
    inflation_min=0.07,
    goal_bonded=0.67,
    blocks_per_year=6311520
)

simulator = CosmosEconomicSimulator(params)

# Run simulation
state = NetworkState(
    total_supply=60_000_000_000,
    bonded_tokens=40_000_000_000,
    current_inflation=0.13
)

result = simulator.simulate(state)
print(result.to_dict())
```

## Support

For issues or questions:
1. Check this README
2. Review simulation outputs and logs
3. Verify parameter ranges are valid (0.0-1.0)
4. Ensure dependencies are installed

## Next Steps

1. **Run your first simulation:**
   ```bash
   ./scripts/economic_simulation/run_full_simulation.sh
   ```

2. **Review the HTML report** to understand current parameter impacts

3. **Experiment with different ranges** to find optimal settings

4. **Validate** promising configurations with actual networks

5. **Document** your findings and parameter choices

---

**Generated:** 2025-10-24
**Version:** 1.0
**Shardeum Economic Simulation Suite**
