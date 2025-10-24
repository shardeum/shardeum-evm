# Economic Simulation - Simple User Guide

**For Non-Technical Team Members**

## What is This Tool?

This tool helps you understand how changing network economic parameters affects:
- **Validator rewards** (how much validators earn)
- **Inflation rate** (how fast new tokens are created)
- **Token supply growth** (how many new tokens per year)

## How to Run a Simulation (5 Steps)

### Step 1: Open Terminal

On your computer, open Terminal (Mac/Linux) or Git Bash (Windows)

### Step 2: Navigate to Project

```bash
cd /home/marc/evm/shardeum-evm
```

*(Replace the path with where your project is located)*

### Step 3: Run Interactive Mode

```bash
./scripts/economic_simulation/run_interactive.sh
```

### Step 4: Choose a Preset

You'll see a menu with options. **We recommend option 2**:

```
1) Quick Test            - 5 minutes, basic test
2) Standard Analysis     - 15 minutes, RECOMMENDED  ⭐
3) Comprehensive Study   - 45 minutes, very thorough
4) Current Settings      - 2 minutes, test current network
5) Custom Parameters     - Advanced users only
```

**Type `2` and press Enter**

### Step 5: Confirm and Wait

- You'll see a summary of what will run
- Type `y` and press Enter to confirm
- Wait for the simulation to complete (progress bar shows status)
- At the end, you'll see a path to an HTML report
- Open that file in your web browser to see results!

## Understanding the Results

When you open the HTML report, you'll see:

### Summary Statistics Table
- **Validator APY**: Shows the range of yearly returns for validators
  - Higher = more rewards for staking
  - Example: "12% to 35%" means APY varies depending on parameters

- **Inflation Rate**: Shows how fast new tokens are created
  - Higher = more new tokens per year
  - Current network is around 7-20%

- **Annual Provisions**: Total new tokens minted per year
  - Shows in billions of SHM
  - Example: "6.4B to 9.1B SHM per year"

### Charts and Graphs

1. **APY vs Bonding Ratio**
   - Shows how validator rewards change with different staking levels
   - Look for the curves that match your target APY

2. **Annual Provisions**
   - Shows how many new tokens are created under different scenarios
   - Helps understand token supply growth

## Common Questions

### Q: Which preset should I use?

**A:** Start with "Standard Analysis" (option 2). It's the best balance of speed and thoroughness.

### Q: How long does it take?

**A:**
- Quick Test: ~5 minutes
- Standard Analysis: ~15 minutes (recommended)
- Comprehensive: ~45 minutes

### Q: What if I want to test specific values?

**A:** Use option 5 "Custom Parameters". The tool will guide you through entering specific inflation rates and bonding targets you want to test.

### Q: Do I need to understand the technical details?

**A:** No! The interactive mode asks questions in plain language. You can answer based on business goals (e.g., "I want 15-20% APY for validators").

### Q: Can I run this multiple times?

**A:** Yes! Run as many times as you want. Each run creates new result files.

### Q: What if something goes wrong?

**A:**
1. Check that you have Python 3 installed: `python3 --version`
2. Check that you have jq installed: `jq --version`
3. If issues persist, ask a technical team member for help

## Example: Finding Parameters for 18% Target APY

**Goal:** You want validators to earn about 18% APY

**Steps:**

1. Run the interactive tool
   ```bash
   ./scripts/economic_simulation/run_interactive.sh
   ```

2. Select `2) Standard Analysis`

3. Wait ~15 minutes

4. Open the HTML report (path shown at the end)

5. In the report:
   - Look at the "APY vs Bonding Ratio" chart
   - Find the lines that show ~18% APY at 67% bonding (current target)
   - Note which parameter combinations achieve this

6. If you want to narrow it down:
   - Run again, select `5) Custom Parameters`
   - Enter ranges close to what worked
   - Get more precise results

## Quick Reference

### To run a simulation:
```bash
cd /home/marc/evm/shardeum-evm
./scripts/economic_simulation/run_interactive.sh
```

### To see the help in the tool:
- Select option `6) Help` from the main menu

### Results are saved in:
```
simulation_results/
  reports/
    economic_report_*.html  ← Open this in your browser
```

## Tips for Non-Technical Users

✅ **DO:**
- Start with Standard Analysis
- Take notes on which parameter combinations look promising
- Run multiple times to explore different scenarios
- Share the HTML report with team members (it's self-contained)

❌ **DON'T:**
- Worry about command-line syntax (the menu handles it)
- Edit any Python or shell script files
- Delete files in simulation_results (they're your historical data)
- Use "Comprehensive Study" unless you really need network validation

## Getting Help

If you need assistance:

1. **Check the built-in help**: Select option 6 in the interactive menu

2. **Read the full documentation**: `scripts/economic_simulation/README.md`

3. **Ask a technical team member** to help interpret results

4. **Common issues**:
   - "command not found" = You're not in the right directory (use `cd` command)
   - "python3: not found" = Need to install Python 3
   - Simulation stuck = Press Ctrl+C to cancel and try again

---

**Version:** 1.0
**Last Updated:** 2025-10-24
**For:** Shardeum Economic Parameter Analysis
