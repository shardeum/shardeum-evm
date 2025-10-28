#!/usr/bin/env python3
"""
Genesis Balance Transfer Script

This script transfers balance from one account to another in genesis files.
It handles both Cosmos bech32 addresses and Ethereum hex addresses with automatic conversion.

Usage:
    python3 scripts/transfer_genesis_balance.py <network> <source_addr> <dest_addr> <amount>

Arguments:
    network      - Type of genesis files (mainnet, testnet, local, devnet)
    source_addr  - Source account address (bech32 or hex format)
    dest_addr    - Destination account address (bech32 or hex format)
    amount       - Amount to transfer in ashm (base denomination)

Examples:
    # Transfer using bech32 addresses
    python3 scripts/transfer_genesis_balance.py mainnet shardeum1abc... shardeum1xyz... 1000000000000000000000

    # Transfer using hex addresses
    python3 scripts/transfer_genesis_balance.py testnet 0x1234... 0x5678... 5000000000000000000000

    # Mix of bech32 and hex addresses
    python3 scripts/transfer_genesis_balance.py local shardeum1abc... 0x5678... 2000000000000000000000
"""

import json
import sys
import os
import glob
from pathlib import Path
from typing import Optional, Tuple, Dict, List

# Bech32 encoding/decoding constants
CHARSET = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
GEN = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3]
HRP = "shardeum"

# ============================================================================
# Address Conversion Functions
# ============================================================================

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

def hex_to_bech32(eth_addr_hex: str) -> Optional[str]:
    """Convert Ethereum hex address to Cosmos bech32 address"""
    try:
        # Remove 0x prefix if present
        if eth_addr_hex.startswith('0x'):
            eth_addr_hex = eth_addr_hex[2:]

        # Pad to 40 characters if needed
        eth_addr_hex = eth_addr_hex.lower().zfill(40)

        eth_bytes = bytes.fromhex(eth_addr_hex)
        conv = convertbits(eth_bytes, 8, 5)
        if conv is None:
            return None

        hrp_values = [ord(x) >> 5 for x in HRP] + [0] + [ord(x) & 31 for x in HRP]
        checksum_input = hrp_values + conv + [0, 0, 0, 0, 0, 0]
        polymod = bech32_polymod(checksum_input) ^ 1
        checksum = [(polymod >> 5 * (5 - i)) & 31 for i in range(6)]

        combined = conv + checksum
        return HRP + '1' + ''.join([CHARSET[d] for d in combined])
    except Exception as e:
        print(f"Error converting hex to bech32: {e}", file=sys.stderr)
        return None

def bech32_to_hex(bech32_addr: str) -> Optional[str]:
    """Convert Cosmos bech32 address to Ethereum hex address"""
    try:
        if not bech32_addr.startswith(HRP + '1'):
            return None

        data_part = bech32_addr[len(HRP) + 1:]
        data = [CHARSET.find(c) for c in data_part]

        if any(d == -1 for d in data):
            return None

        # Remove checksum (last 6 characters)
        data = data[:-6]

        # Convert from 5-bit to 8-bit
        converted = convertbits(data, 5, 8, pad=False)
        if converted is None:
            return None

        return '0x' + bytes(converted).hex()
    except Exception as e:
        print(f"Error converting bech32 to hex: {e}", file=sys.stderr)
        return None

def normalize_address(addr: str) -> Tuple[str, str]:
    """
    Normalize address to both bech32 and hex formats.
    Returns: (bech32_addr, hex_addr)
    """
    addr = addr.strip()

    # If it's a hex address
    if addr.startswith('0x'):
        bech32_addr = hex_to_bech32(addr)
        if not bech32_addr:
            raise ValueError(f"Invalid hex address: {addr}")
        return bech32_addr, addr.lower()

    # If it's a bech32 address
    elif addr.startswith(HRP):
        hex_addr = bech32_to_hex(addr)
        if not hex_addr:
            raise ValueError(f"Invalid bech32 address: {addr}")
        return addr, hex_addr

    else:
        raise ValueError(f"Invalid address format: {addr}. Must be hex (0x...) or bech32 ({HRP}...)")

# ============================================================================
# Genesis File Management
# ============================================================================

def find_genesis_files(network: str, script_dir: Path) -> Tuple[Path, List[Path]]:
    """
    Find main genesis file and split account files for a network.
    Returns: (main_genesis_path, account_file_paths)
    """
    config_dir = script_dir.parent / "config" / "environments"

    # Try different naming patterns
    patterns = [
        f"{network}-genesis.genesis.json",
        f"{network}.genesis.json",
        f"{network}-genesis.json"
    ]

    main_genesis = None
    for pattern in patterns:
        path = config_dir / pattern
        if path.exists():
            main_genesis = path
            break

    if not main_genesis:
        raise FileNotFoundError(
            f"Genesis file not found for network '{network}'. Tried:\n" +
            '\n'.join(f"  - {config_dir / p}" for p in patterns)
        )

    # Find account split files
    base_name = main_genesis.stem.replace('.genesis', '')
    account_pattern = str(config_dir / f"{base_name}.genesis.accounts.*.json")
    account_files = sorted(glob.glob(account_pattern),
                          key=lambda x: int(Path(x).stem.split('.')[-1]))

    return main_genesis, account_files

def load_genesis_data(main_genesis: Path, account_files: List[Path]) -> Tuple[Dict, List[Dict], List[Dict], Dict]:
    """
    Load genesis data from main file and account split files.
    Returns: (genesis_json, accounts_list, balances_list, file_chunks_map)

    file_chunks_map: Dict mapping file path -> {'accounts': [...], 'balances': [...]}
                     Used to preserve which accounts belong to which file
    """
    with open(main_genesis, 'r') as f:
        genesis = json.load(f)

    # Get accounts and balances from main genesis
    accounts = genesis.get('app_state', {}).get('auth', {}).get('accounts', [])
    balances = genesis.get('app_state', {}).get('bank', {}).get('balances', [])
    file_chunks = {}

    # If no accounts in main genesis, load from split files
    if not accounts and account_files:
        print(f"Loading from {len(account_files)} split account files...")
        for account_file in account_files:
            with open(account_file, 'r') as f:
                chunk = json.load(f)
                # Store the original chunk data keyed by file path
                file_chunks[str(account_file)] = {
                    'accounts': chunk.get('accounts', []),
                    'balances': chunk.get('balances', [])
                }
                accounts.extend(chunk.get('accounts', []))
                balances.extend(chunk.get('balances', []))

    return genesis, accounts, balances, file_chunks

def save_genesis_data(main_genesis: Path, account_files: List[Path],
                     genesis: Dict, accounts: List[Dict], balances: List[Dict],
                     file_chunks: Dict, changed_addresses: set,
                     validate_supply: bool = True):
    """
    Save modified genesis data back to files, preserving structure to minimize diffs.
    Only updates files that contain changed accounts.

    Args:
        file_chunks: Original file chunks from load_genesis_data
        changed_addresses: Set of addresses that were modified
        validate_supply: If True, validates that bank.supply matches sum of all balances
    """
    # Validate total supply matches sum of balances (sanity check)
    if validate_supply and genesis.get('app_state', {}).get('bank', {}).get('supply'):
        supply_amount = int(genesis['app_state']['bank']['supply'][0]['amount'])

        # Calculate sum of all balances
        total_balances = 0
        for balance_entry in balances:
            for coin in balance_entry.get('coins', []):
                if coin.get('denom') == 'ashm':
                    total_balances += int(coin.get('amount', 0))
                    break

        if total_balances != supply_amount:
            print(f"\nWARNING: Supply mismatch detected!", file=sys.stderr)
            print(f"  bank.supply: {supply_amount} ashm", file=sys.stderr)
            print(f"  Sum of balances: {total_balances} ashm", file=sys.stderr)
            print(f"  Difference: {abs(supply_amount - total_balances)} ashm", file=sys.stderr)
            print(f"\nNote: For transfers, supply should not change (just moving funds).", file=sys.stderr)
            print(f"This warning may indicate an issue with the genesis file.", file=sys.stderr)

    # Create lookup maps for quick access
    account_map = {acc['address']: acc for acc in accounts}
    balance_map = {bal['address']: bal for bal in balances}

    if account_files and file_chunks:
        print(f"\nSaving to {len(account_files)} split account files...")

        # Track which files were modified
        modified_files = []

        # Pre-calculate new addresses to optimize last file write
        all_original_addresses = set()
        for chunk in file_chunks.values():
            all_original_addresses.update(acc['address'] for acc in chunk['accounts'])
        new_addresses = changed_addresses - all_original_addresses

        # Update each file individually, preserving structure
        for idx, account_file in enumerate(account_files):
            file_path = str(account_file)
            original_chunk = file_chunks.get(file_path, {'accounts': [], 'balances': []})

            # Check if this file contains any changed addresses
            file_addresses = {acc['address'] for acc in original_chunk['accounts']}
            has_changes = bool(file_addresses & changed_addresses)

            # Check if this is the last file and we need to add new accounts
            is_last_file = (idx == len(account_files) - 1)
            needs_new_accounts = is_last_file and new_addresses

            if not has_changes and not needs_new_accounts:
                # No changes in this file, skip it
                continue

            # Update accounts in this file
            updated_accounts = []
            for acc in original_chunk['accounts']:
                addr = acc['address']
                if addr in account_map:
                    updated_accounts.append(account_map[addr])
                else:
                    updated_accounts.append(acc)

            # Update balances in this file
            updated_balances = []
            for bal in original_chunk['balances']:
                addr = bal['address']
                if addr in balance_map:
                    updated_balances.append(balance_map[addr])
                else:
                    updated_balances.append(bal)

            # If this is the last file, add new accounts here (optimize: single write)
            if needs_new_accounts:
                for addr in new_addresses:
                    if addr in account_map:
                        updated_accounts.append(account_map[addr])
                    if addr in balance_map:
                        updated_balances.append(balance_map[addr])

            # Save updated file
            chunk_data = {
                'accounts': updated_accounts,
                'balances': updated_balances
            }

            with open(account_file, 'w') as f:
                json.dump(chunk_data, f, indent=2)

            modified_files.append(Path(account_file).name)

            # Print appropriate message
            if has_changes and needs_new_accounts:
                print(f"  {Path(account_file).name}: updated + added {len(new_addresses)} new account(s)")
            elif needs_new_accounts:
                print(f"  {Path(account_file).name}: added {len(new_addresses)} new account(s)")
            else:
                print(f"  {Path(account_file).name}: updated")

        print(f"\nModified {len(modified_files)} file(s)")

    else:
        # Save directly to main genesis
        print(f"\nSaving to main genesis file...")
        genesis['app_state']['auth']['accounts'] = accounts
        genesis['app_state']['bank']['balances'] = balances

        with open(main_genesis, 'w') as f:
            json.dump(genesis, f, indent=2)
        print(f"  {main_genesis.name}: {len(accounts)} accounts, {len(balances)} balances")

# ============================================================================
# Balance Transfer Logic
# ============================================================================

def find_account(accounts: List[Dict], address: str) -> Optional[Dict]:
    """Find account by address in accounts list"""
    for account in accounts:
        if account.get('address') == address:
            return account
    return None

def find_balance(balances: List[Dict], address: str) -> Optional[Dict]:
    """Find balance by address in balances list"""
    for balance in balances:
        if balance.get('address') == address:
            return balance
    return None

def create_base_account(address: str) -> Dict:
    """Create a new BaseAccount with standard structure"""
    return {
        "@type": "/cosmos.auth.v1beta1.BaseAccount",
        "address": address,
        "pub_key": None,
        "sequence": "0"
    }

def create_balance_entry(address: str, amount: str) -> Dict:
    """Create a new balance entry with standard structure"""
    return {
        "address": address,
        "coins": [
            {
                "denom": "ashm",
                "amount": amount
            }
        ]
    }

def get_balance_amount(balance: Optional[Dict]) -> int:
    """Get the ashm balance amount as integer"""
    if not balance:
        return 0

    coins = balance.get('coins', [])
    for coin in coins:
        if coin.get('denom') == 'ashm':
            return int(coin.get('amount', '0'))

    return 0

def set_balance_amount(balance: Dict, amount: int):
    """Set the ashm balance amount"""
    coins = balance.get('coins', [])
    for coin in coins:
        if coin.get('denom') == 'ashm':
            coin['amount'] = str(amount)
            return

    # If ashm coin doesn't exist, add it
    coins.append({
        "denom": "ashm",
        "amount": str(amount)
    })

def transfer_balance(accounts: List[Dict], balances: List[Dict],
                    source_bech32: str, dest_bech32: str, amount: int) -> Tuple[bool, str, set]:
    """
    Execute the balance transfer.
    Returns: (success, message, changed_addresses)
    """
    changed_addresses = set()
    # Step 1: Validate source account exists
    source_account = find_account(accounts, source_bech32)
    if not source_account:
        return False, f"Source account not found in genesis: {source_bech32}", changed_addresses

    # Step 2: Find source balance
    source_balance = find_balance(balances, source_bech32)
    if not source_balance:
        return False, f"Source account has no balance entry: {source_bech32}", changed_addresses

    source_amount = get_balance_amount(source_balance)

    # Step 3: Validate sufficient balance
    if source_amount < amount:
        return False, f"Insufficient balance. Source has {source_amount} ashm, need {amount} ashm", changed_addresses

    # Step 4: Check/create destination account
    dest_account = find_account(accounts, dest_bech32)
    if not dest_account:
        print(f"Creating new account for destination: {dest_bech32}")
        dest_account = create_base_account(dest_bech32)
        accounts.append(dest_account)
        changed_addresses.add(dest_bech32)

    # Step 5: Check/create destination balance
    dest_balance = find_balance(balances, dest_bech32)
    dest_amount = get_balance_amount(dest_balance)

    if not dest_balance:
        print(f"Creating new balance entry for destination: {dest_bech32}")
        dest_balance = create_balance_entry(dest_bech32, "0")
        balances.append(dest_balance)
        dest_amount = 0

    # Step 6: Execute transfer
    new_source_amount = source_amount - amount
    new_dest_amount = dest_amount + amount

    set_balance_amount(source_balance, new_source_amount)
    set_balance_amount(dest_balance, new_dest_amount)

    # Track changed addresses
    changed_addresses.add(source_bech32)
    changed_addresses.add(dest_bech32)

    return True, (
        f"Transfer successful!\n"
        f"  Source ({source_bech32}):\n"
        f"    Previous balance: {source_amount} ashm\n"
        f"    New balance:      {new_source_amount} ashm\n"
        f"  Destination ({dest_bech32}):\n"
        f"    Previous balance: {dest_amount} ashm\n"
        f"    New balance:      {new_dest_amount} ashm\n"
        f"  Amount transferred: {amount} ashm"
    ), changed_addresses

# ============================================================================
# Main Function
# ============================================================================

def main():
    if len(sys.argv) != 5:
        print(__doc__)
        sys.exit(1)

    network = sys.argv[1]
    source_addr = sys.argv[2]
    dest_addr = sys.argv[3]
    amount_str = sys.argv[4]

    # Validate and parse amount
    try:
        amount = int(amount_str)
        if amount <= 0:
            print("Error: Amount must be positive", file=sys.stderr)
            sys.exit(1)
    except ValueError:
        print(f"Error: Invalid amount '{amount_str}'. Must be an integer.", file=sys.stderr)
        sys.exit(1)

    # Normalize addresses
    try:
        source_bech32, source_hex = normalize_address(source_addr)
        dest_bech32, dest_hex = normalize_address(dest_addr)
    except ValueError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

    print("=" * 70)
    print("Genesis Balance Transfer")
    print("=" * 70)
    print(f"Network:      {network}")
    print(f"Source:       {source_bech32}")
    print(f"              {source_hex}")
    print(f"Destination:  {dest_bech32}")
    print(f"              {dest_hex}")
    print(f"Amount:       {amount} ashm")
    print("=" * 70)

    # Find genesis files
    script_dir = Path(__file__).parent
    try:
        main_genesis, account_files = find_genesis_files(network, script_dir)
    except FileNotFoundError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

    print(f"\nGenesis file: {main_genesis}")
    if account_files:
        print(f"Split files:  {len(account_files)} account files found")
    else:
        print(f"Split files:  None (using main genesis)")

    # Load genesis data
    print("\nLoading genesis data...")
    genesis, accounts, balances, file_chunks = load_genesis_data(main_genesis, account_files)
    print(f"Loaded: {len(accounts)} accounts, {len(balances)} balances")

    # Execute transfer
    print("\nExecuting transfer...")
    success, message, changed_addresses = transfer_balance(accounts, balances, source_bech32, dest_bech32, amount)

    if not success:
        print(f"\nError: {message}", file=sys.stderr)
        sys.exit(1)

    print(f"\n{message}")

    # Save changes
    save_genesis_data(main_genesis, account_files, genesis, accounts, balances, file_chunks, changed_addresses)

    print("\n" + "=" * 70)
    print("Transfer completed successfully!")
    print("=" * 70)

if __name__ == "__main__":
    main()
