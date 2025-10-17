#!/usr/bin/env python3
"""
Network Data Verification Script for Shardeum

This script verifies that accounts and supply on a live Shardeum network
match the expected genesis values after a restart. It supports:
- Account balance verification against genesis
- Total supply verification with inflation tolerance
- Concurrent RPC queries for performance
- Detailed reporting of mismatches
"""

import json
import asyncio
import argparse
import sys
from pathlib import Path
from typing import Dict, List, Tuple, Optional
from dataclasses import dataclass
import aiohttp
from decimal import Decimal

# Constants
ASHM_DECIMALS = 18
COSMOS_PREFIX = "shardeum"


@dataclass
class VerificationConfig:
    """Configuration for network verification"""
    genesis_file: Path
    rpc_url: str
    inflation_tolerance: float  # Percentage tolerance for supply (e.g., 0.5 for 0.5%)
    concurrency: int  # Number of concurrent RPC requests
    timeout: int  # RPC timeout in seconds
    verbose: bool


@dataclass
class AccountBalance:
    """Account balance information"""
    address: str  # Cosmos address (shardeum1...)
    eth_address: str  # Ethereum address (0x...)
    genesis_balance: int  # Balance in ashm from genesis
    network_balance: Optional[int] = None  # Balance from network query


@dataclass
class VerificationResult:
    """Results of verification"""
    total_accounts: int
    accounts_checked: int
    accounts_matched: int
    accounts_mismatched: int
    accounts_failed: int
    genesis_supply: int
    network_supply: int
    supply_difference: int
    supply_difference_pct: float
    mismatches: List[Tuple[str, int, int]]  # (address, genesis, network)
    failures: List[Tuple[str, str]]  # (address, error)


def cosmos_to_eth_address(cosmos_addr: str) -> str:
    """
    Convert Cosmos bech32 address to Ethereum hex address.

    Note: This is a simplified conversion. In production, you'd use
    proper bech32 decoding and address derivation.
    """
    try:
        # For now, we'll need to use the actual conversion
        # This requires the cosmos SDK or proper bech32 library
        # Placeholder - in reality you'd decode the bech32 properly
        from bech32 import bech32_decode, convertbits

        hrp, data = bech32_decode(cosmos_addr)
        if hrp != COSMOS_PREFIX or data is None:
            raise ValueError(f"Invalid cosmos address: {cosmos_addr}")

        # Convert from 5-bit to 8-bit encoding
        decoded = convertbits(data, 5, 8, False)
        if decoded is None or len(decoded) != 20:
            raise ValueError(f"Invalid address length: {cosmos_addr}")

        # Convert to hex
        eth_addr = "0x" + bytes(decoded).hex()
        return eth_addr
    except ImportError:
        # Fallback: Return a placeholder - user needs to install bech32
        raise ImportError("bech32 library required. Install with: pip install bech32")


def load_genesis_file(genesis_path: Path) -> dict:
    """Load and parse genesis JSON file"""
    print(f"Loading genesis file: {genesis_path}")
    with open(genesis_path, 'r') as f:
        return json.load(f)


def load_account_files(genesis_dir: Path, prefix: str) -> Tuple[List[dict], List[dict]]:
    """
    Load all split account files matching pattern.
    Expected pattern: {prefix}.genesis.accounts.{N}.json

    Returns:
        Tuple of (accounts list, balances list)
    """
    account_files = sorted(genesis_dir.glob(f"{prefix}.genesis.accounts.*.json"))

    if not account_files:
        print("Warning: No split account files found, checking main genesis file")
        return [], []

    print(f"Found {len(account_files)} account files")
    all_accounts = []
    all_balances = []

    for account_file in account_files:
        print(f"  Loading: {account_file.name}")
        with open(account_file, 'r') as f:
            data = json.load(f)
            accounts = data.get('accounts', [])
            balances = data.get('balances', [])
            all_accounts.extend(accounts)
            all_balances.extend(balances)

    return all_accounts, all_balances


def extract_genesis_balances(genesis_data: dict, account_balances_from_files: List[dict]) -> Tuple[List[AccountBalance], int]:
    """
    Extract account balances and total supply from genesis data.

    Args:
        genesis_data: Main genesis file data
        account_balances_from_files: Balances loaded from split account files

    Returns:
        Tuple of (list of AccountBalance objects, total supply in ashm)
    """
    # Get balances from main genesis bank module (may be empty if using split files)
    bank_balances = genesis_data.get('app_state', {}).get('bank', {}).get('balances', [])

    # Create mapping of address to balance
    balance_map: Dict[str, int] = {}

    # Load balances from main genesis file
    for balance_entry in bank_balances:
        address = balance_entry.get('address')
        coins = balance_entry.get('coins', [])
        for coin in coins:
            if coin.get('denom') == 'ashm':
                balance_map[address] = int(coin.get('amount', 0))
                break

    # Load balances from split account files
    for balance_entry in account_balances_from_files:
        address = balance_entry.get('address')
        coins = balance_entry.get('coins', [])
        for coin in coins:
            if coin.get('denom') == 'ashm':
                balance_map[address] = int(coin.get('amount', 0))
                break

    # Get total supply
    supply_list = genesis_data.get('app_state', {}).get('bank', {}).get('supply', [])
    total_supply = 0
    for supply_entry in supply_list:
        if supply_entry.get('denom') == 'ashm':
            total_supply = int(supply_entry.get('amount', 0))
            break

    # Create AccountBalance objects
    account_balances = []
    for address, balance in balance_map.items():
        try:
            eth_addr = cosmos_to_eth_address(address)
            account_balances.append(AccountBalance(
                address=address,
                eth_address=eth_addr,
                genesis_balance=balance
            ))
        except Exception as e:
            print(f"Warning: Could not convert address {address}: {e}")
            continue

    print(f"\nGenesis Summary:")
    print(f"  Total accounts with balances: {len(account_balances)}")
    print(f"  Total supply (ashm): {total_supply}")
    print(f"  Total supply (SHM): {total_supply / (10 ** ASHM_DECIMALS):.2f}")

    return account_balances, total_supply


async def query_balance(session: aiohttp.ClientSession, rpc_url: str, eth_address: str, timeout: int) -> Optional[int]:
    """
    Query account balance from network using eth_getBalance RPC call.

    Returns:
        Balance in ashm, or None if query failed
    """
    payload = {
        "jsonrpc": "2.0",
        "method": "eth_getBalance",
        "params": [eth_address, "latest"],
        "id": 1
    }

    try:
        async with session.post(rpc_url, json=payload, timeout=aiohttp.ClientTimeout(total=timeout)) as response:
            if response.status != 200:
                return None

            data = await response.json()

            if 'error' in data:
                return None

            # Parse hex result to int
            balance_hex = data.get('result', '0x0')
            balance = int(balance_hex, 16)
            return balance

    except Exception as e:
        return None


async def verify_accounts_batch(
    accounts: List[AccountBalance],
    config: VerificationConfig,
    progress_callback=None
) -> List[AccountBalance]:
    """
    Verify account balances by querying the network.
    Uses async batch processing with controlled concurrency.
    """
    semaphore = asyncio.Semaphore(config.concurrency)

    async def verify_one(account: AccountBalance, idx: int) -> AccountBalance:
        async with semaphore:
            async with aiohttp.ClientSession() as session:
                account.network_balance = await query_balance(
                    session,
                    config.rpc_url,
                    account.eth_address,
                    config.timeout
                )

                if progress_callback and (idx + 1) % 100 == 0:
                    progress_callback(idx + 1, len(accounts))

                return account

    tasks = [verify_one(account, idx) for idx, account in enumerate(accounts)]
    return await asyncio.gather(*tasks)


async def query_total_supply(rpc_url: str, account_balances: List[AccountBalance]) -> int:
    """
    Calculate total supply by summing all network balances.
    This is used as a fallback if there's no direct supply query.
    """
    total = 0
    for account in account_balances:
        if account.network_balance is not None:
            total += account.network_balance
    return total


def analyze_results(
    accounts: List[AccountBalance],
    genesis_supply: int,
    inflation_tolerance: float
) -> VerificationResult:
    """
    Analyze verification results and generate report.
    """
    total_accounts = len(accounts)
    accounts_checked = sum(1 for a in accounts if a.network_balance is not None)
    accounts_matched = 0
    accounts_mismatched = 0
    accounts_failed = 0

    mismatches = []
    failures = []
    network_supply = 0

    for account in accounts:
        if account.network_balance is None:
            accounts_failed += 1
            failures.append((account.address, "Failed to query balance"))
        else:
            network_supply += account.network_balance

            if account.network_balance == account.genesis_balance:
                accounts_matched += 1
            else:
                accounts_mismatched += 1
                mismatches.append((
                    account.address,
                    account.genesis_balance,
                    account.network_balance
                ))

    supply_difference = network_supply - genesis_supply
    supply_difference_pct = (supply_difference / genesis_supply * 100) if genesis_supply > 0 else 0

    return VerificationResult(
        total_accounts=total_accounts,
        accounts_checked=accounts_checked,
        accounts_matched=accounts_matched,
        accounts_mismatched=accounts_mismatched,
        accounts_failed=accounts_failed,
        genesis_supply=genesis_supply,
        network_supply=network_supply,
        supply_difference=supply_difference,
        supply_difference_pct=supply_difference_pct,
        mismatches=mismatches,
        failures=failures
    )


def print_report(result: VerificationResult, config: VerificationConfig):
    """Print detailed verification report"""
    print("\n" + "="*80)
    print("VERIFICATION REPORT")
    print("="*80)

    print("\nAccount Verification:")
    print(f"  Total accounts: {result.total_accounts}")
    print(f"  Accounts checked: {result.accounts_checked}")
    print(f"  Accounts matched: {result.accounts_matched} ✓")
    print(f"  Accounts mismatched: {result.accounts_mismatched}" + (" ✗" if result.accounts_mismatched > 0 else ""))
    print(f"  Accounts failed: {result.accounts_failed}" + (" ✗" if result.accounts_failed > 0 else ""))

    if result.accounts_checked > 0:
        success_rate = (result.accounts_matched / result.accounts_checked) * 100
        print(f"  Success rate: {success_rate:.2f}%")

    print("\nSupply Verification:")
    print(f"  Genesis supply: {result.genesis_supply} ashm ({result.genesis_supply / (10 ** ASHM_DECIMALS):.2f} SHM)")
    print(f"  Network supply: {result.network_supply} ashm ({result.network_supply / (10 ** ASHM_DECIMALS):.2f} SHM)")
    print(f"  Difference: {result.supply_difference:+d} ashm ({result.supply_difference / (10 ** ASHM_DECIMALS):+.2f} SHM)")
    print(f"  Difference %: {result.supply_difference_pct:+.6f}%")

    # Check if within tolerance
    within_tolerance = abs(result.supply_difference_pct) <= config.inflation_tolerance
    print(f"  Tolerance: ±{config.inflation_tolerance}% " + ("✓" if within_tolerance else "✗"))

    # Show mismatches if verbose or if there are few
    if config.verbose or (result.accounts_mismatched > 0 and result.accounts_mismatched <= 20):
        if result.mismatches:
            print("\nBalance Mismatches:")
            for address, genesis_bal, network_bal in result.mismatches[:20]:
                diff = network_bal - genesis_bal
                print(f"  {address}")
                print(f"    Genesis:  {genesis_bal} ashm ({genesis_bal / (10 ** ASHM_DECIMALS):.6f} SHM)")
                print(f"    Network:  {network_bal} ashm ({network_bal / (10 ** ASHM_DECIMALS):.6f} SHM)")
                print(f"    Diff:     {diff:+d} ashm ({diff / (10 ** ASHM_DECIMALS):+.6f} SHM)")

            if len(result.mismatches) > 20:
                print(f"  ... and {len(result.mismatches) - 20} more mismatches")

    if config.verbose and result.failures:
        print("\nFailed Queries:")
        for address, error in result.failures[:10]:
            print(f"  {address}: {error}")
        if len(result.failures) > 10:
            print(f"  ... and {len(result.failures) - 10} more failures")

    print("\n" + "="*80)

    # Overall status
    all_matched = result.accounts_mismatched == 0 and result.accounts_failed == 0
    supply_ok = within_tolerance

    if all_matched and supply_ok:
        print("STATUS: ✓ ALL CHECKS PASSED")
    else:
        print("STATUS: ✗ VERIFICATION FAILED")
        if not all_matched:
            print("  - Account balances do not match")
        if not supply_ok:
            print("  - Supply difference exceeds tolerance")

    print("="*80 + "\n")


async def run_verification(config: VerificationConfig) -> int:
    """
    Run the full verification process.

    Returns:
        Exit code (0 for success, 1 for failure)
    """
    try:
        # Load genesis file
        genesis_data = load_genesis_file(config.genesis_file)

        # Determine prefix from genesis file name
        prefix = config.genesis_file.stem.replace('.genesis', '')
        genesis_dir = config.genesis_file.parent

        # Load account files (returns accounts and balances)
        account_data, balance_data = load_account_files(genesis_dir, prefix)

        # Extract balances and supply
        accounts, genesis_supply = extract_genesis_balances(genesis_data, balance_data)

        if not accounts:
            print("Error: No accounts found in genesis")
            return 1

        # Verify accounts against network
        print(f"\nQuerying network at {config.rpc_url}")
        print(f"Concurrency: {config.concurrency} concurrent requests")
        print("Progress: ", end='', flush=True)

        def progress(current, total):
            if current % 100 == 0:
                print(f"{current}/{total}...", end=' ', flush=True)

        verified_accounts = await verify_accounts_batch(accounts, config, progress)
        print("Done!")

        # Analyze results
        result = analyze_results(verified_accounts, genesis_supply, config.inflation_tolerance)

        # Print report
        print_report(result, config)

        # Return exit code
        all_matched = result.accounts_mismatched == 0 and result.accounts_failed == 0
        supply_ok = abs(result.supply_difference_pct) <= config.inflation_tolerance

        return 0 if (all_matched and supply_ok) else 1

    except Exception as e:
        print(f"\nError during verification: {e}", file=sys.stderr)
        if config.verbose:
            import traceback
            traceback.print_exc()
        return 1


def main():
    parser = argparse.ArgumentParser(
        description='Verify Shardeum network data against genesis',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Verify mainnet
  %(prog)s --genesis shardeum-evm/config/environments/mainnet-genesis.genesis.json --rpc http://localhost:8545

  # Verify testnet with custom tolerance
  %(prog)s --genesis shardeum-evm/config/environments/testnet-genesis.genesis.json \\
           --rpc http://localhost:8545 \\
           --tolerance 1.0

  # Quick verification with verbose output
  %(prog)s --genesis config/local-genesis.genesis.json \\
           --rpc http://localhost:8545 \\
           --concurrency 50 \\
           --verbose
        """
    )

    parser.add_argument(
        '--genesis',
        type=Path,
        required=True,
        help='Path to genesis JSON file'
    )

    parser.add_argument(
        '--rpc',
        default='http://localhost:8545',
        help='RPC URL for network queries (default: http://localhost:8545)'
    )

    parser.add_argument(
        '--tolerance',
        type=float,
        default=0.5,
        help='Inflation tolerance percentage (default: 0.5%%)'
    )

    parser.add_argument(
        '--concurrency',
        type=int,
        default=20,
        help='Number of concurrent RPC requests (default: 20)'
    )

    parser.add_argument(
        '--timeout',
        type=int,
        default=10,
        help='RPC request timeout in seconds (default: 10)'
    )

    parser.add_argument(
        '--verbose',
        action='store_true',
        help='Enable verbose output'
    )

    args = parser.parse_args()

    # Validate genesis file exists
    if not args.genesis.exists():
        print(f"Error: Genesis file not found: {args.genesis}", file=sys.stderr)
        return 1

    # Create config
    config = VerificationConfig(
        genesis_file=args.genesis,
        rpc_url=args.rpc,
        inflation_tolerance=args.tolerance,
        concurrency=args.concurrency,
        timeout=args.timeout,
        verbose=args.verbose
    )

    # Run verification
    exit_code = asyncio.run(run_verification(config))
    return exit_code


if __name__ == '__main__':
    sys.exit(main())
