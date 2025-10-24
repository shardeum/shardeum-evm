#!/usr/bin/env python3
"""
Report Generator for Economic Simulation

Generates visualizations and comprehensive reports from simulation results.
Creates plots, summary tables, and HTML reports for analysis.
"""

import argparse
import csv
import json
import sys
from pathlib import Path
from typing import List, Dict, Any, Tuple
from datetime import datetime
import warnings

# Try to import visualization libraries
try:
    import matplotlib
    matplotlib.use('Agg')  # Use non-interactive backend
    import matplotlib.pyplot as plt
    from matplotlib import cm
    import numpy as np
    VISUALIZATION_AVAILABLE = True
except ImportError:
    print("Warning: matplotlib not available. Install with: pip install matplotlib numpy", file=sys.stderr)
    VISUALIZATION_AVAILABLE = False


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


def generate_summary_statistics(results: List[Dict[str, Any]]) -> Dict[str, Any]:
    """Generate summary statistics from results"""
    if not results:
        return {}

    # Extract key metrics
    apys = [r['validator_apy'] for r in results]
    inflations = [r['current_inflation'] for r in results]
    provisions = [r['annual_provisions_shm'] for r in results]

    return {
        'total_simulations': len(results),
        'validator_apy': {
            'min': min(apys),
            'max': max(apys),
            'mean': sum(apys) / len(apys),
            'median': sorted(apys)[len(apys) // 2],
        },
        'inflation': {
            'min': min(inflations),
            'max': max(inflations),
            'mean': sum(inflations) / len(inflations),
            'median': sorted(inflations)[len(inflations) // 2],
        },
        'annual_provisions': {
            'min': min(provisions),
            'max': max(provisions),
            'mean': sum(provisions) / len(provisions),
            'median': sorted(provisions)[len(provisions) // 2],
        }
    }


def plot_apy_vs_bonding_ratio(
    results: List[Dict[str, Any]],
    output_path: Path,
    title: str = "Validator APY vs Bonding Ratio"
):
    """Plot APY against bonding ratio for different parameter combinations"""
    if not VISUALIZATION_AVAILABLE:
        return

    # Group by parameter combination
    param_groups = {}
    for r in results:
        key = (r['min_inflation'], r['max_inflation'], r['goal_bonded'])
        if key not in param_groups:
            param_groups[key] = []
        param_groups[key].append(r)

    plt.figure(figsize=(12, 8))

    # Plot a sample of parameter combinations (max 10 to avoid clutter)
    sample_keys = sorted(param_groups.keys())[:10]

    for key in sample_keys:
        data = sorted(param_groups[key], key=lambda x: x['bonded_ratio'])
        bonding_ratios = [d['bonded_ratio'] for d in data]
        apys = [d['validator_apy'] * 100 for d in data]  # Convert to percentage

        label = f"min={key[0]:.0%}, max={key[1]:.0%}, goal={key[2]:.0%}"
        plt.plot(bonding_ratios, apys, marker='o', label=label, linewidth=2, markersize=4)

    plt.xlabel('Bonding Ratio', fontsize=12)
    plt.ylabel('Validator APY (%)', fontsize=12)
    plt.title(title, fontsize=14, fontweight='bold')
    plt.grid(True, alpha=0.3)
    plt.legend(bbox_to_anchor=(1.05, 1), loc='upper left', fontsize=9)
    plt.tight_layout()
    plt.savefig(output_path, dpi=300, bbox_inches='tight')
    plt.close()

    print(f"Generated plot: {output_path}")


def plot_heatmap(
    results: List[Dict[str, Any]],
    output_path: Path,
    x_param: str,
    y_param: str,
    z_metric: str,
    title: str = None,
    fixed_params: Dict[str, float] = None
):
    """Generate 2D heatmap for parameter exploration"""
    if not VISUALIZATION_AVAILABLE:
        return

    # Filter results by fixed parameters
    filtered = results
    if fixed_params:
        for param, value in fixed_params.items():
            filtered = [r for r in filtered if abs(r[param] - value) < 0.001]

    if not filtered:
        print(f"Warning: No data for heatmap with fixed params {fixed_params}")
        return

    # Extract unique values for x and y
    x_values = sorted(set(r[x_param] for r in filtered))
    y_values = sorted(set(r[y_param] for r in filtered))

    # Create grid
    z_grid = np.zeros((len(y_values), len(x_values)))

    for i, y_val in enumerate(y_values):
        for j, x_val in enumerate(x_values):
            # Find matching results and average z_metric
            matches = [r for r in filtered
                      if abs(r[x_param] - x_val) < 0.001 and
                         abs(r[y_param] - y_val) < 0.001]
            if matches:
                z_grid[i, j] = sum(r[z_metric] for r in matches) / len(matches)
            else:
                z_grid[i, j] = np.nan

    # Create heatmap
    plt.figure(figsize=(10, 8))
    im = plt.imshow(z_grid, aspect='auto', origin='lower',
                    extent=[min(x_values), max(x_values), min(y_values), max(y_values)],
                    cmap='viridis')

    plt.colorbar(im, label=z_metric.replace('_', ' ').title())
    plt.xlabel(x_param.replace('_', ' ').title(), fontsize=12)
    plt.ylabel(y_param.replace('_', ' ').title(), fontsize=12)

    if title:
        plt.title(title, fontsize=14, fontweight='bold')
    else:
        plt.title(f"{z_metric.replace('_', ' ').title()} Heatmap", fontsize=14, fontweight='bold')

    plt.tight_layout()
    plt.savefig(output_path, dpi=300, bbox_inches='tight')
    plt.close()

    print(f"Generated heatmap: {output_path}")


def plot_provisions_comparison(
    results: List[Dict[str, Any]],
    output_path: Path
):
    """Plot annual provisions for different parameter combinations"""
    if not VISUALIZATION_AVAILABLE:
        return

    # Group by goal_bonded and bonding_ratio
    goal_bonded_values = sorted(set(r['goal_bonded'] for r in results))

    # Sample a few goal_bonded values
    sample_goals = [goal_bonded_values[0], goal_bonded_values[len(goal_bonded_values)//2], goal_bonded_values[-1]]

    fig, axes = plt.subplots(1, len(sample_goals), figsize=(15, 5))
    if len(sample_goals) == 1:
        axes = [axes]

    for ax, goal in zip(axes, sample_goals):
        # Filter results for this goal_bonded
        filtered = [r for r in results if abs(r['goal_bonded'] - goal) < 0.001]

        # Group by min/max inflation
        param_groups = {}
        for r in filtered:
            key = (r['min_inflation'], r['max_inflation'])
            if key not in param_groups:
                param_groups[key] = []
            param_groups[key].append(r)

        # Plot a sample
        sample_keys = sorted(param_groups.keys())[:5]
        for key in sample_keys:
            data = sorted(param_groups[key], key=lambda x: x['bonded_ratio'])
            bonding_ratios = [d['bonded_ratio'] for d in data]
            provisions = [d['annual_provisions_shm'] / 1e9 for d in data]  # Convert to billions

            label = f"min={key[0]:.0%}, max={key[1]:.0%}"
            ax.plot(bonding_ratios, provisions, marker='o', label=label, linewidth=2, markersize=4)

        ax.set_xlabel('Bonding Ratio', fontsize=10)
        ax.set_ylabel('Annual Provisions (B SHM)', fontsize=10)
        ax.set_title(f'Goal Bonded: {goal:.0%}', fontsize=12, fontweight='bold')
        ax.grid(True, alpha=0.3)
        ax.legend(fontsize=8)

    plt.tight_layout()
    plt.savefig(output_path, dpi=300, bbox_inches='tight')
    plt.close()

    print(f"Generated provisions plot: {output_path}")


def generate_html_report(
    results: List[Dict[str, Any]],
    summary_stats: Dict[str, Any],
    plot_files: List[Path],
    output_path: Path,
    metadata: Dict[str, Any] = None
):
    """Generate HTML report with visualizations and statistics"""
    html_content = f"""
<!DOCTYPE html>
<html>
<head>
    <title>Economic Simulation Report</title>
    <style>
        body {{
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
        }}
        .header {{
            background-color: #2c3e50;
            color: white;
            padding: 30px;
            border-radius: 10px;
            margin-bottom: 30px;
        }}
        .header h1 {{
            margin: 0;
            font-size: 2.5em;
        }}
        .header .subtitle {{
            margin-top: 10px;
            opacity: 0.8;
            font-size: 1.1em;
        }}
        .section {{
            background-color: white;
            padding: 25px;
            margin-bottom: 25px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }}
        .section h2 {{
            color: #2c3e50;
            border-bottom: 3px solid #3498db;
            padding-bottom: 10px;
            margin-top: 0;
        }}
        .stats-grid {{
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-top: 20px;
        }}
        .stat-card {{
            background-color: #ecf0f1;
            padding: 20px;
            border-radius: 8px;
            border-left: 4px solid #3498db;
        }}
        .stat-card h3 {{
            margin: 0 0 15px 0;
            color: #2c3e50;
            font-size: 1.1em;
        }}
        .stat-value {{
            display: flex;
            justify-content: space-between;
            margin: 8px 0;
            font-size: 0.95em;
        }}
        .stat-label {{
            color: #7f8c8d;
        }}
        .stat-number {{
            font-weight: bold;
            color: #2c3e50;
        }}
        .plot {{
            margin: 20px 0;
            text-align: center;
        }}
        .plot img {{
            max-width: 100%;
            border-radius: 8px;
            box-shadow: 0 4px 8px rgba(0,0,0,0.1);
        }}
        .plot-caption {{
            margin-top: 10px;
            color: #7f8c8d;
            font-size: 0.9em;
        }}
        .metadata {{
            background-color: #f8f9fa;
            padding: 15px;
            border-radius: 5px;
            font-family: monospace;
            font-size: 0.9em;
        }}
        .footer {{
            text-align: center;
            color: #7f8c8d;
            margin-top: 50px;
            padding-top: 20px;
            border-top: 1px solid #ddd;
        }}
    </style>
</head>
<body>
    <div class="header">
        <h1>Economic Simulation Report</h1>
        <div class="subtitle">Shardeum Cosmos Network Parameter Analysis</div>
        <div class="subtitle">Generated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}</div>
    </div>

    <div class="section">
        <h2>Summary Statistics</h2>
        <div class="stats-grid">
            <div class="stat-card">
                <h3>Validator APY</h3>
                <div class="stat-value">
                    <span class="stat-label">Minimum:</span>
                    <span class="stat-number">{summary_stats['validator_apy']['min']:.2%}</span>
                </div>
                <div class="stat-value">
                    <span class="stat-label">Maximum:</span>
                    <span class="stat-number">{summary_stats['validator_apy']['max']:.2%}</span>
                </div>
                <div class="stat-value">
                    <span class="stat-label">Average:</span>
                    <span class="stat-number">{summary_stats['validator_apy']['mean']:.2%}</span>
                </div>
                <div class="stat-value">
                    <span class="stat-label">Median:</span>
                    <span class="stat-number">{summary_stats['validator_apy']['median']:.2%}</span>
                </div>
            </div>

            <div class="stat-card">
                <h3>Inflation Rate</h3>
                <div class="stat-value">
                    <span class="stat-label">Minimum:</span>
                    <span class="stat-number">{summary_stats['inflation']['min']:.2%}</span>
                </div>
                <div class="stat-value">
                    <span class="stat-label">Maximum:</span>
                    <span class="stat-number">{summary_stats['inflation']['max']:.2%}</span>
                </div>
                <div class="stat-value">
                    <span class="stat-label">Average:</span>
                    <span class="stat-number">{summary_stats['inflation']['mean']:.2%}</span>
                </div>
                <div class="stat-value">
                    <span class="stat-label">Median:</span>
                    <span class="stat-number">{summary_stats['inflation']['median']:.2%}</span>
                </div>
            </div>

            <div class="stat-card">
                <h3>Annual Provisions</h3>
                <div class="stat-value">
                    <span class="stat-label">Minimum:</span>
                    <span class="stat-number">{summary_stats['annual_provisions']['min']/1e9:.2f}B SHM</span>
                </div>
                <div class="stat-value">
                    <span class="stat-label">Maximum:</span>
                    <span class="stat-number">{summary_stats['annual_provisions']['max']/1e9:.2f}B SHM</span>
                </div>
                <div class="stat-value">
                    <span class="stat-label">Average:</span>
                    <span class="stat-number">{summary_stats['annual_provisions']['mean']/1e9:.2f}B SHM</span>
                </div>
                <div class="stat-value">
                    <span class="stat-label">Median:</span>
                    <span class="stat-number">{summary_stats['annual_provisions']['median']/1e9:.2f}B SHM</span>
                </div>
            </div>

            <div class="stat-card">
                <h3>Simulation Info</h3>
                <div class="stat-value">
                    <span class="stat-label">Total Runs:</span>
                    <span class="stat-number">{summary_stats['total_simulations']:,}</span>
                </div>
            </div>
        </div>
    </div>

    <div class="section">
        <h2>Visualizations</h2>
"""

    # Add plots
    for i, plot_file in enumerate(plot_files):
        if plot_file.exists():
            rel_path = plot_file.name
            html_content += f"""
        <div class="plot">
            <img src="{rel_path}" alt="Plot {i+1}">
            <div class="plot-caption">{plot_file.stem.replace('_', ' ').title()}</div>
        </div>
"""

    html_content += """
    </div>
"""

    # Add metadata if provided
    if metadata:
        html_content += f"""
    <div class="section">
        <h2>Simulation Parameters</h2>
        <div class="metadata">
            <pre>{json.dumps(metadata, indent=2)}</pre>
        </div>
    </div>
"""

    html_content += """
    <div class="footer">
        <p>Generated by Shardeum Economic Simulator</p>
    </div>
</body>
</html>
"""

    with open(output_path, 'w') as f:
        f.write(html_content)

    print(f"Generated HTML report: {output_path}")


def main():
    parser = argparse.ArgumentParser(
        description='Generate visualizations and reports from simulation results'
    )

    parser.add_argument(
        '--input', '-i', type=str, required=True,
        help='CSV file with simulation results'
    )
    parser.add_argument(
        '--output-dir', type=str, default='simulation_results/reports',
        help='Output directory for reports and plots (default: simulation_results/reports)'
    )
    parser.add_argument(
        '--metadata', type=str,
        help='JSON file with simulation metadata'
    )

    args = parser.parse_args()

    if not VISUALIZATION_AVAILABLE:
        print("Error: Visualization libraries not available", file=sys.stderr)
        print("Install with: pip install matplotlib numpy", file=sys.stderr)
        sys.exit(1)

    print("Economic Simulation Report Generator")
    print("=" * 80)
    print()

    # Create output directory
    output_dir = Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    # Load results
    print(f"Loading simulation results from: {args.input}")
    results = load_simulation_results(Path(args.input))
    print(f"Loaded {len(results)} simulation results")
    print()

    # Generate summary statistics
    print("Calculating summary statistics...")
    summary_stats = generate_summary_statistics(results)

    # Load metadata if provided
    metadata = None
    if args.metadata:
        with open(args.metadata, 'r') as f:
            metadata = json.load(f)

    # Generate plots
    print("Generating visualizations...")
    plot_files = []

    # Plot 1: APY vs Bonding Ratio
    apy_plot = output_dir / "apy_vs_bonding_ratio.png"
    plot_apy_vs_bonding_ratio(results, apy_plot)
    plot_files.append(apy_plot)

    # Plot 2: Annual Provisions
    provisions_plot = output_dir / "annual_provisions.png"
    plot_provisions_comparison(results, provisions_plot)
    plot_files.append(provisions_plot)

    # Generate HTML report
    print("Generating HTML report...")
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    report_path = output_dir / f"economic_report_{timestamp}.html"
    generate_html_report(results, summary_stats, plot_files, report_path, metadata)

    print()
    print("=" * 80)
    print(f"Report generation complete!")
    print(f"Report saved to: {report_path}")
    print(f"Open in browser: file://{report_path.absolute()}")


if __name__ == "__main__":
    main()
