#!/bin/bash
#
# Interactive Launcher for Economic Simulation
#
# User-friendly interface that guides non-technical users through
# configuring and running economic parameter simulations.
#

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CONFIG_FILE="/tmp/economic_sim_config.json"

# Check dependencies
check_dependencies() {
    local missing_deps=false

    if ! command -v python3 &> /dev/null; then
        echo -e "${RED}✗ Error: python3 not found${NC}"
        missing_deps=true
    fi

    if ! command -v jq &> /dev/null; then
        echo -e "${RED}✗ Error: jq not found (needed for JSON parsing)${NC}"
        echo -e "${YELLOW}  Install with: sudo apt-get install jq${NC}"
        missing_deps=true
    fi

    if [[ "$missing_deps" == "true" ]]; then
        echo
        echo -e "${RED}Please install missing dependencies and try again.${NC}"
        exit 1
    fi
}

# Parse configuration from JSON
read_config() {
    if [[ ! -f "$CONFIG_FILE" ]]; then
        echo -e "${RED}✗ Error: Configuration file not found${NC}"
        exit 1
    fi

    # Read values from JSON
    MIN_INFLATION_MIN=$(jq -r '.min_inflation_min' "$CONFIG_FILE")
    MIN_INFLATION_MAX=$(jq -r '.min_inflation_max' "$CONFIG_FILE")
    MAX_INFLATION_MIN=$(jq -r '.max_inflation_min' "$CONFIG_FILE")
    MAX_INFLATION_MAX=$(jq -r '.max_inflation_max' "$CONFIG_FILE")
    GOAL_BONDED_MIN=$(jq -r '.goal_bonded_min' "$CONFIG_FILE")
    GOAL_BONDED_MAX=$(jq -r '.goal_bonded_max' "$CONFIG_FILE")
    NUM_POINTS=$(jq -r '.num_points' "$CONFIG_FILE")
    NUM_BONDING=$(jq -r '.num_bonding_ratios' "$CONFIG_FILE")
    BONDING_RATIO_MIN=$(jq -r '.bonding_ratio_min // 0.05' "$CONFIG_FILE")
    BONDING_RATIO_MAX=$(jq -r '.bonding_ratio_max // 0.95' "$CONFIG_FILE")
    VALIDATE=$(jq -r '.validate' "$CONFIG_FILE")
    NUM_VALIDATIONS=$(jq -r '.num_validations // 0' "$CONFIG_FILE")
    CONFIG_NAME=$(jq -r '.name' "$CONFIG_FILE")
}

# Show progress bar
show_progress() {
    local current=$1
    local total=$2
    local width=50
    local percentage=$((current * 100 / total))
    local filled=$((width * current / total))
    local empty=$((width - filled))

    printf "\r["
    printf "%${filled}s" | tr ' ' '█'
    printf "%${empty}s" | tr ' ' '░'
    printf "] %3d%% (%d/%d)" "$percentage" "$current" "$total"
}

# Monitor simulation progress
monitor_progress() {
    local csv_pattern=$1
    local expected_total=$2

    # Wait for CSV file to be created
    local csv_file=""
    local wait_count=0
    while [[ -z "$csv_file" && $wait_count -lt 30 ]]; do
        csv_file=$(ls -t simulation_results/${csv_pattern}*.csv 2>/dev/null | head -n1 || true)
        if [[ -z "$csv_file" ]]; then
            sleep 1
            ((wait_count++))
        fi
    done

    if [[ -z "$csv_file" ]]; then
        # Couldn't find file, just show spinner
        return
    fi

    # Monitor file line count
    local last_count=0
    while true; do
        if [[ -f "$csv_file" ]]; then
            local current_count=$(wc -l < "$csv_file" 2>/dev/null || echo "1")
            # Subtract header line
            current_count=$((current_count - 1))

            if [[ $current_count -gt $last_count ]]; then
                show_progress "$current_count" "$expected_total"
                last_count=$current_count
            fi

            # Check if complete
            if [[ $current_count -ge $expected_total ]]; then
                show_progress "$expected_total" "$expected_total"
                echo
                break
            fi
        fi
        sleep 2
    done
}

# Run simulation with configured parameters
run_simulation() {
    cd "$REPO_ROOT"

    echo
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}  Running: $CONFIG_NAME${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo

    # Calculate expected total simulations
    if [[ "$NUM_POINTS" -gt 1 ]]; then
        # For ranges, we test combinations
        local param_combos=$((NUM_POINTS * NUM_POINTS * NUM_POINTS))
    else
        # For single values (current network), just 1 combo
        local param_combos=1
    fi
    local expected_total=$((param_combos * NUM_BONDING))

    # Build simulation command
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local output_name="simulation_${timestamp}"

    local sim_args=(
        --min-inflation-min "$MIN_INFLATION_MIN"
        --min-inflation-max "$MIN_INFLATION_MAX"
        --max-inflation-min "$MAX_INFLATION_MIN"
        --max-inflation-max "$MAX_INFLATION_MAX"
        --goal-bonded-min "$GOAL_BONDED_MIN"
        --goal-bonded-max "$GOAL_BONDED_MAX"
        --num-points "$NUM_POINTS"
        --num-bonding-ratios "$NUM_BONDING"
        --bonding-ratio-min "$BONDING_RATIO_MIN"
        --bonding-ratio-max "$BONDING_RATIO_MAX"
        --output-name "$output_name"
    )

    # Step 1: Run parameter sweep
    echo -e "${GREEN}Step 1/3: Running Economic Simulations...${NC}"
    echo -e "${YELLOW}Testing ~$expected_total scenarios...${NC}"
    echo

    # Run in background so we can monitor progress
    python3 "$SCRIPT_DIR/run_simulation.py" "${sim_args[@]}" > /tmp/sim_output.log 2>&1 &
    local sim_pid=$!

    # Monitor progress
    monitor_progress "$output_name" "$expected_total" &
    local monitor_pid=$!

    # Wait for simulation to complete
    wait $sim_pid
    local sim_exit=$?

    # Stop monitor
    kill $monitor_pid 2>/dev/null || true
    wait $monitor_pid 2>/dev/null || true

    if [[ $sim_exit -ne 0 ]]; then
        echo
        echo -e "${RED}✗ Simulation failed${NC}"
        echo -e "${YELLOW}Check /tmp/sim_output.log for details${NC}"
        exit 1
    fi

    echo
    echo -e "${GREEN}✓ Simulations complete!${NC}"
    echo

    # Find output files
    LATEST_CSV=$(ls -t simulation_results/${output_name}*.csv 2>/dev/null | head -n1)
    LATEST_METADATA=$(ls -t simulation_results/${output_name}*_metadata.json 2>/dev/null | head -n1)

    # Step 2: Network validation (if enabled)
    if [[ "$VALIDATE" == "true" ]]; then
        echo -e "${GREEN}Step 2/3: Running Network Validation...${NC}"
        echo -e "${YELLOW}This will take a while as it launches test networks...${NC}"
        echo

        python3 "$SCRIPT_DIR/validate_simulation.py" \
            --input "$LATEST_CSV" \
            --num-samples "$NUM_VALIDATIONS" \
            --output-dir simulation_results/validation \
            --verbose

        if [[ $? -eq 0 ]]; then
            echo -e "${GREEN}✓ Validation complete!${NC}"
        else
            echo -e "${YELLOW}⚠ Validation had issues (simulation results still valid)${NC}"
        fi
        echo
    else
        echo -e "${BLUE}Step 2/3: Skipped network validation${NC}"
        echo
    fi

    # Step 3: Generate report
    echo -e "${GREEN}Step 3/3: Generating Report...${NC}"

    local report_args=(
        --input "$LATEST_CSV"
        --output-dir simulation_results/reports
    )

    if [[ -n "$LATEST_METADATA" ]]; then
        report_args+=(--metadata "$LATEST_METADATA")
    fi

    python3 "$SCRIPT_DIR/generate_report.py" "${report_args[@]}"

    if [[ $? -eq 0 ]]; then
        echo -e "${GREEN}✓ Report generation complete!${NC}"
    else
        echo -e "${YELLOW}⚠ Report generation had issues${NC}"
    fi

    # Find report
    LATEST_REPORT=$(ls -t simulation_results/reports/economic_report_*.html 2>/dev/null | head -n1)

    # Show summary
    show_summary
}

# Show final summary
show_summary() {
    echo
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}${BOLD}  ✓ SIMULATION COMPLETE${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo

    if [[ -n "$LATEST_CSV" ]]; then
        # Extract some quick stats from CSV
        local line_count=$(wc -l < "$LATEST_CSV")
        local scenario_count=$((line_count - 1))  # Minus header

        echo -e "${BOLD}Results Summary:${NC}"
        echo -e "  • Scenarios tested: ${GREEN}$scenario_count${NC}"

        # Try to get min/max APY
        if command -v awk &> /dev/null; then
            local min_apy=$(awk -F',' 'NR>1 {print $10}' "$LATEST_CSV" | sort -n | head -1)
            local max_apy=$(awk -F',' 'NR>1 {print $10}' "$LATEST_CSV" | sort -n | tail -1)

            if [[ -n "$min_apy" && -n "$max_apy" ]]; then
                min_apy=$(printf "%.1f" $(echo "$min_apy * 100" | bc -l))
                max_apy=$(printf "%.1f" $(echo "$max_apy * 100" | bc -l))
                echo -e "  • APY range: ${YELLOW}${min_apy}% to ${max_apy}%${NC}"
            fi
        fi

        echo
        echo -e "${BOLD}Output Files:${NC}"
        echo -e "  📊 Data: ${BLUE}$LATEST_CSV${NC}"
    fi

    if [[ -n "$LATEST_REPORT" ]]; then
        echo -e "  📈 Report: ${BLUE}$LATEST_REPORT${NC}"
        echo
        echo -e "${GREEN}${BOLD}Open the report in your browser to see all visualizations!${NC}"
        echo -e "${BLUE}file://$(realpath "$LATEST_REPORT")${NC}"
    fi

    echo
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Main execution
main() {
    # Check dependencies
    check_dependencies

    # Run interactive menu
    python3 "$SCRIPT_DIR/interactive_menu.py"

    # Check if user cancelled
    if [[ $? -ne 0 ]]; then
        echo
        echo "Cancelled by user."
        exit 0
    fi

    # Read configuration
    read_config

    # Run simulation
    run_simulation

    # Clean up
    rm -f "$CONFIG_FILE" 2>/dev/null || true
}

# Handle Ctrl+C gracefully
trap 'echo -e "\n\n${YELLOW}Cancelled by user. Exiting...${NC}"; exit 130' INT

# Run main
main
