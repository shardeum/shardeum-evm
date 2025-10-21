#!/usr/bin/env python3
"""
Network Parameter Validation Script

This script compares expected network parameters (from expected_params.json)
against actual parameters from a running Shardeum node via RPC.

Usage:
    python scripts/validate_network_params.py <RPC_URL> [EXPECTED_PARAMS_FILE]
    python scripts/validate_network_params.py http://localhost:1317
    python scripts/validate_network_params.py http://localhost:1317 scripts/expected_params.json
    python scripts/validate_network_params.py http://localhost:1317 custom_params.json
"""

import json
import sys
import requests
from pathlib import Path
from typing import Any, Dict, List, Tuple


class ParamValidator:
    """Validates network parameters against RPC responses."""

    def __init__(self, rpc_url: str, expected_params_file: str = "scripts/expected_params.json"):
        """
        Initialize the validator.

        Args:
            rpc_url: Base URL for the RPC server (e.g., http://localhost:1317)
            expected_params_file: Path to expected_params.json
        """
        self.rpc_url = rpc_url.rstrip('/')
        self.expected_params_file = expected_params_file
        self.results = {}

    def load_expected_params(self) -> Dict[str, Any]:
        """Load expected parameters from JSON file."""
        try:
            with open(self.expected_params_file, 'r') as f:
                return json.load(f)
        except FileNotFoundError:
            print(f"Error: Could not find {self.expected_params_file}")
            sys.exit(1)
        except json.JSONDecodeError as e:
            print(f"Error: Invalid JSON in {self.expected_params_file}: {e}")
            sys.exit(1)

    def fetch_params(self, module: str, path: str) -> Dict[str, Any]:
        """
        Fetch parameters from the RPC server.

        Args:
            module: Module name (for error reporting)
            path: API endpoint path

        Returns:
            The params object from the response
        """
        url = f"{self.rpc_url}{path}"
        try:
            response = requests.get(url, timeout=10)
            response.raise_for_status()
            data = response.json()

            # Extract params object from response
            if 'params' in data:
                return data['params']
            else:
                print(f"Warning: No 'params' field in response for {module}")
                return {}

        except requests.exceptions.ConnectionError:
            print(f"Error: Could not connect to {self.rpc_url}")
            sys.exit(1)
        except requests.exceptions.Timeout:
            print(f"Error: Request to {module} timed out")
            return {}
        except requests.exceptions.HTTPError as e:
            print(f"Error: HTTP {e.response.status_code} for {module}: {e}")
            return {}
        except json.JSONDecodeError:
            print(f"Error: Invalid JSON response for {module}")
            return {}

    def compare_values(self, expected: Any, actual: Any) -> Tuple[bool, str]:
        """
        Compare expected and actual values, handling type conversions.

        Args:
            expected: Expected value
            actual: Actual value from RPC

        Returns:
            Tuple of (match: bool, description: str)
        """
        # Handle None cases
        if expected is None and actual is None:
            return True, "match"

        # Try string comparison (for numeric strings vs actual numbers)
        expected_str = str(expected)
        actual_str = str(actual)

        if expected_str == actual_str:
            return True, "match"

        return False, f"expected {expected!r}, got {actual!r}"

    def deep_compare(self, expected: Any, actual: Any, path: str = "") -> Tuple[bool, List[str]]:
        """
        Recursively compare expected and actual parameter structures.

        Args:
            expected: Expected parameter value/structure
            actual: Actual parameter value/structure from RPC
            path: Current path in the structure (for error reporting)

        Returns:
            Tuple of (all_match: bool, differences: List[str])
        """
        differences = []
        all_match = True

        # Handle dict/object comparison
        if isinstance(expected, dict):
            if not isinstance(actual, dict):
                differences.append(f"{path}: expected object, got {type(actual).__name__}")
                return False, differences

            # Check for missing keys
            for key in expected:
                new_path = f"{path}.{key}" if path else key
                if key not in actual:
                    differences.append(f"{new_path}: MISSING in actual response")
                    all_match = False
                else:
                    # Recursively compare nested values
                    match, diffs = self.deep_compare(expected[key], actual[key], new_path)
                    if not match:
                        all_match = False
                        differences.extend(diffs)

            # Optionally report extra keys in actual (not in expected)
            # This is informational and doesn't fail the comparison
            for key in actual:
                if key not in expected:
                    new_path = f"{path}.{key}" if path else key
                    differences.append(f"{new_path}: EXTRA in actual response (not in expected)")

        # Handle list comparison
        elif isinstance(expected, list):
            if not isinstance(actual, list):
                differences.append(f"{path}: expected array, got {type(actual).__name__}")
                return False, differences

            if len(expected) != len(actual):
                differences.append(f"{path}: expected list of length {len(expected)}, got {len(actual)}")
                all_match = False

            # Compare elements
            for i, (exp_item, act_item) in enumerate(zip(expected, actual)):
                new_path = f"{path}[{i}]"
                match, diffs = self.deep_compare(exp_item, act_item, new_path)
                if not match:
                    all_match = False
                    differences.extend(diffs)

        # Handle primitive value comparison
        else:
            match, description = self.compare_values(expected, actual)
            if not match:
                all_match = False
                differences.append(f"{path}: {description}")

        return all_match, differences

    def validate(self) -> int:
        """
        Run validation against all modules.

        Returns:
            Exit code (0 for success, 1 for differences found)
        """
        expected_params = self.load_expected_params()

        print("=" * 80)
        print("Network Parameter Validation")
        print("=" * 80)
        print(f"RPC URL: {self.rpc_url}")
        print(f"Expected params file: {self.expected_params_file}\n")

        total_modules = len(expected_params)
        passed_modules = 0
        failed_modules = 0

        # Validate each module
        for module, config in expected_params.items():
            path = config.get('path', '')
            expected = config.get('params', {})

            print(f"Validating {module:20} → {path}")

            # Fetch actual params from RPC
            actual = self.fetch_params(module, path)

            # Compare
            all_match, differences = self.deep_compare(expected, actual)

            if all_match:
                print(f"  ✓ PASS\n")
                passed_modules += 1
            else:
                print(f"  ✗ FAIL")
                for diff in differences:
                    print(f"    • {diff}")
                print()
                failed_modules += 1

            self.results[module] = {
                'passed': all_match,
                'differences': differences
            }

        # Summary
        print("=" * 80)
        print(f"Results: {passed_modules}/{total_modules} modules passed")
        if failed_modules > 0:
            print(f"Failed modules: {failed_modules}")
            return 1
        else:
            print("All parameters validated successfully!")
            return 0


def print_help():
    """Print comprehensive help message."""
    help_text = """
Network Parameter Validation Script

DESCRIPTION:
  This script validates network parameters from a running Shardeum node against
  expected parameter values. It compares actual RPC responses with values defined
  in an expected parameters JSON file and reports any differences.

USAGE:
  python scripts/validate_network_params.py <RPC_URL> [EXPECTED_PARAMS_FILE]
  python scripts/validate_network_params.py --help
  python scripts/validate_network_params.py -h

ARGUMENTS:
  RPC_URL                 URL of the RPC endpoint (e.g., http://localhost:1317)
                          Required for validation, not needed with --help/-h

  EXPECTED_PARAMS_FILE    Path to JSON file containing expected parameters
                          Optional - defaults to scripts/expected_params.json
                          Use absolute or relative paths from project root

EXAMPLES:
  # Validate local node using default expected parameters
  python scripts/validate_network_params.py http://localhost:1317

  # Validate with explicitly specified parameters file
  python scripts/validate_network_params.py http://localhost:1317 scripts/expected_params.json

  # Validate with custom parameters file
  python scripts/validate_network_params.py http://localhost:1317 custom_params.json

  # Validate testnet with testnet-specific parameters
  python scripts/validate_network_params.py http://testnet-rpc:1317 testnet_params.json

  # Display this help message
  python scripts/validate_network_params.py --help

OUTPUT:
  The script displays validation results for each module with:
  - ✓ PASS  : All parameters match expected values
  - ✗ FAIL  : One or more parameters differ from expected

  Detailed differences are listed with:
  - MISSING: Parameters expected but not found in response
  - Different values shown as "expected X, got Y"
  - EXTRA: Parameters in response not in expected (informational)

EXIT CODES:
  0  : All modules validated successfully
  1  : One or more validation failures or errors occurred

REQUIREMENTS:
  - Python 3.x
  - requests library (pip install requests)
  - Network access to RPC endpoint
  - Valid JSON parameters file
"""
    print(help_text)


def main():
    """Main entry point."""
    # Check for help argument
    if len(sys.argv) > 1 and sys.argv[1] in ('--help', '-h'):
        print_help()
        sys.exit(0)

    if len(sys.argv) < 2:
        print("Usage: python scripts/validate_network_params.py <RPC_URL> [EXPECTED_PARAMS_FILE]")
        print("\nExamples:")
        print("  python scripts/validate_network_params.py http://localhost:1317")
        print("  python scripts/validate_network_params.py http://localhost:1317 scripts/expected_params.json")
        print("  python scripts/validate_network_params.py http://localhost:1317 custom_params.json")
        print("\nFor more information, use --help")
        sys.exit(1)

    rpc_url = sys.argv[1]
    expected_params_file = sys.argv[2] if len(sys.argv) > 2 else "scripts/expected_params.json"

    validator = ParamValidator(rpc_url, expected_params_file)
    exit_code = validator.validate()
    sys.exit(exit_code)


if __name__ == "__main__":
    main()
