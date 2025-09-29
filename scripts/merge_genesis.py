#!/usr/bin/env python3
"""
Merge addresses from old genesis file into new genesis file format.
Usage: python merge_genesis.py <addresses.json> <cosmos_genesis_file.json> <output_cosmos_genesis_file.json>
"""

import json
import sys
import argparse
from typing import Dict, List, Set, Optional

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

def merge_genesis_files(old_genesis_path: str, new_genesis_path: str, output_path: str):
    """Merge old genesis addresses into new genesis format"""
    print(f"Loading old genesis from {old_genesis_path}")
    old_genesis = load_old_genesis(old_genesis_path)
    
    print(f"Loading new genesis from {new_genesis_path}")
    new_genesis = load_new_genesis(new_genesis_path)
    
    # Get existing addresses to prevent duplicates
    existing_eth_addresses, existing_cosmos_addresses = get_existing_addresses(new_genesis)
    print(f"Found {len(existing_eth_addresses)} existing Ethereum addresses and {len(existing_cosmos_addresses)} existing Cosmos addresses in new genesis")
    
    # Get existing balances for total supply calculation
    existing_balances = {}
    for balance_entry in new_genesis.get('app_state', {}).get('bank', {}).get('balances', []):
        address = balance_entry.get('address', '')
        for coin in balance_entry.get('coins', []):
            if coin.get('denom') == 'ashm':
                existing_balances[address] = int(coin.get('amount', '0'))
                break
    
    # Get next account number
    auth_accounts = new_genesis.get('app_state', {}).get('auth', {}).get('accounts', [])
    
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
        
        # Skip addresses with 0 balance
        if balance_int == 0:
            print(f"Skipping address with 0 balance: {eth_address}")
            skipped_count += 1
            continue
        
        # Ensure EVM structure exists but keep accounts empty - EVM accounts are populated at runtime
        if 'evm' not in new_genesis['app_state']:
            # Initialize with default EVM structure if missing
            new_genesis['app_state']['evm'] = {
                'accounts': [],
                'params': {
                    'evm_denom': 'ashm',
                    'extra_eips': [],
                    'evm_channels': [],
                    'access_control': {
                        'create': {
                            'access_type': 'ACCESS_TYPE_PERMISSIONLESS',
                            'access_control_list': []
                        },
                        'call': {
                            'access_type': 'ACCESS_TYPE_PERMISSIONLESS', 
                            'access_control_list': []
                        }
                    },
                    'active_static_precompiles': [
                        '0x0000000000000000000000000000000000000100',
                        '0x0000000000000000000000000000000000000400',
                        '0x0000000000000000000000000000000000000800',
                        '0x0000000000000000000000000000000000000801',
                        '0x0000000000000000000000000000000000000802',
                        '0x0000000000000000000000000000000000000803',
                        '0x0000000000000000000000000000000000000804',
                        '0x0000000000000000000000000000000000000805',
                        '0x0000000000000000000000000000000000000806',
                        '0x0000000000000000000000000000000000000807'
                    ],
                    'history_serve_window': '8192'
                },
                'preinstalls': []
            }
        # Note: EVM accounts are NOT added to genesis - they are created at runtime
        
        # Add to auth accounts
        auth_account = create_auth_account(cosmos_address)
        new_genesis['app_state']['auth']['accounts'].append(auth_account)
        
        # Add to bank balances
        bank_balance = create_bank_balance(cosmos_address, balance_wei)
        new_genesis['app_state']['bank']['balances'].append(bank_balance)
        
        # Update existing addresses sets
        existing_eth_addresses.add(eth_addr_normalized)
        existing_cosmos_addresses.add(cosmos_address)
        total_added_balance += balance_int
        added_count += 1
        
        print(f"Added address: {eth_address} -> {cosmos_address} (balance: {balance_wei} wei)")
    
    # Update total supply
    supply_section = new_genesis['app_state']['bank']['supply']
    if not supply_section:
        # Initialize supply section if empty
        supply_section.append({'denom': 'ashm', 'amount': '0'})
    
    current_supply = int(supply_section[0]['amount'])
    new_total_supply = current_supply + total_added_balance
    supply_section[0]['amount'] = str(new_total_supply)
    
    print(f"\nMerge complete:")
    print(f"  Added: {added_count} addresses")
    print(f"  Skipped: {skipped_count} addresses (duplicates or conversion errors)")
    print(f"  Total added balance: {total_added_balance:,} ashm")
    print(f"  Updated total supply from {current_supply:,} to {new_total_supply:,} ashm")
    
    # Sort accounts and balances for consistency
    new_genesis['app_state']['auth']['accounts'].sort(key=lambda x: x['address'])
    new_genesis['app_state']['bank']['balances'].sort(key=lambda x: x['address'])
    
    # Save merged genesis
    print(f"Saving merged genesis to {output_path}")
    try:
        with open(output_path, 'w') as f:
            json.dump(new_genesis, f, indent=2)
        print(f"Successfully saved merged genesis to {output_path}")
    except Exception as e:
        print(f"Error saving merged genesis: {e}", file=sys.stderr)
        sys.exit(1)

def main():
    parser = argparse.ArgumentParser(description="Merge old genesis addresses into new genesis format")
    parser.add_argument("old_genesis", help="Path to old genesis file (simple format)")
    parser.add_argument("new_genesis", help="Path to new genesis file (Cosmos SDK format)")
    parser.add_argument("output", help="Path for output merged genesis file")
    
    args = parser.parse_args()
    
    merge_genesis_files(args.old_genesis, args.new_genesis, args.output)

if __name__ == "__main__":
    main()
