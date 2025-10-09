#!/usr/bin/env python3
"""
Genesis Account Verification Script

Verifies that all accounts from the Shardeum accounts.sqlite3 database 
are present in genesis.json with matching balances.

This script uses the same conversion logic as the genesis_importer.sh script
to ensure consistency.
"""

import json
import sqlite3
import sys
import argparse
from typing import Dict, List, Tuple, Optional
import os
import glob
from pathlib import Path

# Bech32 encoding constants (copied from genesis_importer.sh)
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
    """Convert Ethereum address to Cosmos bech32 address (same logic as genesis_importer.sh)"""
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

def extract_shardeum_accounts(db_path: str, min_balance: int = 0, max_accounts: int = 0, 
                             include_nonce: bool = False, balance_multiplier: int = 1) -> List[Tuple[str, int, int, str]]:
    """
    Extract accounts from Shardeum database with balances, nonces, and unique IDs
    Returns: List of tuples (cosmos_address, balance, nonce, unique_id)
    """
    print(f"Extracting accounts from {db_path}...")
    print(f"Options: include_nonce={include_nonce}, balance_multiplier={balance_multiplier}")
    
    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()
    
    # Build base WHERE conditions
    base_conditions = "json_extract(data, '$.accountType') = 0"
    
    # When using balance_multiplier, we can't filter by min_balance at SQL level since
    # the multiplier is applied in Python. Always include accounts and filter in Python.
    # Only exclude accounts with missing balance data
    base_conditions += " AND json_extract(data, '$.account.balance.value') IS NOT NULL"
    
    # Only apply SQL-level balance filtering if no multiplier is used and min_balance > 0
    if balance_multiplier == 1 and min_balance > 0:
        # Safe to filter at SQL level when no multiplier is applied
        base_conditions += " AND json_extract(data, '$.account.balance.value') <> '0'"
    
    if include_nonce:
        query = f"""
            SELECT 
              accountId as unique_id,
              substr(accountId, 1, 40) as eth_addr,
              json_extract(data, '$.account.balance.value') as balance_hex,
              json_extract(data, '$.account.nonce.value') as nonce_hex
            FROM accounts 
            WHERE {base_conditions}
            ORDER BY CAST('0x' || COALESCE(json_extract(data, '$.account.balance.value'), '0') AS INTEGER) DESC
        """
    else:
        query = f"""
            SELECT 
              accountId as unique_id,
              substr(accountId, 1, 40) as eth_addr,
              json_extract(data, '$.account.balance.value') as balance_hex,
              '0' as nonce_hex
            FROM accounts 
            WHERE {base_conditions}
            ORDER BY CAST('0x' || COALESCE(json_extract(data, '$.account.balance.value'), '0') AS INTEGER) DESC
        """
    
    if max_accounts > 0:
        query += f" LIMIT {max_accounts}"
    
    cursor.execute(query)
    
    accounts = []
    processed = 0
    
    for row in cursor:
        unique_id = row[0]
        eth_addr = row[1]
        balance_hex = row[2]
        nonce_hex = row[3] if include_nonce and len(row) > 3 else '0'
        
        if not eth_addr:
            continue

        cosmos_addr = fast_bech32_encode(eth_addr)
        
        # Convert balance - handle null, empty, or zero values
        try:
            if not balance_hex or balance_hex == '0' or balance_hex == '':
                balance_dec = 0
            else:
                balance_dec = int(balance_hex, 16)
        except ValueError:
            print(f"Warning: Invalid balance hex '{balance_hex}' for address {eth_addr}", file=sys.stderr)
            balance_dec = 0
        
        # Apply balance multiplier
        balance_dec = balance_dec * balance_multiplier
        
        try:
            nonce_dec = int(nonce_hex, 16) if nonce_hex and nonce_hex != '0' else 0
        except ValueError:
            print(f"Warning: Invalid nonce hex {nonce_hex} for address {eth_addr}, using 0", file=sys.stderr)
            nonce_dec = 0
        
        if balance_dec < min_balance:
            continue
        
        if not cosmos_addr:
            print(f"Warning: Failed to encode address {eth_addr}", file=sys.stderr)
            continue
        
        accounts.append((cosmos_addr, balance_dec, nonce_dec, unique_id))
        processed += 1
        
        if processed % 10000 == 0:
            print(f"Processed {processed} accounts...", file=sys.stderr)
    
    conn.close()
    print(f"Extracted {len(accounts)} valid accounts from database", file=sys.stderr)
    return accounts

def load_genesis_accounts(genesis_path: str, include_nonce: bool = False, balance_multiplier: int = 1) -> Tuple[Dict[str, Dict], int]:
    """
    Load accounts, balances, sequences (nonces), and unique IDs from genesis.json.
    Supports both monolithic and split genesis formats.
    Returns: (account_data_dict, total_supply)
    where account_data_dict = {address: {'balance': int, 'sequence': int}}
    """
    print(f"Loading genesis accounts from {genesis_path}...")
    print(f"Options: include_nonce={include_nonce}, balance_multiplier={balance_multiplier}")
    
    with open(genesis_path, 'r') as f:
        genesis = json.load(f)
    
    account_data = {}
    total_supply = 0
    
    # Check if this is a split genesis (no accounts/balances in main file)
    auth_accounts = genesis.get('app_state', {}).get('auth', {}).get('accounts', [])
    bank_balances = genesis.get('app_state', {}).get('bank', {}).get('balances', [])
    
    # If main genesis has accounts, use monolithic approach
    if auth_accounts or bank_balances:
        print("Loading from monolithic genesis file...")
        
        # Extract sequences from auth.accounts
        for acc in auth_accounts:
            address = acc.get('address')
            sequence = int(acc.get('sequence', '0')) if include_nonce else 0
            account_data[address] = {'balance': 0, 'sequence': sequence}
        
        # Extract balances from bank.balances
        for balance_entry in bank_balances:
            address = balance_entry.get('address')
            coins = balance_entry.get('coins', [])
            
            for coin in coins:
                if coin.get('denom') == 'ashm':
                    amount = int(coin.get('amount', 0))
                    
                    if address not in account_data:
                        account_data[address] = {'balance': 0, 'sequence': 0}
                    
                    account_data[address]['balance'] = amount
                    total_supply += amount
                    break
    
    else:
        # Split genesis format - load from account files
        print("No accounts found in main genesis, checking for split account files...")
        
        genesis_dir = os.path.dirname(genesis_path)
        genesis_name = os.path.basename(genesis_path)
        
        # Determine base name for account files
        base_name = genesis_name.replace('.genesis.json', '')
        if base_name == genesis_name:
            base_name = genesis_name.replace('.json', '')
        
        # Handle already split pattern (e.g., "mainnet-genesis.genesis.json")
        if base_name.endswith('.genesis'):
            base_name = base_name[:-8]  # Remove '.genesis'
        
        # Find account files
        account_pattern = os.path.join(genesis_dir, f"{base_name}.genesis.accounts.*.json")
        account_files = sorted(glob.glob(account_pattern), 
                             key=lambda x: int(Path(x).stem.split('.')[-1]))
        
        if not account_files:
            print(f"Warning: No split account files found matching pattern: {account_pattern}", file=sys.stderr)
            print(f"Looked for files like: {base_name}.genesis.accounts.N.json", file=sys.stderr)
        else:
            print(f"Found {len(account_files)} account files, loading...", file=sys.stderr)
            
            # Load accounts and balances from split files
            for account_file in account_files:
                print(f"Loading {os.path.basename(account_file)}...", file=sys.stderr)
                with open(account_file, 'r') as f:
                    chunk_data = json.load(f)
                
                # Process accounts (sequences/nonces)
                chunk_accounts = chunk_data.get('accounts', [])
                for acc in chunk_accounts:
                    address = acc.get('address')
                    if address:
                        sequence = int(acc.get('sequence', '0')) if include_nonce else 0
                        account_data[address] = {'balance': 0, 'sequence': sequence}
                
                # Process balances
                chunk_balances = chunk_data.get('balances', [])
                for balance_entry in chunk_balances:
                    address = balance_entry.get('address')
                    if address:
                        coins = balance_entry.get('coins', [])
                        
                        for coin in coins:
                            if coin.get('denom') == 'ashm':
                                amount = int(coin.get('amount', 0))
                                
                                if address not in account_data:
                                    account_data[address] = {'balance': 0, 'sequence': 0}
                                
                                account_data[address]['balance'] = amount
                                total_supply += amount
                                break
    
    print(f"Loaded {len(account_data)} accounts from genesis with total supply {total_supply}", file=sys.stderr)
    return account_data, total_supply

def load_secure_accounts_for_verification(secure_accounts_path: str, balance_multiplier: int = 1) -> List[Dict]:
    """
    Load secure accounts from JSON file for verification (simple array format)
    Supports PriveVaultAddress replacement verification
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
            # Get source address
            source_eth_addr = acc.get('SourceFundsAddress', '').lower().replace('0x', '')
            if not source_eth_addr:
                print(f"Warning: Missing SourceFundsAddress in secure account", file=sys.stderr)
                continue

            source_cosmos_addr = fast_bech32_encode(source_eth_addr)
            if not source_cosmos_addr:
                print(f"Warning: Failed to encode secure account {acc.get('SourceFundsAddress')}", file=sys.stderr)
                continue

            # Get prime vault address if provided
            prime_vault_eth_addr = acc.get('PriveVaultAddress', '').lower().replace('0x', '')
            prime_vault_cosmos_addr = None
            if prime_vault_eth_addr and prime_vault_eth_addr != source_eth_addr:
                prime_vault_cosmos_addr = fast_bech32_encode(prime_vault_eth_addr)
                if not prime_vault_cosmos_addr:
                    print(f"Warning: Failed to encode PriveVaultAddress {acc.get('PriveVaultAddress')}", file=sys.stderr)

            # Get balance from SourceFundsBalance field
            balance_str = acc.get('SourceFundsBalance', '0')
            try:
                balance = int(balance_str)
                # Apply balance multiplier for verification
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
                'expected_balance': balance,
                'expected_nonce': nonce,
                'name': acc.get('Name', ''),
            }

            # If replacement is happening, we should verify the prime vault address
            if prime_vault_cosmos_addr:
                account_info['address'] = prime_vault_cosmos_addr
                account_info['prime_vault_eth_address'] = prime_vault_eth_addr
                account_info['is_replacement'] = True
                print(f"  Will verify replacement: {acc.get('Name')} at {prime_vault_cosmos_addr}", file=sys.stderr)
            else:
                account_info['address'] = source_cosmos_addr
                account_info['is_replacement'] = False
                print(f"  Will verify: {acc.get('Name')} at {source_cosmos_addr}", file=sys.stderr)

            accounts.append(account_info)

        print(f"Loaded {len(accounts)} secure accounts for verification", file=sys.stderr)
        return accounts
    except Exception as e:
        print(f"Error loading secure accounts: {e}", file=sys.stderr)
        return []

def verify_secure_accounts(secure_accounts: List[Dict], genesis_data: Dict[str, Dict], check_nonce: bool = False) -> Tuple[bool, Dict]:
    """
    Verify that all secure accounts are present in genesis with correct balances and nonces
    Supports verification of PriveVaultAddress replacements
    """
    print("Verifying secure accounts...")

    results = {
        'total_secure': len(secure_accounts),
        'missing_secure': [],
        'balance_mismatches': [],
        'nonce_mismatches': [],
        'source_still_present': [],  # Old addresses that should have been removed
        'matched_secure': 0,
        'total_expected_balance': sum(acc['expected_balance'] for acc in secure_accounts)
    }

    for acc in secure_accounts:
        target_address = acc['address']  # This is either source or prime vault
        expected_balance = acc['expected_balance']
        expected_nonce = acc.get('expected_nonce', 0)
        name = acc['name']
        is_replacement = acc.get('is_replacement', False)
        source_address = acc.get('source_address')

        # If this is a replacement, verify old address was removed
        if is_replacement and source_address:
            if source_address in genesis_data:
                results['source_still_present'].append({
                    'address': source_address,
                    'name': name,
                    'note': f'Old source address should have been replaced with {target_address}'
                })

        # Verify the target address (either prime vault or source) is present
        if target_address not in genesis_data:
            results['missing_secure'].append({
                'address': target_address,
                'name': name,
                'expected_balance': expected_balance,
                'expected_nonce': expected_nonce if check_nonce else None,
                'is_replacement': is_replacement
            })
        else:
            # Verify balance
            genesis_balance = genesis_data[target_address].get('balance', 0)
            balance_match = genesis_balance == expected_balance

            # Verify nonce if requested
            genesis_nonce = genesis_data[target_address].get('sequence', 0)
            nonce_match = genesis_nonce == expected_nonce if check_nonce else True

            if not balance_match:
                results['balance_mismatches'].append({
                    'address': target_address,
                    'name': name,
                    'expected_balance': expected_balance,
                    'genesis_balance': genesis_balance,
                    'difference': genesis_balance - expected_balance,
                    'is_replacement': is_replacement
                })

            if check_nonce and not nonce_match:
                results['nonce_mismatches'].append({
                    'address': target_address,
                    'name': name,
                    'expected_nonce': expected_nonce,
                    'genesis_nonce': genesis_nonce,
                    'difference': genesis_nonce - expected_nonce,
                    'is_replacement': is_replacement
                })

            if balance_match and nonce_match:
                results['matched_secure'] += 1

    is_valid = (len(results['missing_secure']) == 0 and
                len(results['balance_mismatches']) == 0 and
                len(results['source_still_present']) == 0 and
                (not check_nonce or len(results['nonce_mismatches']) == 0))

    return is_valid, results

def verify_accounts(shardeum_accounts: List[Tuple[str, int, int, str]],
                   genesis_data: Dict[str, Dict],
                   check_nonce: bool = False, balance_multiplier: int = 1,
                   secure_accounts: List[Dict] = None) -> Tuple[bool, Dict]:
    """Verify that all Shardeum accounts are present in genesis with correct balances, nonces, and unique IDs"""
    print("Verifying accounts...")
    print(f"Options: check_nonce={check_nonce}, balance_multiplier={balance_multiplier}")

    # Build sets of addresses to skip (source addresses that were replaced)
    replaced_source_addresses = set()
    secure_target_addresses = {}  # Maps target address -> secure account data

    if secure_accounts:
        for acc in secure_accounts:
            if acc.get('is_replacement'):
                # This is a replacement - skip the old source address
                replaced_source_addresses.add(acc['source_address'])
                # Store the target address with its expected values
                secure_target_addresses[acc['address']] = {
                    'balance': acc['expected_balance'],
                    'nonce': acc['expected_nonce'],
                    'name': acc['name']
                }
            else:
                # No replacement - target is same as source
                secure_target_addresses[acc['address']] = {
                    'balance': acc['expected_balance'],
                    'nonce': acc['expected_nonce'],
                    'name': acc['name']
                }

    # Calculate total Shardeum accounts excluding replaced source addresses
    # and add any new secure target addresses not in Shardeum DB
    total_shardeum_count = len(shardeum_accounts) - len(replaced_source_addresses)
    total_shardeum_supply = sum(balance for addr, balance, _, _ in shardeum_accounts
                                 if addr not in replaced_source_addresses)

    # Add new secure target addresses (not in Shardeum DB)
    for target_address, secure_data in secure_target_addresses.items():
        shardeum_addrs_set = {addr for addr, _, _, _ in shardeum_accounts}
        if target_address not in shardeum_addrs_set:
            total_shardeum_count += 1
            total_shardeum_supply += secure_data['balance']

    results = {
        'total_shardeum': total_shardeum_count,
        'total_genesis': len(genesis_data),
        'missing_accounts': [],
        'balance_mismatches': [],
        'nonce_mismatches': [],
        'matched_accounts': 0,
        'total_shardeum_supply': total_shardeum_supply,
        'matched_supply': 0,
        'check_nonce': check_nonce,
    }

    # Create dict for easy lookup
    shardeum_dict = {addr: {'balance': balance, 'nonce': nonce}
                     for addr, balance, nonce, _ in shardeum_accounts}

    for address, expected_balance, expected_nonce, expected_unique_id in shardeum_accounts:
        # Skip if this address was replaced by a secure account
        if address in replaced_source_addresses:
            print(f"  Skipping replaced source address {address} from verification", file=sys.stderr)
            continue

        # If this address is a secure target address, use the secure account values
        if address in secure_target_addresses:
            expected_balance = secure_target_addresses[address]['balance']
            expected_nonce = secure_target_addresses[address]['nonce']
            print(f"  Using secure account values for {address}: balance={expected_balance}, nonce={expected_nonce}", file=sys.stderr)
        if address not in genesis_data:
            results['missing_accounts'].append({
                'address': address,
                'expected_balance': expected_balance,
                'expected_nonce': expected_nonce if check_nonce else None,
                'expected_unique_id': expected_unique_id
            })
        else:
            genesis_balance = genesis_data[address].get('balance', 0)
            genesis_sequence = genesis_data[address].get('sequence', 0)
            balance_match = genesis_balance == expected_balance
            nonce_match = genesis_sequence == expected_nonce if check_nonce else True
            
            if not balance_match:
                results['balance_mismatches'].append({
                    'address': address,
                    'expected_balance': expected_balance,
                    'genesis_balance': genesis_balance,
                    'difference': genesis_balance - expected_balance
                })
            
            if check_nonce and not nonce_match:
                results['nonce_mismatches'].append({
                    'address': address,
                    'expected_nonce': expected_nonce,
                    'genesis_sequence': genesis_sequence,
                    'difference': genesis_sequence - expected_nonce
                })
            
            if balance_match and nonce_match:
                results['matched_accounts'] += 1
                results['matched_supply'] += expected_balance
    
    # Check for secure target addresses that aren't in Shardeum DB (new addresses)
    # These need to be verified separately
    for target_address, secure_data in secure_target_addresses.items():
        if target_address not in shardeum_dict:
            # This is a new address not in Shardeum DB (e.g., new prime vault)
            # Verify it in genesis
            expected_balance = secure_data['balance']
            expected_nonce = secure_data['nonce']

            print(f"  Verifying new secure target address {target_address} (not in Shardeum DB)", file=sys.stderr)

            if target_address not in genesis_data:
                results['missing_accounts'].append({
                    'address': target_address,
                    'expected_balance': expected_balance,
                    'expected_nonce': expected_nonce if check_nonce else None,
                    'expected_unique_id': 'N/A (secure account)'
                })
            else:
                genesis_balance = genesis_data[target_address].get('balance', 0)
                genesis_sequence = genesis_data[target_address].get('sequence', 0)
                balance_match = genesis_balance == expected_balance
                nonce_match = genesis_sequence == expected_nonce if check_nonce else True

                if not balance_match:
                    results['balance_mismatches'].append({
                        'address': target_address,
                        'expected_balance': expected_balance,
                        'genesis_balance': genesis_balance,
                        'difference': genesis_balance - expected_balance
                    })

                if check_nonce and not nonce_match:
                    results['nonce_mismatches'].append({
                        'address': target_address,
                        'expected_nonce': expected_nonce,
                        'genesis_sequence': genesis_sequence,
                        'difference': genesis_sequence - expected_nonce
                    })

                if balance_match and nonce_match:
                    results['matched_accounts'] += 1
                    results['matched_supply'] += expected_balance

    # Check for extra accounts in genesis (not from Shardeum and not secure accounts)
    extra_accounts = []
    validator_accounts = []

    for address, data in genesis_data.items():
        # Skip if it's in Shardeum or is a secure target address
        if address not in shardeum_dict and address not in secure_target_addresses:
            balance = data.get('balance', 0)
            # These could be validator accounts or dev accounts
            if balance >= 1000000000000000000000:  # >= 1000 SHM (likely validator/dev account)
                validator_accounts.append({'address': address, 'balance': balance})
            else:
                extra_accounts.append({'address': address, 'balance': balance})

    results['extra_accounts'] = extra_accounts
    results['validator_accounts'] = validator_accounts
    
    is_valid = (len(results['missing_accounts']) == 0 and 
                len(results['balance_mismatches']) == 0 and
                (not check_nonce or len(results['nonce_mismatches']) == 0))
    
    # If secure accounts were provided, verify them separately
    if secure_accounts:
        secure_valid, secure_results = verify_secure_accounts(secure_accounts, genesis_data, check_nonce)
        results['secure_accounts_results'] = secure_results
        is_valid = is_valid and secure_valid
    
    return is_valid, results

def print_report(results: Dict, genesis_path: str = ''):
    """Print verification report"""
    print("\n" + "="*80)
    print("GENESIS ACCOUNT VERIFICATION REPORT")
    print("="*80)
    
    if genesis_path:
        print(f"Genesis file: {genesis_path}")
        
        # Check if this was loaded from split files
        genesis_dir = os.path.dirname(genesis_path)
        genesis_name = os.path.basename(genesis_path)
        base_name = genesis_name.replace('.genesis.json', '')
        if base_name == genesis_name:
            base_name = genesis_name.replace('.json', '')
        if base_name.endswith('.genesis'):
            base_name = base_name[:-8]
        
        account_pattern = os.path.join(genesis_dir, f"{base_name}.genesis.accounts.*.json")
        account_files = glob.glob(account_pattern)
        if account_files:
            print(f"Split account files: {len(account_files)} files detected")
        print()
    
    print(f"Total accounts in Shardeum DB: {results['total_shardeum']:,}")
    print(f"Total accounts in Genesis:     {results['total_genesis']:,}")
    print(f"Matched accounts:              {results['matched_accounts']:,}")
    print(f"Missing accounts:              {len(results['missing_accounts']):,}")
    print(f"Balance mismatches:            {len(results['balance_mismatches']):,}")
    
    if results.get('check_nonce'):
        print(f"Nonce mismatches:              {len(results.get('nonce_mismatches', [])):,}")
    
    print(f"\nSupply Information:")
    print(f"Total Shardeum supply:         {results['total_shardeum_supply']:,} ashm")
    print(f"Matched supply:                {results['matched_supply']:,} ashm")
    
    if results['missing_accounts']:
        print(f"\n❌ MISSING ACCOUNTS ({len(results['missing_accounts'])}):")
        for i, account in enumerate(results['missing_accounts'][:10]):  # Show first 10
            msg = f"  {i+1}. {account['address']} (balance: {account['expected_balance']:,}"
            if results.get('check_nonce') and account.get('expected_nonce') is not None:
                msg += f", nonce: {account['expected_nonce']}"
            msg += f", unique_id: {account.get('expected_unique_id', 'N/A')[:16]}...)"
            print(msg)
        if len(results['missing_accounts']) > 10:
            print(f"  ... and {len(results['missing_accounts']) - 10} more")
    
    if results['balance_mismatches']:
        print(f"\n❌ BALANCE MISMATCHES ({len(results['balance_mismatches'])}):")
        for i, mismatch in enumerate(results['balance_mismatches'][:10]):  # Show first 10
            print(f"  {i+1}. {mismatch['address']}")
            print(f"     Expected: {mismatch['expected_balance']:,}")
            print(f"     Genesis:  {mismatch['genesis_balance']:,}")
            print(f"     Diff:     {mismatch['difference']:+,}")
        if len(results['balance_mismatches']) > 10:
            print(f"  ... and {len(results['balance_mismatches']) - 10} more")
    
    if results.get('check_nonce') and results.get('nonce_mismatches'):
        print(f"\n❌ NONCE/SEQUENCE MISMATCHES ({len(results['nonce_mismatches'])}):")
        for i, mismatch in enumerate(results['nonce_mismatches'][:10]):  # Show first 10
            print(f"  {i+1}. {mismatch['address']}")
            print(f"     Expected nonce:    {mismatch['expected_nonce']}")
            print(f"     Genesis sequence:  {mismatch['genesis_sequence']}")
            print(f"     Diff:              {mismatch['difference']:+}")
        if len(results['nonce_mismatches']) > 10:
            print(f"  ... and {len(results['nonce_mismatches']) - 10} more")
    
    # Print secure accounts verification if present
    if 'secure_accounts_results' in results:
        secure_results = results['secure_accounts_results']
        print(f"\n" + "-"*80)
        print("SECURE ACCOUNTS VERIFICATION")
        print("-"*80)
        print(f"Total secure accounts:         {secure_results['total_secure']}")
        print(f"Matched secure accounts:       {secure_results['matched_secure']}")
        print(f"Missing secure accounts:       {len(secure_results['missing_secure'])}")
        print(f"Balance mismatches:            {len(secure_results['balance_mismatches'])}")
        if results.get('check_nonce'):
            print(f"Nonce mismatches:              {len(secure_results.get('nonce_mismatches', []))}")
        print(f"Old addresses still present:   {len(secure_results.get('source_still_present', []))}")

        if secure_results.get('source_still_present'):
            print(f"\n❌ OLD SOURCE ADDRESSES STILL PRESENT (should have been replaced):")
            for acc in secure_results['source_still_present']:
                print(f"  - {acc['name']}: {acc['address']}")
                print(f"    Note: {acc['note']}")

        if secure_results['missing_secure']:
            print(f"\n❌ MISSING SECURE ACCOUNTS:")
            for acc in secure_results['missing_secure']:
                replacement_note = " (REPLACEMENT)" if acc.get('is_replacement') else ""
                print(f"  - {acc['name']}: {acc['address']}{replacement_note}")
                print(f"    Expected balance: {acc['expected_balance']:,} wei")
                if results.get('check_nonce') and acc.get('expected_nonce') is not None:
                    print(f"    Expected nonce: {acc['expected_nonce']}")

        if secure_results['balance_mismatches']:
            print(f"\n❌ SECURE ACCOUNT BALANCE MISMATCHES:")
            for mismatch in secure_results['balance_mismatches']:
                replacement_note = " (REPLACEMENT)" if mismatch.get('is_replacement') else ""
                print(f"  - {mismatch['name']}: {mismatch['address']}{replacement_note}")
                print(f"    Expected: {mismatch['expected_balance']:,}")
                print(f"    Genesis:  {mismatch['genesis_balance']:,}")
                print(f"    Diff:     {mismatch['difference']:+,}")

        if results.get('check_nonce') and secure_results.get('nonce_mismatches'):
            print(f"\n❌ SECURE ACCOUNT NONCE MISMATCHES:")
            for mismatch in secure_results['nonce_mismatches']:
                replacement_note = " (REPLACEMENT)" if mismatch.get('is_replacement') else ""
                print(f"  - {mismatch['name']}: {mismatch['address']}{replacement_note}")
                print(f"    Expected nonce: {mismatch['expected_nonce']}")
                print(f"    Genesis nonce:  {mismatch['genesis_nonce']}")
                print(f"    Diff:           {mismatch['difference']:+}")
    
    print("\n" + "="*80)
    
    check_nonce = results.get('check_nonce', False)
    has_secure_accounts = 'secure_accounts_results' in results
    
    # Check overall verification status
    basic_valid = (len(results['missing_accounts']) == 0 and 
                   len(results['balance_mismatches']) == 0 and 
                   (not check_nonce or len(results.get('nonce_mismatches', [])) == 0))
    
    secure_valid = True
    if has_secure_accounts:
        secure_results = results['secure_accounts_results']
        secure_valid = (len(secure_results['missing_secure']) == 0 and
                       len(secure_results['balance_mismatches']) == 0 and
                       len(secure_results.get('source_still_present', [])) == 0 and
                       (not check_nonce or len(secure_results.get('nonce_mismatches', [])) == 0))
    
    if basic_valid and secure_valid:
        print("✅ VERIFICATION PASSED: All accounts are correctly imported!")
    else:
        print("❌ VERIFICATION FAILED: Some accounts are missing or have incorrect data!")
            
    print("="*80)

def main():
    parser = argparse.ArgumentParser(
        description='Verify genesis accounts against Shardeum database',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Basic verification (balances only)
  %(prog)s --genesis-path genesis.json

  # Verify with nonce checking
  %(prog)s --genesis-path genesis.json --check-nonce

  # Verify with custom database path
  %(prog)s --db-path /path/to/accounts.sqlite3 --genesis-path genesis.json

  # Filter accounts with minimum balance
  %(prog)s --genesis-path genesis.json --min-balance 1000000000000000000

  # Verify with balance multiplier (e.g., if genesis was generated with multiplier 250)
  %(prog)s --genesis-path genesis.json --balance-multiplier 250

  # Verify split genesis files (automatically detected)
  %(prog)s --genesis-path config/environments/mainnet-genesis.genesis.json

  # Verify with secure accounts file
  %(prog)s --genesis-path genesis.json --secure-accounts secure_accounts.json
        """
    )
    parser.add_argument('--db-path', default='accounts.sqlite3',
                       help='Path to Shardeum accounts.sqlite3 database')
    parser.add_argument('--genesis-path', default='.evmd/config/genesis.json',
                       help='Path to genesis.json file')
    parser.add_argument('--min-balance', type=int, default=0,
                       help='Minimum balance threshold (in wei)')
    parser.add_argument('--max-accounts', type=int, default=0,
                       help='Maximum number of accounts to verify (0 for unlimited)')
    parser.add_argument('--balance-multiplier', type=int, default=1,
                       help='Multiplier applied to Shardeum balances for verification (default: 1)')
    parser.add_argument('--check-nonce', action='store_true', default=False,
                       help='Also verify nonces/sequences (default: False)')
    parser.add_argument('--secure-accounts',
                       help='Path to JSON file containing secure accounts to verify')
    parser.add_argument('--verbose', '-v', action='store_true',
                       help='Enable verbose output')
    parser.add_argument('--auto-detect-split', action='store_true', default=True,
                       help='Automatically detect and load split genesis files (default: True)')
    
    args = parser.parse_args()
    
    try:
        # Extract accounts from Shardeum database
        shardeum_accounts = extract_shardeum_accounts(
            args.db_path, 
            args.min_balance, 
            args.max_accounts,
            args.check_nonce,
            args.balance_multiplier
        )
        
        # Load genesis accounts
        genesis_data, total_supply = load_genesis_accounts(
            args.genesis_path, 
            args.check_nonce,
            args.balance_multiplier
        )
        
        # Load secure accounts if specified
        secure_accounts = None
        if args.secure_accounts:
            secure_accounts = load_secure_accounts_for_verification(args.secure_accounts, args.balance_multiplier)
        
        # Verify accounts
        is_valid, results = verify_accounts(
            shardeum_accounts, 
            genesis_data,
            args.check_nonce,
            args.balance_multiplier,
            secure_accounts
        )
        
        # Print report
        print_report(results, args.genesis_path)

        print("Total Genesis accounts (including validator account): ", results['total_genesis'])
        print("Total Shardeum accounts: ", results['total_shardeum'])
        print("Total Genesis supply (including validator balance): ", total_supply)
        print("Total Shardeum supply: ", results['total_shardeum_supply'])
        
        # Exit with appropriate code
        sys.exit(0 if is_valid else 1)
        
    except FileNotFoundError as e:
        print(f"Error: File not found - {e}", file=sys.stderr)
        sys.exit(2)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        if args.verbose:
            import traceback
            traceback.print_exc()
        sys.exit(2)

if __name__ == '__main__':
    main()