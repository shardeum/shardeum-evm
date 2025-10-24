#!/usr/bin/env python3
"""
Interactive Menu for Economic Simulation

User-friendly interface for non-technical users to configure and run
economic parameter simulations without command-line knowledge.
"""

import sys
import json
from pathlib import Path
from typing import Dict, Any, Optional, Tuple


# Preset configurations
PRESETS = {
    'quick': {
        'name': 'Quick Test',
        'description': 'Fast test with minimal parameter combinations',
        'duration': '~5 minutes',
        'num_points': 5,
        'num_bonding_ratios': 5,
        'bonding_ratio_min': 0.10,
        'bonding_ratio_max': 0.90,
        'validate': False,
        'min_inflation_min': 0.05,
        'min_inflation_max': 0.10,
        'max_inflation_min': 0.15,
        'max_inflation_max': 0.25,
        'goal_bonded_min': 0.50,
        'goal_bonded_max': 0.80,
    },
    'standard': {
        'name': 'Standard Analysis',
        'description': 'Comprehensive analysis with recommended settings',
        'duration': '~15 minutes',
        'num_points': 10,
        'num_bonding_ratios': 10,
        'bonding_ratio_min': 0.05,
        'bonding_ratio_max': 0.95,
        'validate': False,
        'min_inflation_min': 0.03,
        'min_inflation_max': 0.15,
        'max_inflation_min': 0.10,
        'max_inflation_max': 0.40,
        'goal_bonded_min': 0.20,
        'goal_bonded_max': 0.95,
    },
    'comprehensive': {
        'name': 'Comprehensive Study',
        'description': 'Extensive analysis with network validation',
        'duration': '~45 minutes',
        'num_points': 15,
        'num_bonding_ratios': 15,
        'bonding_ratio_min': 0.01,
        'bonding_ratio_max': 0.99,
        'validate': True,
        'num_validations': 20,
        'min_inflation_min': 0.03,
        'min_inflation_max': 0.15,
        'max_inflation_min': 0.10,
        'max_inflation_max': 0.40,
        'goal_bonded_min': 0.10,
        'goal_bonded_max': 0.99,
    },
    'current': {
        'name': 'Current Network Settings',
        'description': 'Test only current network parameters',
        'duration': '~2 minutes',
        'num_points': 1,
        'num_bonding_ratios': 10,
        'bonding_ratio_min': 0.05,
        'bonding_ratio_max': 0.95,
        'validate': False,
        'min_inflation_min': 0.07,
        'min_inflation_max': 0.07,
        'max_inflation_min': 0.20,
        'max_inflation_max': 0.20,
        'goal_bonded_min': 0.67,
        'goal_bonded_max': 0.67,
    }
}


def clear_screen():
    """Clear terminal screen (works on Unix/Linux)"""
    print("\033[2J\033[H", end="")


def print_header(title: str):
    """Print formatted header"""
    print("\n" + "=" * 70)
    print(f"  {title}")
    print("=" * 70 + "\n")


def print_section(title: str):
    """Print section divider"""
    print(f"\n{'─' * 70}")
    print(f"  {title}")
    print('─' * 70)


def print_success(message: str):
    """Print success message"""
    print(f"✓ {message}")


def print_error(message: str):
    """Print error message"""
    print(f"✗ {message}")


def print_info(message: str):
    """Print info message"""
    print(f"ℹ {message}")


def get_input(prompt: str, default: Optional[str] = None) -> str:
    """Get user input with optional default"""
    if default:
        full_prompt = f"{prompt} [{default}]: "
    else:
        full_prompt = f"{prompt}: "

    response = input(full_prompt).strip()
    if not response and default:
        return default
    return response


def get_yes_no(prompt: str, default: bool = True) -> bool:
    """Get yes/no input from user"""
    default_str = "Y/n" if default else "y/N"
    response = get_input(f"{prompt} ({default_str})", "y" if default else "n")

    if response.lower() in ['y', 'yes']:
        return True
    elif response.lower() in ['n', 'no']:
        return False
    else:
        return default


def get_percentage(prompt: str, min_val: float = 0, max_val: float = 100,
                   default: Optional[float] = None) -> float:
    """Get percentage input from user"""
    while True:
        default_str = f"{default:.1f}" if default else None
        response = get_input(prompt, default_str)

        try:
            value = float(response)
            if min_val <= value <= max_val:
                return value / 100.0  # Convert to decimal
            else:
                print_error(f"Please enter a value between {min_val} and {max_val}")
        except ValueError:
            print_error("Please enter a valid number")


def get_integer(prompt: str, min_val: int = 1, max_val: int = 100,
               default: Optional[int] = None) -> int:
    """Get integer input from user"""
    while True:
        default_str = str(default) if default else None
        response = get_input(prompt, default_str)

        try:
            value = int(response)
            if min_val <= value <= max_val:
                return value
            else:
                print_error(f"Please enter a value between {min_val} and {max_val}")
        except ValueError:
            print_error("Please enter a valid integer")


def show_welcome():
    """Display welcome screen"""
    clear_screen()
    print_header("Economic Simulation Tool")

    print("This tool helps you understand how different economic parameters affect:\n")
    print("  • Validator rewards (APY - Annual Percentage Yield)")
    print("  • Token inflation rate")
    print("  • New tokens minted per year")
    print()
    print("The simulation tests different combinations of parameters to find")
    print("optimal settings for your network's economic goals.")
    print()


def show_parameter_help():
    """Show help about parameters"""
    clear_screen()
    print_header("Parameter Guide")

    print("📊 INFLATION PARAMETERS\n")
    print("  Minimum Inflation:")
    print("    The lowest annual inflation rate the network can have.")
    print("    Lower = less new tokens created")
    print("    Current network: 7%\n")

    print("  Maximum Inflation:")
    print("    The highest annual inflation rate the network can have.")
    print("    Higher = more rewards to encourage staking")
    print("    Current network: 20%\n")

    print("  Goal Bonded Ratio:")
    print("    Target percentage of tokens that should be staked.")
    print("    If actual staking is below this, inflation increases")
    print("    If actual staking is above this, inflation decreases")
    print("    Current network: 67%")
    print("    Can test from 1% to 99%\n")

    print("\n🎯 SIMULATION PARAMETERS\n")
    print("  Number of Points:")
    print("    How many different values to test for each parameter.")
    print("    More points = more accurate but slower")
    print("    Example: 10 points means testing 10 different min inflation values\n")

    print("  Bonding Ratios:")
    print("    How many different actual staking percentages to test.")
    print("    Can test from 1% to 99% actual staking")
    print("    Tests scenarios from very low to very high staking\n")

    print("\n💡 QUICK TIPS\n")
    print("  • Start with 'Standard Analysis' for most use cases")
    print("  • Use 'Quick Test' to verify setup before long runs")
    print("  • 'Comprehensive Study' includes network validation (slower but accurate)")
    print()
    input("Press Enter to return to main menu...")


def show_preset_menu() -> Optional[str]:
    """Show preset selection menu and return choice"""
    print_section("Select Simulation Mode")

    print("\n1) Quick Test")
    print(f"   {PRESETS['quick']['description']}")
    print(f"   Duration: {PRESETS['quick']['duration']}")
    print(f"   Scenarios: ~{PRESETS['quick']['num_points']**3 * PRESETS['quick']['num_bonding_ratios']}\n")

    print("2) Standard Analysis (RECOMMENDED)")
    print(f"   {PRESETS['standard']['description']}")
    print(f"   Duration: {PRESETS['standard']['duration']}")
    print(f"   Scenarios: ~{PRESETS['standard']['num_points']**3 * PRESETS['standard']['num_bonding_ratios']}\n")

    print("3) Comprehensive Study")
    print(f"   {PRESETS['comprehensive']['description']}")
    print(f"   Duration: {PRESETS['comprehensive']['duration']}")
    print(f"   Scenarios: ~{PRESETS['comprehensive']['num_points']**3 * PRESETS['comprehensive']['num_bonding_ratios']}\n")

    print("4) Current Network Settings")
    print(f"   {PRESETS['current']['description']}")
    print(f"   Duration: {PRESETS['current']['duration']}\n")

    print("5) Custom Parameters (Advanced)")
    print("   Manually configure all parameters\n")

    print("6) Help - Explain Parameters")
    print("   Learn what each parameter means\n")

    print("0) Exit\n")

    choice = get_input("Enter choice [0-6]", "2")

    if choice == '1':
        return 'quick'
    elif choice == '2':
        return 'standard'
    elif choice == '3':
        return 'comprehensive'
    elif choice == '4':
        return 'current'
    elif choice == '5':
        return 'custom'
    elif choice == '6':
        return 'help'
    elif choice == '0':
        return 'exit'
    else:
        print_error("Invalid choice. Please select 0-6.")
        return None


def get_custom_parameters() -> Dict[str, Any]:
    """Get custom parameters from user"""
    clear_screen()
    print_header("Custom Parameter Configuration")

    print("\n📊 INFLATION RATE RANGE\n")
    print("You'll define the range of inflation rates to test.\n")

    print("Minimum Inflation (the floor):")
    print("  What's the LOWEST inflation rate to test?")
    print("  Suggested: 3% to 10% | Current network: 7%")
    min_inf_min = get_percentage("  Enter minimum (%)", 0, 50, 3.0)

    print("\n  What's the HIGHEST value for minimum inflation?")
    print("  (Must be >= the lowest value you just entered)")
    min_inf_max = get_percentage("  Enter maximum (%)", min_inf_min * 100, 50, 15.0)

    print("\n\nMaximum Inflation (the ceiling):")
    print("  What's the LOWEST value for maximum inflation to test?")
    print("  Suggested: 10% to 40% | Current network: 20%")
    print("  (Must be >= maximum minimum inflation)")
    max_inf_min = get_percentage("  Enter minimum (%)", min_inf_max * 100, 100, 10.0)

    print("\n  What's the HIGHEST value for maximum inflation?")
    max_inf_max = get_percentage("  Enter maximum (%)", max_inf_min * 100, 100, 40.0)

    print("\n\n🎯 BONDING TARGET RANGE\n")
    print("Goal Bonded Ratio (target staking percentage):")
    print("  This is the network's TARGET - if actual staking is below this,")
    print("  inflation increases. If above, inflation decreases.")
    print()
    print("  What's the LOWEST target to test?")
    print("  Suggested: 30% to 70% | Current network: 67%")
    goal_min = get_percentage("  Enter minimum (%)", 1, 98, 40.0)

    print("\n  What's the HIGHEST target to test?")
    print("  Suggested: 60% to 95%")
    goal_max = get_percentage("  Enter maximum (%)", goal_min * 100, 99, 90.0)

    print("\n\n⚙️  SIMULATION DETAIL\n")
    print("Number of test points per parameter:")
    print("  More points = more scenarios tested = longer runtime")
    print("  Suggested: 5 (quick) to 15 (thorough)")
    num_points = get_integer("  Enter number of points", 3, 20, 10)

    print("\n\n🎲 BONDING RATIO TESTING\n")
    print("Bonding ratio is the percentage of tokens staked in the network.")
    print()
    print("What's the LOWEST bonding ratio to test?")
    print("  Lower values = testing scenarios with less staking")
    print("  Suggested: 1% (very low staking) to 30% (low staking)")
    bonding_min = get_percentage("  Enter minimum (%)", 1, 98, 5.0)

    print("\nWhat's the HIGHEST bonding ratio to test?")
    print("  Higher values = testing scenarios with heavy staking")
    print("  Suggested: 70% to 99%")
    bonding_max = get_percentage("  Enter maximum (%)", bonding_min * 100, 99, 95.0)

    print("\nHow many bonding ratios to test between these values?")
    print("  More points = more scenarios = longer runtime")
    print(f"  Will test from {bonding_min:.0%} to {bonding_max:.0%}")
    num_bonding = get_integer("  Enter number of bonding ratios", 5, 30, 10)

    print("\n\n🔍 NETWORK VALIDATION\n")
    print("Validate results against actual test networks?")
    print("  This launches real networks to verify simulation accuracy")
    print("  WARNING: Significantly increases runtime (adds 30+ minutes)")
    validate = get_yes_no("  Enable validation?", False)

    num_validations = 0
    if validate:
        print("\nHow many parameter sets to validate?")
        print("  Suggested: 10-20 samples")
        num_validations = get_integer("  Enter number", 5, 30, 15)

    # Calculate estimated scenarios
    total_scenarios = (num_points ** 3) * num_bonding
    est_minutes = total_scenarios / 50  # Rough estimate

    return {
        'name': 'Custom Configuration',
        'description': f'Custom simulation with ~{total_scenarios} scenarios',
        'duration': f'~{int(est_minutes)} minutes',
        'num_points': num_points,
        'num_bonding_ratios': num_bonding,
        'bonding_ratio_min': bonding_min,
        'bonding_ratio_max': bonding_max,
        'validate': validate,
        'num_validations': num_validations if validate else 0,
        'min_inflation_min': min_inf_min,
        'min_inflation_max': min_inf_max,
        'max_inflation_min': max_inf_min,
        'max_inflation_max': max_inf_max,
        'goal_bonded_min': goal_min,
        'goal_bonded_max': goal_max,
    }


def show_configuration_summary(config: Dict[str, Any]) -> bool:
    """Show configuration summary and confirm"""
    print_section("Configuration Summary")

    print(f"\nSimulation: {config['name']}")
    print(f"Description: {config['description']}")
    print(f"Estimated Duration: {config['duration']}\n")

    print("Parameter Ranges:")
    print(f"  • Minimum Inflation: {config['min_inflation_min']:.1%} to {config['min_inflation_max']:.1%}")
    print(f"  • Maximum Inflation: {config['max_inflation_min']:.1%} to {config['max_inflation_max']:.1%}")
    print(f"  • Goal Bonded Ratio: {config['goal_bonded_min']:.1%} to {config['goal_bonded_max']:.1%}")
    print(f"\nSimulation Detail:")
    print(f"  • Test points per parameter: {config['num_points']}")
    print(f"  • Bonding ratios tested: {config['num_bonding_ratios']}")

    # Show bonding ratio range if available
    if 'bonding_ratio_min' in config and 'bonding_ratio_max' in config:
        print(f"  • Bonding ratio range: {config['bonding_ratio_min']:.1%} to {config['bonding_ratio_max']:.1%}")

    print(f"  • Network validation: {'Yes' if config.get('validate', False) else 'No'}")

    if config.get('validate'):
        print(f"  • Validation samples: {config.get('num_validations', 0)}")

    # Calculate total
    if config['num_points'] > 1:
        param_combos = config['num_points'] ** 3
    else:
        param_combos = 1
    total_sims = param_combos * config['num_bonding_ratios']

    print(f"\nTotal Scenarios: ~{total_sims:,}")
    print()

    return get_yes_no("Start simulation with these settings?", True)


def save_config_to_file(config: Dict[str, Any]):
    """Save configuration for shell script to read"""
    config_file = Path("/tmp/economic_sim_config.json")
    with open(config_file, 'w') as f:
        json.dump(config, f, indent=2)
    return config_file


def main():
    """Main interactive menu"""
    show_welcome()

    while True:
        choice = show_preset_menu()

        if choice is None:
            continue

        if choice == 'exit':
            print("\nExiting...")
            sys.exit(0)

        if choice == 'help':
            show_parameter_help()
            show_welcome()
            continue

        # Get configuration
        if choice == 'custom':
            config = get_custom_parameters()
        else:
            config = PRESETS[choice].copy()

        # Show summary and confirm
        if show_configuration_summary(config):
            # Save config for shell script
            config_file = save_config_to_file(config)
            print_success(f"Configuration saved to {config_file}")
            print("\nStarting simulation...")
            print("(This may take a while. You can monitor progress below)\n")

            # Exit successfully - shell script will read config and run
            sys.exit(0)
        else:
            print("\nCancelled. Returning to main menu...\n")
            input("Press Enter to continue...")
            show_welcome()


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\nCancelled by user. Exiting...")
        sys.exit(1)
    except Exception as e:
        print_error(f"An error occurred: {e}")
        sys.exit(1)
