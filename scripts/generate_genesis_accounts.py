#!/usr/bin/env python3
"""
Standalone Genesis Accounts Generator

Generates a new genesis.json file with accounts imported from Shardeum database.
This script updates an existing genesis file with accounts, balances, and optionally nonces
from the Shardeum accounts.sqlite3 database.

Usage:
    python3 generate_genesis_accounts.py [options]
"""

import json
import sqlite3
import sys
import argparse
import time
from typing import Dict, List, Tuple, Optional
import shutil
from datetime import datetime

# Bech32 encoding constants
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

def fast_bech32_encode(eth_addr_hex: str) -> Optional[str]:
    """Convert Ethereum address to Cosmos bech32 address"""
    try:
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
        print(f"Error encoding address {eth_addr_hex}: {e}", file=sys.stderr)
        return None

def extract_shardeum_accounts(db_path: str, 
                             min_balance: int = 0, 
                             max_accounts: int = 0,
                             include_nonce: bool = False,
                             balance_multiplier: int = 1) -> Tuple[List[Dict], int]:
    """
    Extract accounts from Shardeum database
    Returns: (accounts_list, total_supply)
    """
    print(f"Extracting accounts from {db_path}...")
    print(f"Options: include_nonce={include_nonce}, min_balance={min_balance}, max_accounts={max_accounts}, balance_multiplier={balance_multiplier}")
    
    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()
    
    # Build the query to extract accounts with balance, unique ID and optionally nonce
    if include_nonce:
        query = """
            SELECT 
              accountId as unique_id,
              substr(accountId, 1, 40) as eth_addr,
              json_extract(data, '$.account.balance.value') as balance_hex,
              json_extract(data, '$.account.nonce.value') as nonce_hex
            FROM accounts 
            WHERE json_extract(data, '$.accountType') = 0 
              AND json_extract(data, '$.account.balance.value') IS NOT NULL
              AND json_extract(data, '$.account.balance.value') <> '0'
            ORDER BY length(json_extract(data, '$.account.balance.value')) DESC
        """
    else:
        query = """
            SELECT 
              accountId as unique_id,
              substr(accountId, 1, 40) as eth_addr,
              json_extract(data, '$.account.balance.value') as balance_hex,
              '0' as nonce_hex
            FROM accounts 
            WHERE json_extract(data, '$.accountType') = 0 
              AND json_extract(data, '$.account.balance.value') IS NOT NULL
              AND json_extract(data, '$.account.balance.value') <> '0'
            ORDER BY length(json_extract(data, '$.account.balance.value')) DESC
        """
    
    if max_accounts > 0:
        query += f" LIMIT {max_accounts}"
    
    cursor.execute(query)
    
    accounts_data = []
    total_supply = 0
    processed = 0
    skipped_low_balance = 0
    failed_encoding = 0
    
    for row in cursor:
        unique_id = row[0]
        eth_addr = row[1]
        balance_hex = row[2]
        nonce_hex = row[3] if include_nonce and row[3] else '0'
        
        if not eth_addr or not balance_hex:
            continue
        
        # Convert balance
        try:
            balance_dec = int(balance_hex, 16)
        except ValueError:
            print(f"Warning: Invalid balance hex {balance_hex} for address {eth_addr}", file=sys.stderr)
            continue
        
        # Apply balance multiplier
        balance_dec = balance_dec * balance_multiplier
        
        # Check minimum balance threshold
        if balance_dec < min_balance:
            skipped_low_balance += 1
            continue
        
        # Convert nonce
        try:
            nonce_dec = int(nonce_hex, 16) if nonce_hex and nonce_hex != '0' else 0
        except ValueError:
            print(f"Warning: Invalid nonce hex {nonce_hex} for address {eth_addr}, using 0", file=sys.stderr)
            nonce_dec = 0
        
        # Convert to Cosmos address
        cosmos_addr = fast_bech32_encode(eth_addr)
        if not cosmos_addr:
            print(f"Warning: Failed to encode address {eth_addr}", file=sys.stderr)
            failed_encoding += 1
            continue
        
        accounts_data.append({
            'unique_id': unique_id,
            'address': cosmos_addr,
            'balance': balance_dec,
            'nonce': nonce_dec,
            'eth_address': eth_addr  # Keep for debugging
        })
        
        total_supply += balance_dec
        processed += 1
        
        if processed % 10000 == 0:
            print(f"Processed {processed} accounts...", file=sys.stderr)
    
    conn.close()
    
    print(f"\nExtraction Summary:", file=sys.stderr)
    print(f"  Total accounts processed: {processed:,}", file=sys.stderr)
    print(f"  Skipped (low balance): {skipped_low_balance:,}", file=sys.stderr)
    print(f"  Failed encoding: {failed_encoding:,}", file=sys.stderr)
    print(f"  Total supply: {total_supply:,} ashm", file=sys.stderr)
    
    return accounts_data, total_supply

def load_secure_accounts(secure_accounts_path: str, balance_multiplier: int = 1) -> List[Dict]:
    """
    Load secure accounts from JSON file (simple array format)
    Supports account replacement with Prime Vault Address
    """
    if not secure_accounts_path:
        return []

    print(f"Loading secure accounts from {secure_accounts_path}...")
    try:
        with open(secure_accounts_path, 'r') as f:
            secure_accounts_list = json.load(f)

        # Handle simple array format
        if not isinstance(secure_accounts_list, list):
            print(f"Error: Secure accounts file must contain a JSON array", file=sys.stderr)
            return []

        accounts = []
        for acc in secure_accounts_list:
            # Get the source funds address (Ethereum format)
            source_eth_addr = acc.get('SourceFundsAddress', '').lower().replace('0x', '')
            if not source_eth_addr:
                print(f"Warning: Missing SourceFundsAddress in secure account", file=sys.stderr)
                continue

            # Convert source address to Cosmos address
            source_cosmos_addr = fast_bech32_encode(source_eth_addr)
            if not source_cosmos_addr:
                print(f"Warning: Failed to encode secure account {acc.get('SourceFundsAddress')}", file=sys.stderr)
                continue

            # Get the prime vault address if provided (for replacement)
            prime_vault_eth_addr = acc.get('PriveVaultAddress', '').lower().replace('0x', '')
            prime_vault_cosmos_addr = None
            if prime_vault_eth_addr and prime_vault_eth_addr != source_eth_addr:
                prime_vault_cosmos_addr = fast_bech32_encode(prime_vault_eth_addr)
                if not prime_vault_cosmos_addr:
                    print(f"Warning: Failed to encode PriveVaultAddress {acc.get('PriveVaultAddress')}", file=sys.stderr)
                    prime_vault_cosmos_addr = None

            # Get balance from SourceFundsBalance field
            balance_str = acc.get('SourceFundsBalance', '0')
            try:
                balance = int(balance_str)
                # Apply balance multiplier
                balance = balance * balance_multiplier
            except ValueError:
                print(f"Warning: Invalid balance {balance_str} for account {acc.get('Name', 'Unknown')}", file=sys.stderr)
                balance = 0

            # Get nonce from SourceFundsNonce field
            nonce_str = acc.get('SourceFundsNonce', '0')
            try:
                nonce = int(nonce_str)
            except ValueError:
                print(f"Warning: Invalid nonce {nonce_str} for account {acc.get('Name', 'Unknown')}, using 0", file=sys.stderr)
                nonce = 0

            account_info = {
                'source_address': source_cosmos_addr,
                'source_eth_address': source_eth_addr,
                'balance': balance,
                'nonce': nonce,
                'name': acc.get('Name', ''),
                'is_secure': True
            }

            # If PriveVaultAddress is provided and different, include it
            if prime_vault_cosmos_addr:
                account_info['prime_vault_address'] = prime_vault_cosmos_addr
                account_info['prime_vault_eth_address'] = prime_vault_eth_addr
                print(f"Secure account {acc.get('Name')}: {source_cosmos_addr} -> {prime_vault_cosmos_addr} (REPLACEMENT)")
            else:
                print(f"Secure account {acc.get('Name')}: {source_cosmos_addr} (NO REPLACEMENT)")

            accounts.append(account_info)

        print(f"Loaded {len(accounts)} secure accounts", file=sys.stderr)
        return accounts
    except Exception as e:
        print(f"Error loading secure accounts: {e}", file=sys.stderr)
        return []

def update_genesis_with_accounts(genesis_path: str, 
                                accounts_data: List[Dict],
                                total_new_supply: int,
                                output_path: str,
                                include_balance: bool = True,
                                include_nonce: bool = False,
                                balance_multiplier: int = 1,
                                secure_accounts: List[Dict] = None) -> None:
    """
    Update genesis file with new accounts
    """
    print(f"\nUpdating genesis file...")
    print(f"Options: include_balance={include_balance}, include_nonce={include_nonce}, balance_multiplier={balance_multiplier}")
    if secure_accounts:
        print(f"Including {len(secure_accounts)} secure accounts")
    
    # Load existing genesis
    with open(genesis_path, 'r') as f:
        genesis = json.load(f)
    
    # Get existing accounts and balances (optimized with dictionaries for O(1) lookup)
    existing_accounts = {acc['address']: acc for acc in genesis['app_state']['auth']['accounts']}
    existing_balances = {}
    existing_balance_entries = {}
    
    for i, balance_entry in enumerate(genesis['app_state']['bank']['balances']):
        address = balance_entry['address']
        existing_balance_entries[address] = i  # Store index for fast updates
        for coin in balance_entry['coins']:
            if coin['denom'] == 'ashm':
                existing_balances[address] = int(coin['amount'])
                break
    
    # Track statistics
    new_accounts = 0
    updated_accounts = 0
    new_balance_entries = 0
    
    # Find the highest existing account number to continue numbering
    max_account_number = 0
    for acc in existing_accounts.values():
        try:
            acc_num = int(acc.get('account_number', '0'))
            max_account_number = max(max_account_number, acc_num)
        except (ValueError, TypeError):
            pass
    
    next_account_number = max_account_number + 1
    
    # First, process secure accounts if provided (with replacement support)
    secure_addresses = set()  # Tracks all secure account addresses (new or old)
    replaced_addresses = set()  # Tracks addresses that were replaced

    if secure_accounts:
        for account_data in secure_accounts:
            source_address = account_data['source_address']
            prime_vault_address = account_data.get('prime_vault_address')
            balance = account_data['balance']
            nonce = account_data['nonce']
            name = account_data.get('name', 'Unknown')

            # Determine if this is a replacement operation
            is_replacement = prime_vault_address and prime_vault_address != source_address

            if is_replacement:
                # REPLACEMENT MODE: Replace source_address with prime_vault_address
                target_address = prime_vault_address
                secure_addresses.add(prime_vault_address)
                replaced_addresses.add(source_address)

                print(f"  Replacing {name}: {source_address} -> {prime_vault_address}", file=sys.stderr)

                # Remove old source account from auth.accounts if it exists
                old_account_count = len(genesis['app_state']['auth']['accounts'])
                genesis['app_state']['auth']['accounts'] = [
                    acc for acc in genesis['app_state']['auth']['accounts']
                    if acc['address'] != source_address
                ]
                new_account_count = len(genesis['app_state']['auth']['accounts'])

                if old_account_count > new_account_count:
                    print(f"    ✓ Removed old account {source_address} from auth.accounts", file=sys.stderr)
                    # Remove from tracking dict
                    if source_address in existing_accounts:
                        del existing_accounts[source_address]
                else:
                    print(f"    ℹ Old account {source_address} not in genesis (will be skipped if in Shardeum DB)", file=sys.stderr)

                # Remove old source balance from bank.balances if it exists
                old_balance_count = len(genesis['app_state']['bank']['balances'])
                genesis['app_state']['bank']['balances'] = [
                    bal for bal in genesis['app_state']['bank']['balances']
                    if bal['address'] != source_address
                ]
                new_balance_count = len(genesis['app_state']['bank']['balances'])

                if old_balance_count > new_balance_count:
                    print(f"    ✓ Removed old balance {source_address} from bank.balances", file=sys.stderr)
                    # Remove from tracking dicts
                    if source_address in existing_balances:
                        del existing_balances[source_address]
                    if source_address in existing_balance_entries:
                        del existing_balance_entries[source_address]
            else:
                # NO REPLACEMENT: Just use source_address
                target_address = source_address
                secure_addresses.add(source_address)
                print(f"  Processing {name}: {source_address} (no replacement)", file=sys.stderr)

            # Add/update the target account in auth.accounts
            if target_address not in existing_accounts:
                new_account = {
                    "@type": "/cosmos.auth.v1beta1.BaseAccount",
                    "address": target_address,
                    "pub_key": None,
                    "sequence": str(nonce) if include_nonce else "0"
                }
                genesis['app_state']['auth']['accounts'].append(new_account)
                new_accounts += 1
                next_account_number += 1
                print(f"    Added new account {target_address}", file=sys.stderr)
            else:
                # Update existing account's sequence if needed
                if include_nonce:
                    for acc in genesis['app_state']['auth']['accounts']:
                        if acc['address'] == target_address:
                            acc['sequence'] = str(nonce)
                            break
                print(f"    Updated existing account {target_address}", file=sys.stderr)

            # Set balance if include_balance is True
            if include_balance:
                # Need to rebuild balance entries index since we may have removed entries
                existing_balance_entries = {
                    bal['address']: i
                    for i, bal in enumerate(genesis['app_state']['bank']['balances'])
                }

                if target_address not in existing_balance_entries:
                    # Add new balance entry
                    new_balance = {
                        "address": target_address,
                        "coins": [{"denom": "ashm", "amount": str(balance)}]
                    }
                    genesis['app_state']['bank']['balances'].append(new_balance)
                    new_balance_entries += 1
                    print(f"    Added new balance entry for {target_address}: {balance:,} ashm", file=sys.stderr)
                else:
                    # Update existing balance (override)
                    balance_index = existing_balance_entries[target_address]
                    balance_entry = genesis['app_state']['bank']['balances'][balance_index]
                    for coin in balance_entry['coins']:
                        if coin['denom'] == 'ashm':
                            coin['amount'] = str(balance)
                            break
                    print(f"    Updated balance for {target_address}: {balance:,} ashm", file=sys.stderr)
    
    # Process each account from Shardeum (skip if it's a secure account or was replaced)
    for account_data in accounts_data:
        address = account_data['address']

        # Skip if this is a secure account (already processed) or was replaced
        if address in secure_addresses or address in replaced_addresses:
            if address in replaced_addresses:
                print(f"  Skipping replaced address {address} from Shardeum data", file=sys.stderr)
            continue
            
        balance = account_data['balance']
        nonce = account_data['nonce']
        
        # Update auth accounts
        if address in existing_accounts:
            # Update existing account
            if include_nonce:
                existing_accounts[address]['sequence'] = str(nonce)
            updated_accounts += 1
        else:
            # Add new account
            new_account = {
                "@type": "/cosmos.auth.v1beta1.BaseAccount",
                "address": address,
                "pub_key": None,
                "sequence": str(nonce) if include_nonce else "0"
            }
            genesis['app_state']['auth']['accounts'].append(new_account)
            new_accounts += 1
            next_account_number += 1
        
        # Update bank balances if include_balance is True
        if include_balance:
            if address not in existing_balances:
                # Add new balance entry
                new_balance = {
                    "address": address,
                    "coins": [{"denom": "ashm", "amount": str(balance)}]
                }
                genesis['app_state']['bank']['balances'].append(new_balance)
                new_balance_entries += 1
            else:
                # Update existing balance (optimized O(1) lookup)
                balance_index = existing_balance_entries[address]
                balance_entry = genesis['app_state']['bank']['balances'][balance_index]
                for coin in balance_entry['coins']:
                    if coin['denom'] == 'ashm':
                        coin['amount'] = str(balance)
                        break
    
    # Update total supply if balances were included
    # IMPORTANT: The supply must exactly match the sum of all balances in the final genesis
    if include_balance:
        current_supply = int(genesis['app_state']['bank']['supply'][0]['amount'])

        # Recalculate total supply from the actual final balances in genesis
        # This ensures accuracy after all modifications
        all_balances_sum = 0
        for balance_entry in genesis['app_state']['bank']['balances']:
            for coin in balance_entry.get('coins', []):
                if coin.get('denom') == 'ashm':
                    all_balances_sum += int(coin.get('amount', 0))
                    break

        genesis['app_state']['bank']['supply'][0]['amount'] = str(all_balances_sum)
        print(f"Updated total supply from {current_supply:,} to {all_balances_sum:,} ashm", file=sys.stderr)
    
    # Sort accounts and balances for consistency
    genesis['app_state']['auth']['accounts'].sort(key=lambda x: x['address'])
    genesis['app_state']['bank']['balances'].sort(key=lambda x: x['address'])
    
    # Save the updated genesis
    with open(output_path, 'w') as f:
        json.dump(genesis, f, indent=2)
    
    print(f"\nUpdate Summary:", file=sys.stderr)
    print(f"  New accounts added: {new_accounts:,}", file=sys.stderr)
    print(f"  Existing accounts updated: {updated_accounts:,}", file=sys.stderr)
    if secure_accounts:
        print(f"  Secure account replacements: {len(replaced_addresses):,}", file=sys.stderr)
    if include_balance:
        print(f"  New balance entries: {new_balance_entries:,}", file=sys.stderr)
    print(f"  Genesis file saved to: {output_path}", file=sys.stderr)

def validate_genesis(genesis_path: str) -> bool:
    """
    Basic validation of the generated genesis file
    """
    try:
        with open(genesis_path, 'r') as f:
            genesis = json.load(f)
        
        # Check required fields exist
        required_fields = ['chain_id', 'app_state']
        for field in required_fields:
            if field not in genesis:
                print(f"Error: Missing required field '{field}' in genesis", file=sys.stderr)
                return False
        
        # Check app_state structure
        required_app_state = ['auth', 'bank']
        for field in required_app_state:
            if field not in genesis['app_state']:
                print(f"Error: Missing required app_state field '{field}'", file=sys.stderr)
                return False
        
        # Verify accounts and balances arrays exist
        if 'accounts' not in genesis['app_state']['auth']:
            print("Error: Missing auth.accounts array", file=sys.stderr)
            return False
        
        if 'balances' not in genesis['app_state']['bank']:
            print("Error: Missing bank.balances array", file=sys.stderr)
            return False
        
        # Count accounts and balances
        num_accounts = len(genesis['app_state']['auth']['accounts'])
        num_balances = len(genesis['app_state']['bank']['balances'])
        
        print(f"\nGenesis validation passed:", file=sys.stderr)
        print(f"  Total auth accounts: {num_accounts:,}", file=sys.stderr)
        print(f"  Total bank balances: {num_balances:,}", file=sys.stderr)
        
        return True
        
    except json.JSONDecodeError as e:
        print(f"Error: Invalid JSON in genesis file - {e}", file=sys.stderr)
        return False
    except Exception as e:
        print(f"Error validating genesis: {e}", file=sys.stderr)
        return False

def main():
    parser = argparse.ArgumentParser(
        description='Generate genesis.json with accounts from Shardeum database',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Generate with all accounts and balances (default)
  %(prog)s --input-genesis original.json --output new_genesis.json

  # Include nonces (sequences) from Shardeum
  %(prog)s --input-genesis original.json --output new_genesis.json --include-nonce

  # Filter accounts with minimum balance
  %(prog)s --input-genesis original.json --output new_genesis.json --min-balance 1000000000000000000

  # Limit number of accounts
  %(prog)s --input-genesis original.json --output new_genesis.json --max-accounts 1000

  # Apply balance multiplier (e.g., multiply all balances by 250)
  %(prog)s --input-genesis original.json --output new_genesis.json --balance-multiplier 250
        """
    )
    
    # Required arguments
    parser.add_argument('--input-genesis', required=True,
                       help='Path to input genesis.json file')
    parser.add_argument('--output', required=True,
                       help='Path to output genesis.json file')
    
    # Database options
    parser.add_argument('--db-path', 
                       default='accounts.sqlite3',
                       help='Path to Shardeum accounts.sqlite3 database')
    
    # Account filtering options
    parser.add_argument('--min-balance', type=int, default=0,
                       help='Minimum balance threshold in wei (default: 0)')
    parser.add_argument('--max-accounts', type=int, default=0,
                       help='Maximum number of accounts to import (0 for unlimited)')
    parser.add_argument('--balance-multiplier', type=int, default=1,
                       help='Multiplier to apply to all balances (default: 1)')
    
    # Import options
    parser.add_argument('--include-balance', action='store_true', default=True,
                       help='Include balances from Shardeum (default: True)')
    parser.add_argument('--no-include-balance', dest='include_balance', action='store_false',
                       help='Do not include balances from Shardeum')
    
    parser.add_argument('--include-nonce', action='store_true', default=False,
                       help='Include nonces as sequence numbers (default: False)')
    parser.add_argument('--no-include-nonce', dest='include_nonce', action='store_false',
                       help='Do not include nonces (default behavior)')
    
    # Secure accounts option
    parser.add_argument('--secure-accounts',
                       help='Path to JSON file containing secure accounts to preserve')
    
    # Other options
    parser.add_argument('--backup', action='store_true',
                       help='Create backup of output file if it exists')
    parser.add_argument('--validate', action='store_true', default=True,
                       help='Validate the generated genesis file (default: True)')
    parser.add_argument('--batch-size', type=int, default=10000,
                       help='Process accounts in batches (default: 10000)')
    
    args = parser.parse_args()
    
    try:
        # Validate input files exist
        with open(args.input_genesis, 'r') as f:
            pass
        
        if not args.db_path or not sqlite3.connect(args.db_path):
            raise FileNotFoundError(f"Database not found: {args.db_path}")
        
        # Create backup if requested and output file exists
        if args.backup and args.output != args.input_genesis:
            import os
            if os.path.exists(args.output):
                backup_path = f"{args.output}.backup.{datetime.now().strftime('%Y%m%d_%H%M%S')}"
                shutil.copy2(args.output, backup_path)
                print(f"Created backup: {backup_path}", file=sys.stderr)
        
        # Extract accounts from Shardeum database
        start_time = time.time()
        accounts_data, total_supply = extract_shardeum_accounts(
            args.db_path,
            args.min_balance,
            args.max_accounts,
            args.include_nonce,
            args.balance_multiplier
        )
        
        if not accounts_data:
            print("Warning: No accounts found to import", file=sys.stderr)
            sys.exit(1)
        
        # Load secure accounts if specified
        secure_accounts = []
        if args.secure_accounts:
            secure_accounts = load_secure_accounts(args.secure_accounts, args.balance_multiplier)
        
        # Update genesis with accounts
        update_genesis_with_accounts(
            args.input_genesis,
            accounts_data,
            total_supply,
            args.output,
            args.include_balance,
            args.include_nonce,
            args.balance_multiplier,
            secure_accounts
        )
        
        # Validate the generated genesis if requested
        if args.validate:
            if not validate_genesis(args.output):
                print("Error: Genesis validation failed", file=sys.stderr)
                sys.exit(1)
        
        elapsed = time.time() - start_time
        print(f"\n✅ Genesis generation completed in {elapsed:.2f} seconds", file=sys.stderr)
        print(f"Output file: {args.output}", file=sys.stderr)
        
    except FileNotFoundError as e:
        print(f"Error: File not found - {e}", file=sys.stderr)
        sys.exit(2)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(2)

if __name__ == '__main__':
    main()