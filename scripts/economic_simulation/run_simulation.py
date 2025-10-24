#!/usr/bin/env python3
"""
Parameter Sweep Runner for Economic Simulation

Generates combinations of mint parameters and runs the economic simulator
for each combination across various bonding ratios.

Output: CSV and JSON files with simulation results for analysis.
"""

import argparse
import csv
import json
import itertools
import sys
from pathlib import Path
from typing import List, Dict, Any
from datetime import datetime

# Import the economic simulator
from economic_simulator import (
    run_single_simulation,
    INITIAL_SUPPLY_SHM,
    BLOCKS_PER_YEAR
)


def generate_parameter_combinations(
    min_inflation_range: tuple,
    max_inflation_range: tuple,
    goal_bonded_range: tuple,
    num_points: int = 10
) -> List[Dict[str, float]]:
    """
    Generate parameter combinations for simulation.

    Args:
        min_inflation_range: (min, max) for min_inflation parameter
        max_inflation_range: (min, max) for max_inflation parameter
        goal_bonded_range: (min, max) for goal_bonded parameter
        num_points: Number of points to sample per parameter

    Returns:
        List of parameter dictionaries
    """
    # Generate evenly spaced points for each parameter
    # Special case: if num_points is 1, just use the min value
    if num_points == 1:
        min_inflations = [min_inflation_range[0]]
        max_inflations = [max_inflation_range[0]]
        goal_bondeds = [goal_bonded_range[0]]
    else:
        min_inflations = [
            min_inflation_range[0] + i * (min_inflation_range[1] - min_inflation_range[0]) / (num_points - 1)
            for i in range(num_points)
        ]

        max_inflations = [
            max_inflation_range[0] + i * (max_inflation_range[1] - max_inflation_range[0]) / (num_points - 1)
            for i in range(num_points)
        ]

        goal_bondeds = [
            goal_bonded_range[0] + i * (goal_bonded_range[1] - goal_bonded_range[0]) / (num_points - 1)
            for i in range(num_points)
        ]

    # Generate all combinations
    combinations = []
    for min_inf, max_inf, goal_bond in itertools.product(
        min_inflations, max_inflations, goal_bondeds
    ):
        # Validate: max_inflation must be >= min_inflation
        if max_inf >= min_inf:
            combinations.append({
                'min_inflation': round(min_inf, 4),
                'max_inflation': round(max_inf, 4),
                'goal_bonded': round(goal_bond, 4),
            })

    return combinations


def generate_bonding_ratios(
    min_ratio: float = 0.05,
    max_ratio: float = 0.95,
    num_points: int = 10
) -> List[float]:
    """
    Generate bonding ratios to test.

    Args:
        min_ratio: Minimum bonding ratio
        max_ratio: Maximum bonding ratio
        num_points: Number of points to sample

    Returns:
        List of bonding ratios
    """
    return [
        round(min_ratio + i * (max_ratio - min_ratio) / (num_points - 1), 4)
        for i in range(num_points)
    ]


def run_parameter_sweep(
    param_combinations: List[Dict[str, float]],
    bonding_ratios: List[float],
    total_supply: float = INITIAL_SUPPLY_SHM,
    starting_inflation: float = 0.13,
    inflation_rate_change: float = 0.13,
    verbose: bool = True
) -> List[Dict[str, Any]]:
    """
    Run simulation for all parameter combinations and bonding ratios.

    Args:
        param_combinations: List of parameter dictionaries
        bonding_ratios: List of bonding ratios to test
        total_supply: Total supply in SHM
        starting_inflation: Initial inflation rate
        inflation_rate_change: Rate of inflation adjustment
        verbose: Print progress

    Returns:
        List of result dictionaries
    """
    results = []
    total_runs = len(param_combinations) * len(bonding_ratios)
    run_count = 0

    if verbose:
        print(f"Running {len(param_combinations)} parameter combinations")
        print(f"Testing {len(bonding_ratios)} bonding ratios each")
        print(f"Total simulation runs: {total_runs}")
        print()

    for params in param_combinations:
        for bonded_ratio in bonding_ratios:
            run_count += 1

            if verbose and run_count % 100 == 0:
                print(f"Progress: {run_count}/{total_runs} ({100*run_count/total_runs:.1f}%)")

            # Run simulation
            result = run_single_simulation(
                total_supply_shm=total_supply,
                bonded_ratio=bonded_ratio,
                current_inflation=starting_inflation,
                min_inflation=params['min_inflation'],
                max_inflation=params['max_inflation'],
                goal_bonded=params['goal_bonded'],
                inflation_rate_change=inflation_rate_change
            )

            # Convert to dict and append
            results.append(result.to_dict())

    if verbose:
        print(f"Completed {total_runs} simulations")

    return results


def save_results_csv(results: List[Dict[str, Any]], output_path: Path):
    """Save results to CSV file"""
    if not results:
        print("No results to save")
        return

    # Get all keys from first result
    fieldnames = list(results[0].keys())

    with open(output_path, 'w', newline='') as csvfile:
        writer = csv.DictWriter(csvfile, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(results)

    print(f"Saved CSV results to: {output_path}")


def save_results_json(results: List[Dict[str, Any]], output_path: Path):
    """Save results to JSON file"""
    with open(output_path, 'w') as jsonfile:
        json.dump(results, jsonfile, indent=2)

    print(f"Saved JSON results to: {output_path}")


def save_metadata(
    output_path: Path,
    param_combinations: List[Dict[str, float]],
    bonding_ratios: List[float],
    args: argparse.Namespace
):
    """Save simulation metadata"""
    metadata = {
        'timestamp': datetime.now().isoformat(),
        'total_simulations': len(param_combinations) * len(bonding_ratios),
        'num_parameter_combinations': len(param_combinations),
        'num_bonding_ratios': len(bonding_ratios),
        'parameters': {
            'min_inflation_range': [args.min_inflation_min, args.min_inflation_max],
            'max_inflation_range': [args.max_inflation_min, args.max_inflation_max],
            'goal_bonded_range': [args.goal_bonded_min, args.goal_bonded_max],
            'bonding_ratio_range': [args.bonding_ratio_min, args.bonding_ratio_max],
            'num_points_per_param': args.num_points,
            'num_bonding_ratios': args.num_bonding_ratios,
        },
        'constants': {
            'total_supply_shm': args.total_supply,
            'starting_inflation': args.starting_inflation,
            'inflation_rate_change': args.inflation_rate_change,
            'blocks_per_year': BLOCKS_PER_YEAR,
        }
    }

    with open(output_path, 'w') as f:
        json.dump(metadata, f, indent=2)

    print(f"Saved metadata to: {output_path}")


def main():
    parser = argparse.ArgumentParser(
        description='Run economic parameter sweep simulation'
    )

    # Parameter ranges
    parser.add_argument(
        '--min-inflation-min', type=float, default=0.03,
        help='Minimum value for min_inflation (default: 0.03 = 3%%)'
    )
    parser.add_argument(
        '--min-inflation-max', type=float, default=0.15,
        help='Maximum value for min_inflation (default: 0.15 = 15%%)'
    )
    parser.add_argument(
        '--max-inflation-min', type=float, default=0.10,
        help='Minimum value for max_inflation (default: 0.10 = 10%%)'
    )
    parser.add_argument(
        '--max-inflation-max', type=float, default=0.40,
        help='Maximum value for max_inflation (default: 0.40 = 40%%)'
    )
    parser.add_argument(
        '--goal-bonded-min', type=float, default=0.40,
        help='Minimum value for goal_bonded (default: 0.40 = 40%%)'
    )
    parser.add_argument(
        '--goal-bonded-max', type=float, default=0.90,
        help='Maximum value for goal_bonded (default: 0.90 = 90%%)'
    )

    # Bonding ratio range
    parser.add_argument(
        '--bonding-ratio-min', type=float, default=0.05,
        help='Minimum bonding ratio to test (default: 0.05 = 5%%, can go as low as 0.01 = 1%%)'
    )
    parser.add_argument(
        '--bonding-ratio-max', type=float, default=0.95,
        help='Maximum bonding ratio to test (default: 0.95 = 95%%, can go up to 0.99 = 99%%)'
    )

    # Sampling parameters
    parser.add_argument(
        '--num-points', type=int, default=10,
        help='Number of points to sample per parameter (default: 10)'
    )
    parser.add_argument(
        '--num-bonding-ratios', type=int, default=10,
        help='Number of bonding ratios to test (default: 10)'
    )

    # Network parameters
    parser.add_argument(
        '--total-supply', type=float, default=INITIAL_SUPPLY_SHM,
        help=f'Total supply in SHM (default: {INITIAL_SUPPLY_SHM:,.0f})'
    )
    parser.add_argument(
        '--starting-inflation', type=float, default=0.13,
        help='Starting inflation rate (default: 0.13 = 13%%)'
    )
    parser.add_argument(
        '--inflation-rate-change', type=float, default=0.13,
        help='Inflation rate change parameter (default: 0.13)'
    )

    # Output parameters
    parser.add_argument(
        '--output-dir', type=str, default='simulation_results',
        help='Output directory for results (default: simulation_results)'
    )
    parser.add_argument(
        '--output-name', type=str, default='sweep',
        help='Base name for output files (default: sweep)'
    )
    parser.add_argument(
        '--quiet', action='store_true',
        help='Suppress progress output'
    )

    args = parser.parse_args()

    # Create output directory
    output_dir = Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    print("Economic Parameter Sweep Simulation")
    print("=" * 80)
    print()

    # Generate parameter combinations
    print("Generating parameter combinations...")
    param_combinations = generate_parameter_combinations(
        min_inflation_range=(args.min_inflation_min, args.min_inflation_max),
        max_inflation_range=(args.max_inflation_min, args.max_inflation_max),
        goal_bonded_range=(args.goal_bonded_min, args.goal_bonded_max),
        num_points=args.num_points
    )
    print(f"Generated {len(param_combinations)} valid parameter combinations")
    print()

    # Generate bonding ratios
    bonding_ratios = generate_bonding_ratios(
        min_ratio=args.bonding_ratio_min,
        max_ratio=args.bonding_ratio_max,
        num_points=args.num_bonding_ratios
    )

    # Run parameter sweep
    print("Running simulations...")
    results = run_parameter_sweep(
        param_combinations=param_combinations,
        bonding_ratios=bonding_ratios,
        total_supply=args.total_supply,
        starting_inflation=args.starting_inflation,
        inflation_rate_change=args.inflation_rate_change,
        verbose=not args.quiet
    )
    print()

    # Save results
    print("Saving results...")
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    csv_path = output_dir / f"{args.output_name}_{timestamp}.csv"
    json_path = output_dir / f"{args.output_name}_{timestamp}.json"
    metadata_path = output_dir / f"{args.output_name}_{timestamp}_metadata.json"

    save_results_csv(results, csv_path)
    save_results_json(results, json_path)
    save_metadata(metadata_path, param_combinations, bonding_ratios, args)

    print()
    print("=" * 80)
    print(f"Simulation complete!")
    print(f"Total runs: {len(results)}")
    print(f"Results saved to: {output_dir}")


if __name__ == "__main__":
    main()
