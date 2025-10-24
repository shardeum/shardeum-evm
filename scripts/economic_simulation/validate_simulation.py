#!/usr/bin/env python3
"""
Validation Runner for Economic Simulation

Selects diverse parameter combinations from simulation results,
launches actual test networks, and validates simulated results
against real network behavior.
"""

import argparse
import json
import subprocess
import sys
from pathlib import Path
from typing import List, Dict, Any
import random
import csv
from datetime import datetime


def load_simulation_results(csv_path: Path) -> List[Dict[str, Any]]:
    """Load simulation results from CSV file"""
    results = []
    with open(csv_path, 'r') as f:
        reader = csv.DictReader(f)
        for row in reader:
            # Convert numeric fields
            numeric_fields = [
                'total_supply_shm', 'bonded_tokens_shm', 'bonded_ratio',
                'min_inflation', 'max_inflation', 'goal_bonded',
                'inflation_rate_change', 'current_inflation',
                'annual_provisions_shm', 'validator_apy',
                'inflation_delta', 'bonded_deviation'
            ]
            for field in numeric_fields:
                if field in row:
                    row[field] = float(row[field])
            results.append(row)
    return results


def get_unique_parameter_combinations(
    results: List[Dict[str, Any]]
) -> List[Dict[str, Any]]:
    """
    Extract unique parameter combinations from results.

    Each combination is defined by: min_inflation, max_inflation, goal_bonded
    """
    seen = set()
    unique_combos = []

    for result in results:
        key = (
            result['min_inflation'],
            result['max_inflation'],
            result['goal_bonded']
        )
        if key not in seen:
            seen.add(key)
            unique_combos.append({
                'min_inflation': result['min_inflation'],
                'max_inflation': result['max_inflation'],
                'goal_bonded': result['goal_bonded'],
                'inflation_rate_change': result['inflation_rate_change'],
            })

    return unique_combos


def select_validation_samples(
    combinations: List[Dict[str, Any]],
    num_samples: int = 20,
    strategy: str = 'diverse'
) -> List[Dict[str, Any]]:
    """
    Select parameter combinations for validation.

    Strategies:
    - 'diverse': Select corners, center, and random samples
    - 'random': Purely random selection
    - 'corners': Only corner cases
    """
    if len(combinations) <= num_samples:
        return combinations

    if strategy == 'random':
        return random.sample(combinations, num_samples)

    if strategy == 'corners':
        # Find min/max for each parameter
        min_infs = [c['min_inflation'] for c in combinations]
        max_infs = [c['max_inflation'] for c in combinations]
        goal_bonds = [c['goal_bonded'] for c in combinations]

        corners = []
        for min_i in [min(min_infs), max(min_infs)]:
            for max_i in [min(max_infs), max(max_infs)]:
                for goal_b in [min(goal_bonds), max(goal_bonds)]:
                    # Find closest combination to this corner
                    closest = min(
                        combinations,
                        key=lambda c: (
                            abs(c['min_inflation'] - min_i) +
                            abs(c['max_inflation'] - max_i) +
                            abs(c['goal_bonded'] - goal_b)
                        )
                    )
                    if closest not in corners:
                        corners.append(closest)
        return corners[:num_samples]

    # 'diverse' strategy
    selected = []

    # 1. Add corners (up to 8)
    min_infs = sorted(set(c['min_inflation'] for c in combinations))
    max_infs = sorted(set(c['max_inflation'] for c in combinations))
    goal_bonds = sorted(set(c['goal_bonded'] for c in combinations))

    for min_i in [min_infs[0], min_infs[-1]]:
        for max_i in [max_infs[0], max_infs[-1]]:
            for goal_b in [goal_bonds[0], goal_bonds[-1]]:
                closest = min(
                    combinations,
                    key=lambda c: (
                        abs(c['min_inflation'] - min_i) +
                        abs(c['max_inflation'] - max_i) +
                        abs(c['goal_bonded'] - goal_b)
                    )
                )
                if closest not in selected:
                    selected.append(closest)

    # 2. Add center point (1)
    mid_min = min_infs[len(min_infs) // 2]
    mid_max = max_infs[len(max_infs) // 2]
    mid_goal = goal_bonds[len(goal_bonds) // 2]

    center = min(
        combinations,
        key=lambda c: (
            abs(c['min_inflation'] - mid_min) +
            abs(c['max_inflation'] - mid_max) +
            abs(c['goal_bonded'] - mid_goal)
        )
    )
    if center not in selected:
        selected.append(center)

    # 3. Fill remaining with random samples
    remaining = [c for c in combinations if c not in selected]
    needed = min(num_samples - len(selected), len(remaining))
    selected.extend(random.sample(remaining, needed))

    return selected[:num_samples]


def generate_genesis_file(
    params: Dict[str, Any],
    output_dir: Path
) -> Path:
    """Generate genesis file for given parameters"""
    script_dir = Path(__file__).parent

    # Format filename
    param_str = (
        f"min{int(params['min_inflation']*100)}_"
        f"max{int(params['max_inflation']*100)}_"
        f"goal{int(params['goal_bonded']*100)}"
    )
    output_path = output_dir / f"test-genesis-{param_str}.json"

    # Run generate_test_genesis.py
    cmd = [
        'python3',
        str(script_dir / 'generate_test_genesis.py'),
        '--min-inflation', str(params['min_inflation']),
        '--max-inflation', str(params['max_inflation']),
        '--goal-bonded', str(params['goal_bonded']),
        '--output', str(output_path)
    ]

    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        raise RuntimeError(f"Failed to generate genesis: {result.stderr}")

    return output_path


def run_network_validation(
    genesis_path: Path,
    output_file: Path,
    verbose: bool = False
) -> Dict[str, Any]:
    """Run network validator and return results"""
    script_dir = Path(__file__).parent

    cmd = [
        str(script_dir / 'network_validator.sh'),
        '-g', str(genesis_path),
        '-o', str(output_file)
    ]

    if verbose:
        cmd.append('-v')

    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        raise RuntimeError(f"Network validation failed: {result.stderr}")

    # Parse JSON output from last line
    # (script outputs results JSON at the end)
    try:
        # Load from output file
        with open(output_file, 'r') as f:
            return json.load(f)
    except (json.JSONDecodeError, FileNotFoundError):
        raise RuntimeError("Failed to parse network validation results")


def compare_results(
    simulated: Dict[str, Any],
    actual: Dict[str, Any],
    params: Dict[str, Any]
) -> Dict[str, Any]:
    """
    Compare simulated vs actual network results.

    Returns comparison with discrepancies.
    """
    # Extract actual values from network query
    actual_inflation = float(actual['inflation']['inflation'])
    actual_provisions = float(actual['annual_provisions']['annual_provisions'])

    # Get matching simulated results
    # (for the closest bonding ratio, since network might have different bonded amount)
    sim_inflation = simulated['current_inflation']
    sim_provisions = simulated['annual_provisions_shm']

    # Calculate discrepancies
    inflation_diff = abs(actual_inflation - sim_inflation)
    provisions_diff = abs(actual_provisions - sim_provisions)

    inflation_pct_error = (inflation_diff / sim_inflation * 100) if sim_inflation > 0 else 0
    provisions_pct_error = (provisions_diff / sim_provisions * 100) if sim_provisions > 0 else 0

    return {
        'params': params,
        'simulated': {
            'inflation': sim_inflation,
            'annual_provisions': sim_provisions,
        },
        'actual': {
            'inflation': actual_inflation,
            'annual_provisions': actual_provisions,
        },
        'discrepancy': {
            'inflation_abs': inflation_diff,
            'inflation_pct': inflation_pct_error,
            'provisions_abs': provisions_diff,
            'provisions_pct': provisions_pct_error,
        },
        'match': inflation_pct_error < 1.0 and provisions_pct_error < 1.0
    }


def main():
    parser = argparse.ArgumentParser(
        description='Validate simulation results against actual network behavior'
    )

    parser.add_argument(
        '--input', '-i', type=str, required=True,
        help='CSV file with simulation results'
    )
    parser.add_argument(
        '--num-samples', '-n', type=int, default=15,
        help='Number of parameter combinations to validate (default: 15)'
    )
    parser.add_argument(
        '--strategy', '-s', type=str, default='diverse',
        choices=['diverse', 'random', 'corners'],
        help='Sample selection strategy (default: diverse)'
    )
    parser.add_argument(
        '--output-dir', type=str, default='simulation_results/validation',
        help='Output directory for validation results (default: simulation_results/validation)'
    )
    parser.add_argument(
        '--verbose', '-v', action='store_true',
        help='Verbose output'
    )

    args = parser.parse_args()

    print("Simulation Validation Runner")
    print("=" * 80)
    print()

    # Create output directories
    output_dir = Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    genesis_dir = Path('simulation_configs')
    genesis_dir.mkdir(parents=True, exist_ok=True)

    # Load simulation results
    print(f"Loading simulation results from: {args.input}")
    results = load_simulation_results(Path(args.input))
    print(f"Loaded {len(results)} simulation results")

    # Get unique parameter combinations
    combinations = get_unique_parameter_combinations(results)
    print(f"Found {len(combinations)} unique parameter combinations")

    # Select samples for validation
    print(f"Selecting {args.num_samples} samples using '{args.strategy}' strategy...")
    selected = select_validation_samples(
        combinations,
        num_samples=args.num_samples,
        strategy=args.strategy
    )
    print(f"Selected {len(selected)} parameter combinations for validation")
    print()

    # Run validations
    validation_results = []
    for i, params in enumerate(selected, 1):
        print(f"[{i}/{len(selected)}] Validating parameters:")
        print(f"  min_inflation: {params['min_inflation']:.2%}")
        print(f"  max_inflation: {params['max_inflation']:.2%}")
        print(f"  goal_bonded: {params['goal_bonded']:.2%}")

        try:
            # Generate genesis file
            print("  Generating genesis file...")
            genesis_path = generate_genesis_file(params, genesis_dir)

            # Run network validation
            print("  Launching test network...")
            network_output = output_dir / f"network_result_{i}.json"
            actual_results = run_network_validation(
                genesis_path,
                network_output,
                verbose=args.verbose
            )

            # Find matching simulated result
            # (use result with bonding ratio closest to 0.67 for comparison)
            matching_sim = min(
                [r for r in results if (
                    r['min_inflation'] == params['min_inflation'] and
                    r['max_inflation'] == params['max_inflation'] and
                    r['goal_bonded'] == params['goal_bonded']
                )],
                key=lambda r: abs(r['bonded_ratio'] - 0.67)
            )

            # Compare results
            comparison = compare_results(matching_sim, actual_results, params)
            validation_results.append(comparison)

            match_status = "✓ MATCH" if comparison['match'] else "✗ DISCREPANCY"
            print(f"  {match_status}")
            print(f"    Inflation error: {comparison['discrepancy']['inflation_pct']:.2f}%")
            print(f"    Provisions error: {comparison['discrepancy']['provisions_pct']:.2f}%")

        except Exception as e:
            print(f"  ✗ FAILED: {e}")
            validation_results.append({
                'params': params,
                'error': str(e),
                'match': False
            })

        print()

    # Save validation results
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    results_file = output_dir / f"validation_results_{timestamp}.json"

    with open(results_file, 'w') as f:
        json.dump({
            'timestamp': datetime.now().isoformat(),
            'num_validations': len(validation_results),
            'num_matches': sum(1 for r in validation_results if r.get('match', False)),
            'validations': validation_results
        }, f, indent=2)

    print("=" * 80)
    print(f"Validation complete!")
    print(f"Total validations: {len(validation_results)}")
    print(f"Matches: {sum(1 for r in validation_results if r.get('match', False))}")
    print(f"Discrepancies: {sum(1 for r in validation_results if not r.get('match', True))}")
    print(f"Results saved to: {results_file}")


if __name__ == "__main__":
    main()
