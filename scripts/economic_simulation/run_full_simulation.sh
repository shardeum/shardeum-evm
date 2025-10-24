#!/bin/bash
#
# Full Economic Simulation Runner
#
# Master orchestrator that runs the complete simulation workflow:
# 1. Parameter sweep simulation
# 2. Network validation (optional)
# 3. Report generation
#

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Default values
RUN_VALIDATION=false
NUM_VALIDATIONS=10
SKIP_REPORT=false
OUTPUT_DIR="simulation_results"
OUTPUT_NAME="sweep"

# Timing
START_TIME=$(date +%s)

# Usage
usage() {
    cat <<EOF
Usage: $0 [OPTIONS]

Run complete economic simulation workflow.

OPTIONS:
    --validate              Run network validation after simulation
    --num-validations N     Number of validations to run (default: 10)
    --skip-report          Skip report generation
    --output-dir DIR       Output directory (default: simulation_results)
    --output-name NAME     Base name for output files (default: sweep)

    # Parameter ranges (passed to run_simulation.py)
    --min-inflation-min F   Minimum value for min_inflation (default: 0.03)
    --min-inflation-max F   Maximum value for min_inflation (default: 0.15)
    --max-inflation-min F   Minimum value for max_inflation (default: 0.10)
    --max-inflation-max F   Maximum value for max_inflation (default: 0.40)
    --goal-bonded-min F     Minimum value for goal_bonded (default: 0.40)
    --goal-bonded-max F     Maximum value for goal_bonded (default: 0.90)
    --num-points N          Number of points per parameter (default: 10)

    -h, --help             Show this help message

EXAMPLE:
    # Quick simulation with default parameters
    $0

    # Full simulation with validation
    $0 --validate --num-validations 15

    # Custom parameter ranges
    $0 --min-inflation-min 0.05 --min-inflation-max 0.12 --num-points 5

EOF
    exit 1
}

# Parse arguments
SIMULATION_ARGS=()
while [[ $# -gt 0 ]]; do
    case $1 in
        --validate)
            RUN_VALIDATION=true
            shift
            ;;
        --num-validations)
            NUM_VALIDATIONS="$2"
            shift 2
            ;;
        --skip-report)
            SKIP_REPORT=true
            shift
            ;;
        --output-dir)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        --output-name)
            OUTPUT_NAME="$2"
            shift 2
            ;;
        --min-inflation-min|--min-inflation-max|--max-inflation-min|--max-inflation-max|\
        --goal-bonded-min|--goal-bonded-max|--num-points|--bonding-ratio-min|--bonding-ratio-max|\
        --num-bonding-ratios)
            SIMULATION_ARGS+=("$1" "$2")
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo -e "${RED}Error: Unknown option $1${NC}"
            usage
            ;;
    esac
done

log_step() {
    echo -e "\n${BLUE}==>${NC} ${GREEN}$1${NC}\n"
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Check dependencies
check_dependencies() {
    log_step "Checking dependencies..."

    local missing_deps=false

    if ! command -v python3 &> /dev/null; then
        log_error "python3 not found"
        missing_deps=true
    fi

    if ! command -v jq &> /dev/null; then
        log_error "jq not found (needed for JSON parsing)"
        missing_deps=true
    fi

    # Check Python packages
    if ! python3 -c "import matplotlib" &> /dev/null; then
        echo -e "${YELLOW}[WARN]${NC} matplotlib not found - install with: pip install matplotlib numpy"
    fi

    if [[ "$missing_deps" == "true" ]]; then
        log_error "Missing required dependencies"
        exit 1
    fi

    log_success "All required dependencies found"
}

# Print configuration
print_config() {
    log_step "Simulation Configuration"
    echo "Output Directory: $OUTPUT_DIR"
    echo "Output Name: $OUTPUT_NAME"
    echo "Run Validation: $RUN_VALIDATION"
    if [[ "$RUN_VALIDATION" == "true" ]]; then
        echo "Number of Validations: $NUM_VALIDATIONS"
    fi
    echo "Skip Report: $SKIP_REPORT"

    if [[ ${#SIMULATION_ARGS[@]} -gt 0 ]]; then
        echo -e "\nCustom Parameters:"
        for ((i=0; i<${#SIMULATION_ARGS[@]}; i+=2)); do
            echo "  ${SIMULATION_ARGS[i]}: ${SIMULATION_ARGS[i+1]}"
        done
    fi
}

# Run parameter sweep
run_parameter_sweep() {
    log_step "Running parameter sweep simulation..."

    cd "$REPO_ROOT"

    local cmd=(
        python3 "$SCRIPT_DIR/run_simulation.py"
        --output-dir "$OUTPUT_DIR"
        --output-name "$OUTPUT_NAME"
    )

    # Add custom args
    cmd+=("${SIMULATION_ARGS[@]}")

    log_info "Command: ${cmd[*]}"
    echo

    if ! "${cmd[@]}"; then
        log_error "Parameter sweep failed"
        exit 1
    fi

    # Find the latest CSV file
    LATEST_CSV=$(ls -t "$OUTPUT_DIR"/${OUTPUT_NAME}_*.csv 2>/dev/null | head -n1)
    LATEST_JSON=$(ls -t "$OUTPUT_DIR"/${OUTPUT_NAME}_*.json 2>/dev/null | head -n1)
    LATEST_METADATA=$(ls -t "$OUTPUT_DIR"/${OUTPUT_NAME}_*_metadata.json 2>/dev/null | head -n1)

    if [[ -z "$LATEST_CSV" ]]; then
        log_error "Could not find simulation output CSV"
        exit 1
    fi

    log_success "Parameter sweep complete: $LATEST_CSV"

    # Export for use by other functions
    export SIMULATION_CSV="$LATEST_CSV"
    export SIMULATION_JSON="$LATEST_JSON"
    export SIMULATION_METADATA="$LATEST_METADATA"
}

# Run validation
run_validation() {
    if [[ "$RUN_VALIDATION" != "true" ]]; then
        return 0
    fi

    log_step "Running network validation..."

    cd "$REPO_ROOT"

    local cmd=(
        python3 "$SCRIPT_DIR/validate_simulation.py"
        --input "$SIMULATION_CSV"
        --num-samples "$NUM_VALIDATIONS"
        --output-dir "$OUTPUT_DIR/validation"
        --verbose
    )

    log_info "Command: ${cmd[*]}"
    echo

    if ! "${cmd[@]}"; then
        log_error "Validation failed"
        exit 1
    fi

    log_success "Validation complete"
}

# Generate report
generate_report() {
    if [[ "$SKIP_REPORT" == "true" ]]; then
        return 0
    fi

    log_step "Generating report..."

    cd "$REPO_ROOT"

    local cmd=(
        python3 "$SCRIPT_DIR/generate_report.py"
        --input "$SIMULATION_CSV"
        --output-dir "$OUTPUT_DIR/reports"
    )

    if [[ -n "$SIMULATION_METADATA" ]]; then
        cmd+=(--metadata "$SIMULATION_METADATA")
    fi

    log_info "Command: ${cmd[*]}"
    echo

    if ! "${cmd[@]}"; then
        log_error "Report generation failed"
        exit 1
    fi

    # Find the latest report
    LATEST_REPORT=$(ls -t "$OUTPUT_DIR"/reports/economic_report_*.html 2>/dev/null | head -n1)

    if [[ -n "$LATEST_REPORT" ]]; then
        log_success "Report generated: $LATEST_REPORT"
        echo
        echo -e "${GREEN}Open in browser:${NC} file://$(realpath "$LATEST_REPORT")"
    fi
}

# Print summary
print_summary() {
    local end_time=$(date +%s)
    local duration=$((end_time - START_TIME))
    local minutes=$((duration / 60))
    local seconds=$((duration % 60))

    echo
    echo "=========================================="
    echo -e "${GREEN}Simulation Complete!${NC}"
    echo "=========================================="
    echo
    echo "Duration: ${minutes}m ${seconds}s"
    echo
    echo "Results:"
    echo "  CSV: $SIMULATION_CSV"
    if [[ -n "$SIMULATION_JSON" ]]; then
        echo "  JSON: $SIMULATION_JSON"
    fi
    if [[ -n "$LATEST_REPORT" ]]; then
        echo "  Report: $LATEST_REPORT"
    fi
    echo
    echo "Output directory: $OUTPUT_DIR"
    echo
}

# Main execution
main() {
    echo "=========================================="
    echo "  Economic Simulation Runner"
    echo "  Shardeum Cosmos Network"
    echo "=========================================="
    echo

    check_dependencies
    print_config
    run_parameter_sweep
    run_validation
    generate_report
    print_summary
}

# Run main
main
