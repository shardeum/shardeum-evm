#!/usr/bin/env python3
"""
Create split account file from old genesis addresses.
Usage: python merge_genesis.py <addresses.json> <main_genesis_file.json> [--balance-multiplier N] [--network NAME]

This script creates a new split account file that will be automatically
merged when the network starts, instead of modifying the main genesis file.

Options:
  --balance-multiplier N    Multiply all balances by N (default: 1)
  --network NAME           Network name for split files (overrides auto-detection)
"""

import json
import sys
import argparse
import glob
import os
from pathlib import Path
from typing import Dict, Set, Optional

# Bech32 encoding constants (from generate_cosmos_address.py)
CHARSET = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
GEN = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3]

def bech32_polymod(values):
    """Calculate bech32 polymod checksum"""
    chk = 1
    for value in values:
        top = chk >> 25
        chk = (chk & 0x1ffffff) << 5 ^ value
        for i in range(5):
            chk ^= GEN[i] if ((top >> i) & 1) else 0
    return chk

def convertbits(data, frombits, tobits, pad=True):
    """Convert between bit groups"""
    acc = 0
    bits = 0
    ret = []
    maxv = (1 << tobits) - 1
    max_acc = (1 << (frombits + tobits - 1)) - 1
    for value in data:
        if value < 0 or (value >> frombits):
            return None
        acc = ((acc << frombits) | value) & max_acc
        bits += frombits
        while bits >= tobits:
            bits -= tobits
            ret.append((acc >> bits) & maxv)
    if pad:
        if bits:
            ret.append((acc << (tobits - bits)) & maxv)
    elif bits >= frombits or ((acc << (tobits - bits)) & maxv):
        return None
    return ret

def eth_to_cosmos_address(eth_addr_hex: str) -> Optional[str]:
    """Convert Ethereum address to Cosmos bech32 address"""
    try:
        # Remove 0x prefix if present
        if eth_addr_hex.startswith('0x'):
            eth_addr_hex = eth_addr_hex[2:]
        
        eth_bytes = bytes.fromhex(eth_addr_hex)
        conv = convertbits(eth_bytes, 8, 5)
        if conv is None:
            return None
        
        hrp = 'shardeum'
        hrp_values = [ord(x) >> 5 for x in hrp] + [0] + [ord(x) & 31 for x in hrp]
        checksum_input = hrp_values + conv + [0, 0, 0, 0, 0, 0]
        polymod = bech32_polymod(checksum_input) ^ 1
        checksum = [(polymod >> 5 * (5 - i)) & 31 for i in range(6)]
        
        combined = conv + checksum
        return hrp + '1' + ''.join([CHARSET[d] for d in combined])
    except Exception as e:
        print(f"Error converting address {eth_addr_hex}: {e}", file=sys.stderr)
        return None

def load_old_genesis(filepath: str) -> Dict[str, str]:
    """Load old genesis file with Ethereum addresses and balances"""
    try:
        with open(filepath, 'r') as f:
            return json.load(f)
    except Exception as e:
        print(f"Error loading old genesis file {filepath}: {e}", file=sys.stderr)
        sys.exit(1)

def load_new_genesis(filepath: str) -> Dict:
    """Load new genesis file in Cosmos SDK format"""
    try:
        with open(filepath, 'r') as f:
            return json.load(f)
    except Exception as e:
        print(f"Error loading new genesis file {filepath}: {e}", file=sys.stderr)
        sys.exit(1)

def get_existing_addresses(genesis_data: Dict) -> tuple[Set[str], Set[str]]:
    """Get sets of existing Ethereum and Cosmos addresses from new genesis file"""
    existing_eth_addresses = set()
    existing_cosmos_addresses = set()
    
    # Check EVM accounts for ethereum addresses
    evm_accounts = genesis_data.get('app_state', {}).get('evm', {}).get('accounts', [])
    for account in evm_accounts:
        eth_addr = account.get('address', '')
        if eth_addr:
            existing_eth_addresses.add(eth_addr.lower())
    
    # Check auth accounts for cosmos addresses
    auth_accounts = genesis_data.get('app_state', {}).get('auth', {}).get('accounts', [])
    for account in auth_accounts:
        cosmos_addr = account.get('address', '')
        if cosmos_addr and cosmos_addr.startswith('shardeum'):
            existing_cosmos_addresses.add(cosmos_addr)
    
    # Check bank balances for cosmos addresses
    bank_balances = genesis_data.get('app_state', {}).get('bank', {}).get('balances', [])
    for balance in bank_balances:
        cosmos_addr = balance.get('address', '')
        if cosmos_addr and cosmos_addr.startswith('shardeum'):
            existing_cosmos_addresses.add(cosmos_addr)
    
    return existing_eth_addresses, existing_cosmos_addresses

def create_evm_account(eth_address: str, balance_wei: str) -> Dict:
    """Create EVM account structure for new genesis format (UNUSED - EVM accounts are created at runtime)"""
    return {
        "address": eth_address,
        "code": "",
        "storage": {},
        "balance": balance_wei
    }

def create_auth_account(cosmos_address: str) -> Dict:
    """Create auth account structure for new genesis format"""
    return {
        "@type": "/cosmos.auth.v1beta1.BaseAccount",
        "address": cosmos_address,
        "pub_key": None,
        "sequence": "0"
    }

def create_bank_balance(cosmos_address: str, balance_wei: str) -> Dict:
    """Create bank balance structure for new genesis format"""
    return {
        "address": cosmos_address,
        "coins": [
            {
                "denom": "ashm",
                "amount": balance_wei
            }
        ]
    }

def find_next_split_number(genesis_dir: str, base_name: str) -> int:
    """Find the next sequential number for split account files"""
    pattern = os.path.join(genesis_dir, f"{base_name}.genesis.accounts.*.json")
    existing_files = glob.glob(pattern)

    if not existing_files:
        return 1

    # Extract numbers from existing files
    numbers = []
    for file_path in existing_files:
        try:
            # Extract number from filename like "network.genesis.accounts.3.json"
            filename = os.path.basename(file_path)
            number_part = filename.split('.')[-2]  # Get the part before .json
            numbers.append(int(number_part))
        except (ValueError, IndexError):
            continue

    return max(numbers) + 1 if numbers else 1

def get_existing_addresses_from_all_sources(genesis_dir: str, base_name: str, main_genesis: Dict) -> tuple[Set[str], Set[str]]:
    """Get existing addresses from main genesis and all split files"""
    existing_eth_addresses, existing_cosmos_addresses = get_existing_addresses(main_genesis)

    # Also check split files for existing addresses
    pattern = os.path.join(genesis_dir, f"{base_name}.genesis.accounts.*.json")
    split_files = glob.glob(pattern)

    for split_file in split_files:
        try:
            with open(split_file, 'r') as f:
                split_data = json.load(f)

            # Check accounts for cosmos addresses
            for account in split_data.get('accounts', []):
                cosmos_addr = account.get('address', '')
                if cosmos_addr and cosmos_addr.startswith('shardeum'):
                    existing_cosmos_addresses.add(cosmos_addr)

            # Check balances for cosmos addresses
            for balance in split_data.get('balances', []):
                cosmos_addr = balance.get('address', '')
                if cosmos_addr and cosmos_addr.startswith('shardeum'):
                    existing_cosmos_addresses.add(cosmos_addr)

        except Exception as e:
            print(f"Warning: Could not read split file {split_file}: {e}", file=sys.stderr)

    print(f"Found {len(existing_eth_addresses)} existing Ethereum addresses and {len(existing_cosmos_addresses)} existing Cosmos addresses")
    return existing_eth_addresses, existing_cosmos_addresses

def update_main_genesis_supply(main_genesis_path: str, added_balance: int) -> bool:
    """Update the total supply in the main genesis file"""
    try:
        with open(main_genesis_path, 'r') as f:
            genesis = json.load(f)

        # Get current supply
        supply_section = genesis.get('app_state', {}).get('bank', {}).get('supply', [])

        # Find ashm supply entry
        ashm_supply = None
        for supply_entry in supply_section:
            if supply_entry.get('denom') == 'ashm':
                ashm_supply = supply_entry
                break

        if ashm_supply:
            current_amount = int(ashm_supply.get('amount', '0'))
            new_amount = current_amount + added_balance
            ashm_supply['amount'] = str(new_amount)
            print(f"Updated total supply from {current_amount:,} to {new_amount:,} ashm")
        else:
            # Add new ashm supply entry if it doesn't exist
            supply_section.append({
                'denom': 'ashm',
                'amount': str(added_balance)
            })
            print(f"Added new ashm supply entry: {added_balance:,} ashm")

        # Save updated genesis
        with open(main_genesis_path, 'w') as f:
            json.dump(genesis, f, indent=2)

        return True

    except Exception as e:
        print(f"Error updating main genesis supply: {e}", file=sys.stderr)
        return False

def merge_genesis_files(old_genesis_path: str, new_genesis_path: str, balance_multiplier: int = 1, network: str = None):
    """Create split account file from old genesis addresses"""
    print(f"Loading old genesis from {old_genesis_path}")
    old_genesis = load_old_genesis(old_genesis_path)

    print(f"Loading main genesis from {new_genesis_path}")
    main_genesis = load_new_genesis(new_genesis_path)

    print(f"Using balance multiplier: {balance_multiplier}")

    # Determine base name and directory for split files
    genesis_path = Path(new_genesis_path)
    genesis_dir = str(genesis_path.parent)

    if network:
        base_name = network
        print(f"Using network override: {network}")
    else:
        base_name = genesis_path.stem.replace('.genesis', '')
        if base_name == 'genesis':
            base_name = 'network'
        print(f"Auto-detected base name: {base_name}")

    # Get existing addresses from ALL sources (main genesis + split files)
    existing_eth_addresses, existing_cosmos_addresses = get_existing_addresses_from_all_sources(genesis_dir, base_name, main_genesis)

    # Prepare data for new split file
    new_accounts = []
    new_balances = []
    added_count = 0
    skipped_count = 0
    total_added_balance = 0
    
    print(f"Processing {len(old_genesis)} addresses from old genesis")
    
    for eth_address, balance_data in old_genesis.items():
        # Normalize ethereum address
        eth_addr_normalized = eth_address.lower()
        
        # Convert to cosmos address first to check for duplicates
        cosmos_address = eth_to_cosmos_address(eth_address)
        if not cosmos_address:
            print(f"Failed to convert address: {eth_address}")
            skipped_count += 1
            continue
        
        # Check if either ethereum or cosmos address already exists
        if eth_addr_normalized in existing_eth_addresses:
            print(f"Skipping duplicate Ethereum address: {eth_address}")
            skipped_count += 1
            continue
            
        if cosmos_address in existing_cosmos_addresses:
            print(f"Skipping duplicate Cosmos address: {cosmos_address} (from Ethereum {eth_address})")
            skipped_count += 1
            continue
        
        # Get balance in wei
        balance_wei = balance_data.get('wei', '0')
        balance_int = int(balance_wei)

        # Apply balance multiplier
        balance_int = balance_int * balance_multiplier
        balance_wei = str(balance_int)

        # Skip addresses with 0 balance
        if balance_int == 0:
            print(f"Skipping address with 0 balance: {eth_address}")
            skipped_count += 1
            continue

        # Create auth account and bank balance for the split file
        auth_account = create_auth_account(cosmos_address)
        bank_balance = create_bank_balance(cosmos_address, balance_wei)

        # Add to new split file data
        new_accounts.append(auth_account)
        new_balances.append(bank_balance)

        # Update existing addresses sets
        existing_eth_addresses.add(eth_addr_normalized)
        existing_cosmos_addresses.add(cosmos_address)
        total_added_balance += balance_int
        added_count += 1

        if balance_multiplier != 1:
            original_balance = balance_data.get('wei', '0')
            print(f"Will add address: {eth_address} -> {cosmos_address} (balance: {original_balance} -> {balance_wei} wei, multiplier: {balance_multiplier})")
        else:
            print(f"Will add address: {eth_address} -> {cosmos_address} (balance: {balance_wei} wei)")
    
    if added_count == 0:
        print("No new addresses to add. No split file created.")
        return

    # Sort accounts and balances for consistency
    new_accounts.sort(key=lambda x: x['address'])
    new_balances.sort(key=lambda x: x['address'])

    # Create split file data
    split_data = {
        'accounts': new_accounts,
        'balances': new_balances
    }

    # Find next split file number and create the file
    next_number = find_next_split_number(genesis_dir, base_name)
    split_file_path = os.path.join(genesis_dir, f"{base_name}.genesis.accounts.{next_number}.json")

    print(f"\nCreating split file:")
    print(f"  Added: {added_count} addresses")
    print(f"  Skipped: {skipped_count} addresses (duplicates or conversion errors)")
    if balance_multiplier != 1:
        print(f"  Balance multiplier: {balance_multiplier}")
    print(f"  Total added balance: {total_added_balance:,} ashm")

    # Save split file
    try:
        with open(split_file_path, 'w') as f:
            json.dump(split_data, f, indent=2)
        print(f"Successfully created split file: {split_file_path}")
    except Exception as e:
        print(f"Error creating split file: {e}", file=sys.stderr)
        sys.exit(1)

    # Update main genesis supply
    if total_added_balance > 0:
        print(f"Updating main genesis supply...")
        if not update_main_genesis_supply(new_genesis_path, total_added_balance):
            print(f"Warning: Failed to update supply in main genesis", file=sys.stderr)
        else:
            print(f"Successfully updated supply in main genesis")

    print(f"\nSplit file creation complete!")
    print(f"The new accounts will be automatically included when the network starts.")

def main():
    parser = argparse.ArgumentParser(description="Create split account file from old genesis addresses")
    parser.add_argument("old_genesis", help="Path to old genesis file (simple format)")
    parser.add_argument("main_genesis", help="Path to main genesis file (Cosmos SDK format)")
    parser.add_argument("--balance-multiplier", type=int, default=1,
                       help="Multiplier to apply to all balances (default: 1)")
    parser.add_argument("--network", type=str, default=None,
                       help="Network name to use for split files (overrides auto-detection)")

    args = parser.parse_args()

    merge_genesis_files(args.old_genesis, args.main_genesis, args.balance_multiplier, args.network)

if __name__ == "__main__":
    main()
