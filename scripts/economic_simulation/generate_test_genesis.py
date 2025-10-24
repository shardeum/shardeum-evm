#!/usr/bin/env python3
"""
Genesis Configuration Generator for Economic Simulation

Generates modified genesis files with custom mint parameters for testing.
Takes a base genesis file and outputs a new file with specified parameters.
"""

import argparse
import json
import sys
from pathlib import Path
from typing import Dict, Any


def load_genesis(genesis_path: Path) -> Dict[str, Any]:
    """Load genesis JSON file"""
    try:
        with open(genesis_path, 'r') as f:
            return json.load(f)
    except FileNotFoundError:
        print(f"Error: Genesis file not found: {genesis_path}", file=sys.stderr)
        sys.exit(1)
    except json.JSONDecodeError as e:
        print(f"Error: Invalid JSON in genesis file: {e}", file=sys.stderr)
        sys.exit(1)


def modify_mint_params(
    genesis: Dict[str, Any],
    min_inflation: float,
    max_inflation: float,
    goal_bonded: float,
    inflation_rate_change: float = None,
    starting_inflation: float = None,
    blocks_per_year: int = None
) -> Dict[str, Any]:
    """
    Modify mint parameters in genesis file.

    Args:
        genesis: Genesis dictionary
        min_inflation: Minimum inflation rate
        max_inflation: Maximum inflation rate
        goal_bonded: Target bonding ratio
        inflation_rate_change: Rate of inflation adjustment (optional)
        starting_inflation: Initial inflation rate (optional)
        blocks_per_year: Blocks per year (optional)

    Returns:
        Modified genesis dictionary
    """
    # Ensure mint module exists
    if 'app_state' not in genesis:
        print("Error: Genesis file missing 'app_state'", file=sys.stderr)
        sys.exit(1)

    if 'mint' not in genesis['app_state']:
        print("Error: Genesis file missing 'mint' module in app_state", file=sys.stderr)
        sys.exit(1)

    # Update mint parameters
    mint_params = genesis['app_state']['mint']['params']

    mint_params['inflation_min'] = f"{min_inflation:.18f}"
    mint_params['inflation_max'] = f"{max_inflation:.18f}"
    mint_params['goal_bonded'] = f"{goal_bonded:.18f}"

    if inflation_rate_change is not None:
        mint_params['inflation_rate_change'] = f"{inflation_rate_change:.18f}"

    if blocks_per_year is not None:
        mint_params['blocks_per_year'] = str(blocks_per_year)

    # Update starting inflation if specified
    if starting_inflation is not None:
        genesis['app_state']['mint']['minter']['inflation'] = f"{starting_inflation:.18f}"

    return genesis


def save_genesis(genesis: Dict[str, Any], output_path: Path):
    """Save genesis JSON file"""
    # Ensure output directory exists
    output_path.parent.mkdir(parents=True, exist_ok=True)

    with open(output_path, 'w') as f:
        json.dump(genesis, f, indent=2)

    print(f"Generated genesis file: {output_path}")


def format_param_string(min_inf: float, max_inf: float, goal_bond: float) -> str:
    """Format parameters into a short string for filename"""
    return f"min{int(min_inf*100)}_max{int(max_inf*100)}_goal{int(goal_bond*100)}"


def main():
    parser = argparse.ArgumentParser(
        description='Generate genesis file with custom mint parameters for testing'
    )

    # Input/Output
    parser.add_argument(
        '--input', '-i', type=str,
        default='config/environments/local-genesis.genesis.json',
        help='Base genesis file to modify (default: local-genesis.genesis.json)'
    )
    parser.add_argument(
        '--output', '-o', type=str,
        help='Output path for generated genesis file (default: auto-generated in simulation_configs/)'
    )
    parser.add_argument(
        '--output-dir', type=str, default='simulation_configs',
        help='Output directory if --output not specified (default: simulation_configs)'
    )

    # Mint parameters (required)
    parser.add_argument(
        '--min-inflation', type=float, required=True,
        help='Minimum inflation rate (e.g., 0.07 for 7%%)'
    )
    parser.add_argument(
        '--max-inflation', type=float, required=True,
        help='Maximum inflation rate (e.g., 0.20 for 20%%)'
    )
    parser.add_argument(
        '--goal-bonded', type=float, required=True,
        help='Target bonding ratio (e.g., 0.67 for 67%%)'
    )

    # Optional mint parameters
    parser.add_argument(
        '--inflation-rate-change', type=float,
        help='Rate of inflation adjustment (default: keep from base genesis)'
    )
    parser.add_argument(
        '--starting-inflation', type=float,
        help='Initial inflation rate (default: keep from base genesis)'
    )
    parser.add_argument(
        '--blocks-per-year', type=int,
        help='Blocks per year (default: keep from base genesis)'
    )

    # Validation
    parser.add_argument(
        '--validate', action='store_true',
        help='Validate parameters before generating'
    )

    args = parser.parse_args()

    # Validate parameters
    if args.validate or True:  # Always validate
        if not (0.0 <= args.min_inflation <= 1.0):
            print(f"Error: min_inflation must be between 0.0 and 1.0, got {args.min_inflation}", file=sys.stderr)
            sys.exit(1)

        if not (0.0 <= args.max_inflation <= 1.0):
            print(f"Error: max_inflation must be between 0.0 and 1.0, got {args.max_inflation}", file=sys.stderr)
            sys.exit(1)

        if args.min_inflation > args.max_inflation:
            print(f"Error: min_inflation ({args.min_inflation}) cannot be greater than max_inflation ({args.max_inflation})", file=sys.stderr)
            sys.exit(1)

        if not (0.0 <= args.goal_bonded <= 1.0):
            print(f"Error: goal_bonded must be between 0.0 and 1.0, got {args.goal_bonded}", file=sys.stderr)
            sys.exit(1)

        if args.inflation_rate_change is not None and not (0.0 <= args.inflation_rate_change <= 1.0):
            print(f"Error: inflation_rate_change must be between 0.0 and 1.0, got {args.inflation_rate_change}", file=sys.stderr)
            sys.exit(1)

        if args.starting_inflation is not None and not (0.0 <= args.starting_inflation <= 1.0):
            print(f"Error: starting_inflation must be between 0.0 and 1.0, got {args.starting_inflation}", file=sys.stderr)
            sys.exit(1)

    print("Genesis Configuration Generator")
    print("=" * 80)
    print()

    # Resolve input path
    input_path = Path(args.input)
    if not input_path.is_absolute():
        # Assume relative to repo root
        repo_root = Path(__file__).parent.parent.parent
        input_path = repo_root / input_path

    print(f"Loading base genesis: {input_path}")
    genesis = load_genesis(input_path)

    # Modify mint parameters
    print(f"Modifying mint parameters:")
    print(f"  min_inflation: {args.min_inflation:.2%}")
    print(f"  max_inflation: {args.max_inflation:.2%}")
    print(f"  goal_bonded: {args.goal_bonded:.2%}")

    if args.inflation_rate_change is not None:
        print(f"  inflation_rate_change: {args.inflation_rate_change:.2%}")
    if args.starting_inflation is not None:
        print(f"  starting_inflation: {args.starting_inflation:.2%}")
    if args.blocks_per_year is not None:
        print(f"  blocks_per_year: {args.blocks_per_year:,}")

    genesis = modify_mint_params(
        genesis,
        min_inflation=args.min_inflation,
        max_inflation=args.max_inflation,
        goal_bonded=args.goal_bonded,
        inflation_rate_change=args.inflation_rate_change,
        starting_inflation=args.starting_inflation,
        blocks_per_year=args.blocks_per_year
    )

    # Determine output path
    if args.output:
        output_path = Path(args.output)
    else:
        # Auto-generate filename
        param_str = format_param_string(
            args.min_inflation,
            args.max_inflation,
            args.goal_bonded
        )
        output_dir = Path(args.output_dir)
        if not output_dir.is_absolute():
            repo_root = Path(__file__).parent.parent.parent
            output_dir = repo_root / output_dir

        output_path = output_dir / f"test-genesis-{param_str}.json"

    # Save modified genesis
    save_genesis(genesis, output_path)

    print()
    print("=" * 80)
    print(f"Success! Generated genesis file with custom mint parameters")
    print(f"Output: {output_path}")
    print()
    print("To use this genesis file:")
    print(f"  ./local_node.sh -g {output_path}")


if __name__ == "__main__":
    main()
