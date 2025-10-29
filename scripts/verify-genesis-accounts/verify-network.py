#!/usr/bin/env python3
"""
Comprehensive Network Data Verification Script for Shardeum

This script verifies accounts across three sources:
1. Shardeum Database (accounts.sqlite3) - Including account type 0 and 1
2. Genesis Files (split or monolithic) - After unclaimed rewards update
3. Live Network (REST API) - Current on-chain state

Features:
- Multi-source verification (DB, Genesis, Network)
- Account type filtering (0=EOA, 1=Contract)
- Balance and nonce verification
- Unclaimed rewards consideration
- Concurrent network queries
- Detailed mismatch reporting
"""

import json
import asyncio
import argparse
import sqlite3
import sys
import os
import glob
from pathlib import Path
from typing import Dict, List, Tuple, Optional, Set
from dataclasses import dataclass, field
import aiohttp
from decimal import Decimal

# Constants
ASHM_DECIMALS = 18
COSMOS_PREFIX = "shardeum"
CHARSET = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
GEN = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3]


@dataclass
class VerificationConfig:
    """Configuration for comprehensive verification"""
    genesis_file: Optional[Path]
    db_file: Optional[Path]
    rest_api_url: Optional[str]
    unclaimed_rewards_file: Optional[Path]
    manual_transfers_file: Optional[Path]
    output_file: Optional[Path]
    account_types: List[int]  # Account types to check (0=EOA, 1=Contract)
    check_nonce: bool
    balance_multiplier: int
    concurrency: int
    timeout: int
    verbose: bool
    check_db: bool
    check_genesis: bool
    check_network: bool


@dataclass
class AccountData:
    """Account information from various sources"""
    cosmos_address: str
    eth_address: str

    # Database values
    db_balance: Optional[int] = None
    db_nonce: Optional[int] = None
    db_account_type: Optional[int] = None

    # Genesis values
    genesis_balance: Optional[int] = None
    genesis_nonce: Optional[int] = None

    # Network values
    network_balance: Optional[int] = None
    network_nonce: Optional[int] = None

    # Unclaimed rewards
    unclaimed_rewards: Optional[int] = None

    # Manual transfers (positive = received, negative = sent)
    manual_transfer_amount: int = 0

    # Flags
    in_db: bool = False
    in_genesis: bool = False
    in_network: bool = False


@dataclass
class VerificationResult:
    """Results of comprehensive verification"""
    total_accounts: int

    # Presence counts
    in_db_count: int
    in_genesis_count: int
    in_network_count: int
    in_all_sources: int

    # Match counts
    db_genesis_balance_match: int = 0
    db_network_balance_match: int = 0
    genesis_network_balance_match: int = 0

    db_genesis_nonce_match: int = 0
    db_network_nonce_match: int = 0
    genesis_network_nonce_match: int = 0

    # Mismatch details
    balance_mismatches: List[Dict] = field(default_factory=list)
    nonce_mismatches: List[Dict] = field(default_factory=list)
    missing_accounts: Dict[str, List[str]] = field(default_factory=dict)

    # Supply information
    db_supply: int = 0
    genesis_supply: int = 0
    network_supply: int = 0


# Bech32 encoding functions (from verify_genesis_accounts.py)
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


def extract_db_accounts(db_path: str, account_types: List[int],
                       balance_multiplier: int = 1) -> Dict[str, AccountData]:
    """
    Extract accounts from Shardeum database

    Args:
        db_path: Path to accounts.sqlite3
        account_types: List of account types to include (0=EOA, 1=Contract)
        balance_multiplier: Multiplier for balances

    Returns:
        Dictionary mapping cosmos address to AccountData
    """
    print(f"\nExtracting accounts from database: {db_path}")
    print(f"Account types: {account_types}")

    if not os.path.exists(db_path):
        print(f"Warning: Database file not found: {db_path}")
        return {}

    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()

    # Build WHERE clause for account types
    type_conditions = " OR ".join([f"json_extract(data, '$.accountType') = {t}" for t in account_types])

    query = f"""
        SELECT
          accountId,
          json_extract(data, '$.accountType') as account_type,
          json_extract(data, '$.account.balance.value') as balance_hex,
          json_extract(data, '$.account.nonce.value') as nonce_hex
        FROM accounts
        WHERE ({type_conditions})
          AND json_extract(data, '$.account.balance.value') IS NOT NULL
        ORDER BY CAST('0x' || COALESCE(json_extract(data, '$.account.balance.value'), '0') AS INTEGER) DESC
    """

    cursor.execute(query)

    accounts = {}
    processed = 0

    for row in cursor:
        account_id = row[0]
        account_type = row[1]
        balance_hex = row[2]
        nonce_hex = row[3]

        # Extract ETH address (first 40 chars of accountId)
        eth_addr = account_id[:40] if account_id else None
        if not eth_addr:
            continue

        cosmos_addr = eth_to_cosmos_address(eth_addr)
        if not cosmos_addr:
            continue

        # Parse balance
        try:
            if not balance_hex or balance_hex == '0' or balance_hex == '':
                balance_dec = 0
            else:
                balance_dec = int(balance_hex, 16)
        except ValueError:
            print(f"Warning: Invalid balance hex '{balance_hex}' for {eth_addr}", file=sys.stderr)
            balance_dec = 0

        # Apply balance multiplier
        balance_dec = balance_dec * balance_multiplier

        # Parse nonce
        try:
            nonce_dec = int(nonce_hex, 16) if nonce_hex and nonce_hex != '0' else 0
        except ValueError:
            print(f"Warning: Invalid nonce hex {nonce_hex} for {eth_addr}", file=sys.stderr)
            nonce_dec = 0

        accounts[cosmos_addr] = AccountData(
            cosmos_address=cosmos_addr,
            eth_address=f"0x{eth_addr}",
            db_balance=balance_dec,
            db_nonce=nonce_dec,
            db_account_type=account_type,
            in_db=True
        )

        processed += 1
        if processed % 10000 == 0:
            print(f"Processed {processed} accounts...", file=sys.stderr)

    conn.close()
    print(f"Extracted {len(accounts)} accounts from database")
    return accounts


def load_genesis_accounts(genesis_path: str, balance_multiplier: int = 1) -> Tuple[Dict[str, AccountData], int]:
    """
    Load accounts from genesis (monolithic or split format)

    Returns:
        Tuple of (accounts dict, total supply)
    """
    print(f"\nLoading genesis accounts from: {genesis_path}")

    if not os.path.exists(genesis_path):
        print(f"Warning: Genesis file not found: {genesis_path}")
        return {}, 0

    with open(genesis_path, 'r') as f:
        genesis = json.load(f)

    accounts = {}
    total_supply = 0

    # Check for split genesis
    auth_accounts = genesis.get('app_state', {}).get('auth', {}).get('accounts', [])
    bank_balances = genesis.get('app_state', {}).get('bank', {}).get('balances', [])

    if auth_accounts or bank_balances:
        # Monolithic genesis
        print("Loading from monolithic genesis file...")

        # Load ALL accounts from auth.accounts (including zero balance)
        # This ensures we capture accounts that exist in genesis but have no balance
        for acc in auth_accounts:
            address = acc.get('address')
            sequence = int(acc.get('sequence', '0'))
            if address not in accounts:
                accounts[address] = AccountData(
                    cosmos_address=address,
                    eth_address="",  # Will be filled if needed
                    genesis_nonce=sequence,
                    genesis_balance=0,  # Default to 0, will be updated from bank.balances if exists
                    in_genesis=True
                )
            else:
                accounts[address].genesis_nonce = sequence
                accounts[address].in_genesis = True

        # Load balances from bank.balances (only non-zero balances are here)
        for balance_entry in bank_balances:
            address = balance_entry.get('address')
            coins = balance_entry.get('coins', [])

            for coin in coins:
                if coin.get('denom') == 'ashm':
                    amount = int(coin.get('amount', 0))

                    if address not in accounts:
                        # This account has balance but wasn't in auth.accounts
                        # This shouldn't normally happen but handle it
                        accounts[address] = AccountData(
                            cosmos_address=address,
                            eth_address="",
                            genesis_balance=amount,
                            in_genesis=True
                        )
                    else:
                        # Update the balance for existing account
                        accounts[address].genesis_balance = amount

                    total_supply += amount
                    break
    else:
        # Split genesis format
        print("Loading from split genesis files...")

        genesis_dir = os.path.dirname(genesis_path)
        genesis_name = os.path.basename(genesis_path)

        # Determine base name
        base_name = genesis_name.replace('.genesis.json', '')
        if base_name == genesis_name:
            base_name = genesis_name.replace('.json', '')
        if base_name.endswith('.genesis'):
            base_name = base_name[:-8]

        # Find account files
        account_pattern = os.path.join(genesis_dir, f"{base_name}.genesis.accounts.*.json")
        account_files = sorted(glob.glob(account_pattern))

        if not account_files:
            print(f"Warning: No split account files found matching: {account_pattern}")
        else:
            print(f"Found {len(account_files)} split account files")

            for account_file in account_files:
                print(f"Loading {os.path.basename(account_file)}...")
                with open(account_file, 'r') as f:
                    chunk_data = json.load(f)

                # Process accounts (nonces) - load ALL accounts including zero balance
                for acc in chunk_data.get('accounts', []):
                    address = acc.get('address')
                    if address:
                        sequence = int(acc.get('sequence', '0'))
                        if address not in accounts:
                            accounts[address] = AccountData(
                                cosmos_address=address,
                                eth_address="",
                                genesis_nonce=sequence,
                                genesis_balance=0,  # Default to 0, will be updated from balances if exists
                                in_genesis=True
                            )
                        else:
                            accounts[address].genesis_nonce = sequence
                            accounts[address].in_genesis = True

                # Process balances (only non-zero balances)
                for balance_entry in chunk_data.get('balances', []):
                    address = balance_entry.get('address')
                    if address:
                        coins = balance_entry.get('coins', [])

                        for coin in coins:
                            if coin.get('denom') == 'ashm':
                                amount = int(coin.get('amount', 0))

                                if address not in accounts:
                                    # Account has balance but wasn't in accounts section
                                    accounts[address] = AccountData(
                                        cosmos_address=address,
                                        eth_address="",
                                        genesis_balance=amount,
                                        in_genesis=True
                                    )
                                else:
                                    # Update balance for existing account
                                    accounts[address].genesis_balance = amount

                                total_supply += amount
                                break

    # Mark all as in genesis
    for account in accounts.values():
        account.in_genesis = True

    print(f"Loaded {len(accounts)} accounts from genesis with total supply {total_supply}")
    return accounts, total_supply


def load_unclaimed_rewards(rewards_file: str) -> Dict[str, int]:
    """
    Load unclaimed rewards from genesis-addresses.json

    Returns:
        Dictionary mapping ETH address (lowercase, no 0x) to reward amount in wei
    """
    print(f"\nLoading unclaimed rewards from: {rewards_file}")

    if not os.path.exists(rewards_file):
        print(f"Warning: Rewards file not found: {rewards_file}")
        return {}

    with open(rewards_file, 'r') as f:
        rewards_data = json.load(f)

    rewards = {}
    for eth_addr, data in rewards_data.items():
        # Normalize address (remove 0x, lowercase)
        normalized_addr = eth_addr.lower().replace('0x', '')
        wei_amount = int(data['wei'])
        rewards[normalized_addr] = wei_amount

    print(f"Loaded {len(rewards)} accounts with unclaimed rewards")
    return rewards


def load_manual_transfers(transfers_file: str, balance_multiplier: int = 1) -> Dict[str, int]:
    """
    Load manual transfers from JSON file

    Expected format:
    [
      {
        "from": "0xfa22D695Af16D960E07b1D8744731A6ccF055aB1",
        "to": "0x85A23AeBA7841d45F3aa1CbbdE760F38587eF6b8",
        "amount": "500000000",  // In SHM
        "note": "Manual transfer for XYZ"
      }
    ]

    Returns:
        Dictionary mapping ETH address (lowercase, no 0x) to net transfer amount in wei
        Positive = received, Negative = sent

    Note: Manual transfers happen on the network which already has 240x scaling.
          We do NOT multiply transfer amounts - they are already in network scale.
    """
    print(f"\nLoading manual transfers from: {transfers_file}")

    if not os.path.exists(transfers_file):
        print(f"Warning: Transfers file not found: {transfers_file}")
        return {}

    with open(transfers_file, 'r') as f:
        transfers_data = json.load(f)

    # Accumulate transfers per address
    transfers = {}

    for transfer in transfers_data:
        from_addr = transfer.get('from', '').lower().replace('0x', '')
        to_addr = transfer.get('to', '').lower().replace('0x', '')

        # Amount in SHM, convert to wei (NO multiplier - transfers are on network which already has 240x)
        amount_shm = transfer.get('amount')
        if isinstance(amount_shm, str):
            amount_shm = float(amount_shm)

        amount_wei = int(amount_shm * (10 ** 18))

        note = transfer.get('note', '')

        if not from_addr or not to_addr:
            print(f"Warning: Invalid transfer entry: {transfer}", file=sys.stderr)
            continue

        # Subtract from sender
        if from_addr not in transfers:
            transfers[from_addr] = 0
        transfers[from_addr] -= amount_wei

        # Add to receiver
        if to_addr not in transfers:
            transfers[to_addr] = 0
        transfers[to_addr] += amount_wei

        print(f"  Transfer: {amount_shm:,.0f} SHM ({amount_wei:,} wei)")
        print(f"    From: {from_addr}")
        print(f"    To:   {to_addr}")
        if note:
            print(f"    Note: {note}")

    print(f"Loaded {len(transfers_data)} manual transfers affecting {len(transfers)} addresses")
    return transfers


async def fetch_all_network_addresses(rest_url: str, timeout: int) -> List[str]:
    """
    Fetch all account addresses from network using pagination

    Returns:
        List of cosmos addresses (shardeum1...)
    """
    print(f"\nFetching all addresses from network: {rest_url}")

    addresses = []
    next_key = ""
    page_count = 0

    # Use connection pooling with higher limits for pagination
    connector = aiohttp.TCPConnector(limit=10, limit_per_host=10, ttl_dns_cache=300)
    timeout_config = aiohttp.ClientTimeout(total=timeout, connect=5, sock_read=timeout)

    async with aiohttp.ClientSession(connector=connector, timeout=timeout_config) as session:
        while True:
            page_count += 1

            # Build URL with proper parameter encoding
            url = f"{rest_url}/cosmos/auth/v1beta1/accounts"
            params = {'pagination.limit': '1000'}  # Increased from 100 to reduce pagination rounds
            if next_key:
                params['pagination.key'] = next_key

            try:
                async with session.get(url, params=params) as response:
                    if response.status != 200:
                        error_text = await response.text()
                        print(f"Error: HTTP {response.status} from {url}")
                        print(f"Response: {error_text}")
                        break

                    data = await response.json()

                    # Extract addresses from accounts - optimized
                    for account in data.get('accounts', []):
                        # Handle different account shapes
                        addr = (account.get('base_account', {}).get('address') or
                               account.get('address') or
                               account.get('value', {}).get('address'))
                        if addr:
                            addresses.append(addr)

                    # Get next pagination key
                    next_key = data.get('pagination', {}).get('next_key', '')

                    if not next_key or next_key == 'null':
                        break

                    if page_count % 5 == 0:  # Report less frequently
                        print(f"Fetched {page_count} pages, {len(addresses)} addresses so far...")

            except asyncio.TimeoutError:
                print(f"Timeout fetching addresses (page {page_count})")
                if page_count == 1:
                    break
                else:
                    print(f"Continuing with {len(addresses)} addresses fetched so far...")
                    break
            except Exception as e:
                print(f"Error fetching addresses (page {page_count}): {e}")
                if page_count == 1:
                    # If first page fails, abort
                    break
                else:
                    # If later page fails, return what we have so far
                    print(f"Continuing with {len(addresses)} addresses fetched so far...")
                    break

    print(f"Fetched {len(addresses)} total addresses from network")
    return addresses


async def query_network_account(session: aiohttp.ClientSession, rest_url: str,
                               cosmos_address: str, timeout: int) -> Tuple[Optional[int], Optional[int]]:
    """
    Query account balance and nonce from network with retry logic

    Returns:
        Tuple of (balance, sequence/nonce)
    """
    max_retries = 3
    retry_delay = 0.5

    for attempt in range(max_retries):
        try:
            # Query account info for sequence
            account_url = f"{rest_url}/cosmos/auth/v1beta1/accounts/{cosmos_address}"
            async with session.get(account_url) as response:
                if response.status != 200:
                    if attempt < max_retries - 1:
                        await asyncio.sleep(retry_delay)
                        continue
                    return None, None

                account_data = await response.json()
                account_info = account_data.get('account', {})

                # Extract sequence (nonce)
                sequence = int(account_info.get('base_account', {}).get('sequence',
                              account_info.get('sequence', '0')))

            # Query balance
            balance_url = f"{rest_url}/cosmos/bank/v1beta1/balances/{cosmos_address}"
            async with session.get(balance_url) as response:
                if response.status != 200:
                    if attempt < max_retries - 1:
                        await asyncio.sleep(retry_delay)
                        continue
                    return None, sequence

                balance_data = await response.json()
                balances = balance_data.get('balances', [])

                balance = 0
                for coin in balances:
                    if coin.get('denom') == 'ashm':
                        balance = int(coin.get('amount', 0))
                        break

                return balance, sequence

        except asyncio.TimeoutError:
            if attempt < max_retries - 1:
                await asyncio.sleep(retry_delay)
                continue
            return None, None
        except Exception as e:
            if attempt < max_retries - 1:
                await asyncio.sleep(retry_delay)
                continue
            return None, None

    return None, None


async def verify_network_accounts(accounts: Dict[str, AccountData], rest_url: str,
                                  config: VerificationConfig) -> Dict[str, AccountData]:
    """
    Verify accounts against live network with rate limiting
    """
    print(f"\nQuerying network accounts from {rest_url}")
    print(f"Concurrency: {config.concurrency} requests")

    # First, fetch all addresses from network to know what exists
    all_network_addresses = await fetch_all_network_addresses(rest_url, config.timeout)
    network_address_set = set(all_network_addresses)

    # Mark accounts that exist in network
    for address in network_address_set:
        if address not in accounts:
            accounts[address] = AccountData(
                cosmos_address=address,
                eth_address="",
                in_network=True
            )
        else:
            accounts[address].in_network = True

    # Now query details for accounts we're tracking - REUSE SINGLE SESSION
    semaphore = asyncio.Semaphore(config.concurrency)

    # Create a single session with connection pooling for all requests
    connector = aiohttp.TCPConnector(limit=config.concurrency, limit_per_host=config.concurrency)
    async with aiohttp.ClientSession(connector=connector) as session:
        async def query_one(address: str, account: AccountData, idx: int) -> None:
            async with semaphore:
                if not account.in_network:
                    return

                # Reuse the shared session instead of creating a new one
                balance, nonce = await query_network_account(
                    session, rest_url, address, config.timeout
                )

                account.network_balance = balance
                account.network_nonce = nonce

                if (idx + 1) % 100 == 0:
                    print(f"Queried {idx + 1}/{len(accounts)} accounts...")

        # Only query accounts that are in network
        tasks = [
            query_one(addr, acc, idx)
            for idx, (addr, acc) in enumerate(accounts.items())
            if acc.in_network
        ]

        await asyncio.gather(*tasks)

    print(f"Completed network queries for {len([a for a in accounts.values() if a.in_network])} accounts")
    return accounts


def analyze_results(accounts: Dict[str, AccountData],
                    unclaimed_rewards: Dict[str, int],
                    manual_transfers: Dict[str, int],
                    config: VerificationConfig) -> VerificationResult:
    """
    Analyze verification results across all sources

    Note: DB balances have already had the balance_multiplier applied during extraction.
    Genesis and network balances are as-is from those sources.
    Manual transfers are applied to adjust expected network balances.
    """
    print("\nAnalyzing verification results...")
    print(f"Balance multiplier: {config.balance_multiplier}x")
    if manual_transfers:
        print(f"Manual transfers: {len(manual_transfers)} addresses affected")

    result = VerificationResult(
        total_accounts=len(accounts),
        in_db_count=sum(1 for a in accounts.values() if a.in_db),
        in_genesis_count=sum(1 for a in accounts.values() if a.in_genesis),
        in_network_count=sum(1 for a in accounts.values() if a.in_network),
        in_all_sources=sum(1 for a in accounts.values() if a.in_db and a.in_genesis and a.in_network),
        missing_accounts={
            'in_db_not_genesis': [],
            'in_db_not_network': [],
            'in_genesis_not_db': [],
            'in_genesis_not_network': [],
            'in_network_not_db': [],
            'in_network_not_genesis': []
        }
    )

    for addr, account in accounts.items():
        # Handle unclaimed rewards
        # Unclaimed rewards are in original scale (wei) and need to be multiplied by 240
        # Genesis already has: (DB_balance * 240) + (unclaimed_rewards * 240)
        # Network also has the same scaling
        eth_addr_normalized = account.eth_address.lower().replace('0x', '')

        unclaimed_reward_amount = 0
        if eth_addr_normalized in unclaimed_rewards:
            # Store the original reward amount for tracking
            unclaimed_reward_amount = unclaimed_rewards[eth_addr_normalized]
            account.unclaimed_rewards = unclaimed_reward_amount

        # Handle manual transfers (already scaled with multiplier)
        manual_transfer_amount = manual_transfers.get(eth_addr_normalized, 0)
        if manual_transfer_amount != 0:
            account.manual_transfer_amount = manual_transfer_amount

        # Calculate expected balances for comparison
        # DB balance is already multiplied by balance_multiplier during extraction
        db_balance_scaled = account.db_balance or 0

        # Expected genesis balance = DB_balance_scaled + (unclaimed_rewards * multiplier)
        expected_genesis_balance = db_balance_scaled + (unclaimed_reward_amount * config.balance_multiplier)

        # Expected network balance = Expected_genesis + manual_transfers
        # (manual_transfers are already scaled with multiplier)
        expected_network_balance = expected_genesis_balance + manual_transfer_amount

        # Actual balances from files/network
        actual_genesis_balance = account.genesis_balance or 0

        # Calculate supplies
        if account.db_balance is not None:
            result.db_supply += account.db_balance
        if account.genesis_balance is not None:
            result.genesis_supply += account.genesis_balance
        if account.network_balance is not None:
            result.network_supply += account.network_balance

        # Check balance matches (only if both values are available)
        # Note: We compare expected_genesis_balance (DB*240 + rewards*240) against actual genesis/network
        if config.check_db and config.check_genesis:
            if account.in_db and account.in_genesis and db_balance_scaled is not None and actual_genesis_balance is not None:
                if expected_genesis_balance == actual_genesis_balance:
                    result.db_genesis_balance_match += 1
                else:
                    result.balance_mismatches.append({
                        'address': addr,
                        'type': 'db_vs_genesis',
                        'db_balance_scaled': db_balance_scaled,
                        'expected_genesis_balance': expected_genesis_balance,
                        'actual_genesis_balance': actual_genesis_balance,
                        'difference': actual_genesis_balance - expected_genesis_balance,
                        'unclaimed_rewards': unclaimed_reward_amount,
                        'unclaimed_rewards_scaled': unclaimed_reward_amount * config.balance_multiplier if unclaimed_reward_amount else 0
                    })

        if config.check_db and config.check_network:
            if account.in_db and account.in_network and db_balance_scaled is not None and account.network_balance is not None:
                # Compare expected (DB*240 + rewards*240 + transfers) against network
                if expected_network_balance == account.network_balance:
                    result.db_network_balance_match += 1
                else:
                    result.balance_mismatches.append({
                        'address': addr,
                        'type': 'db_vs_network',
                        'db_balance_scaled': db_balance_scaled,
                        'expected_balance': expected_network_balance,
                        'network_balance': account.network_balance,
                        'difference': account.network_balance - expected_network_balance,
                        'unclaimed_rewards': unclaimed_reward_amount,
                        'unclaimed_rewards_scaled': unclaimed_reward_amount * config.balance_multiplier if unclaimed_reward_amount else 0,
                        'manual_transfer': manual_transfer_amount
                    })

        if config.check_genesis and config.check_network:
            if account.in_genesis and account.in_network and actual_genesis_balance is not None and account.network_balance is not None:
                if actual_genesis_balance == account.network_balance:
                    result.genesis_network_balance_match += 1
                else:
                    result.balance_mismatches.append({
                        'address': addr,
                        'type': 'genesis_vs_network',
                        'genesis_balance': actual_genesis_balance,
                        'network_balance': account.network_balance,
                        'difference': account.network_balance - actual_genesis_balance
                    })

        # Check nonce matches (if enabled)
        if config.check_nonce:
            if config.check_db and config.check_genesis:
                if account.in_db and account.in_genesis:
                    if account.db_nonce == account.genesis_nonce:
                        result.db_genesis_nonce_match += 1
                    else:
                        result.nonce_mismatches.append({
                            'address': addr,
                            'type': 'db_vs_genesis',
                            'db_nonce': account.db_nonce,
                            'genesis_nonce': account.genesis_nonce
                        })

            if config.check_db and config.check_network:
                if account.in_db and account.in_network:
                    if account.db_nonce == account.network_nonce:
                        result.db_network_nonce_match += 1
                    else:
                        result.nonce_mismatches.append({
                            'address': addr,
                            'type': 'db_vs_network',
                            'db_nonce': account.db_nonce,
                            'network_nonce': account.network_nonce
                        })

            if config.check_genesis and config.check_network:
                if account.in_genesis and account.in_network:
                    if account.genesis_nonce == account.network_nonce:
                        result.genesis_network_nonce_match += 1
                    else:
                        result.nonce_mismatches.append({
                            'address': addr,
                            'type': 'genesis_vs_network',
                            'genesis_nonce': account.genesis_nonce,
                            'network_nonce': account.network_nonce
                        })

        # Track missing accounts
        if account.in_db and not account.in_genesis:
            result.missing_accounts['in_db_not_genesis'].append(addr)
        if account.in_db and not account.in_network:
            result.missing_accounts['in_db_not_network'].append(addr)
        if account.in_genesis and not account.in_db:
            result.missing_accounts['in_genesis_not_db'].append(addr)
        if account.in_genesis and not account.in_network:
            result.missing_accounts['in_genesis_not_network'].append(addr)
        if account.in_network and not account.in_db:
            result.missing_accounts['in_network_not_db'].append(addr)
        if account.in_network and not account.in_genesis:
            result.missing_accounts['in_network_not_genesis'].append(addr)

    return result


def print_report(result: VerificationResult, config: VerificationConfig):
    """Print comprehensive verification report"""
    print("\n" + "="*80)
    print("COMPREHENSIVE VERIFICATION REPORT")
    print("="*80)

    # Configuration
    print(f"\n⚙️  Configuration:")
    sources_checked = []
    if config.check_db:
        sources_checked.append("Database")
    if config.check_genesis:
        sources_checked.append("Genesis")
    if config.check_network:
        sources_checked.append("Network")
    print(f"  Sources verified: {', '.join(sources_checked)}")

    if config.balance_multiplier > 1:
        print(f"  Balance multiplier: {config.balance_multiplier}x")
    if config.check_nonce:
        print(f"  Nonce verification: Enabled")

    # Account counts
    print("\n📊 Account Counts by Source:")
    if config.check_db:
        print(f"  Database: {result.in_db_count:,} accounts")
    if config.check_genesis:
        print(f"  Genesis: {result.in_genesis_count:,} accounts")
    if config.check_network:
        print(f"  Network: {result.in_network_count:,} accounts")

    # Supply summary
    print("\n💰 Total Supply by Source:")
    if config.check_db:
        print(f"  Database: {result.db_supply / 10**18:,.2f} SHM")
    if config.check_genesis:
        print(f"  Genesis: {result.genesis_supply / 10**18:,.2f} SHM")
    if config.check_network:
        print(f"  Network: {result.network_supply / 10**18:,.2f} SHM")

    # Missing accounts (accounts that don't exist in one source but exist in another)
    has_missing = any(len(v) > 0 for v in result.missing_accounts.values())
    if has_missing:
        print("\n❌ Missing Accounts (present in one source but not another):")

        # Only show relevant comparisons based on what's being checked
        if config.check_db and config.check_genesis:
            missing_in_genesis = result.missing_accounts.get('in_db_not_genesis', [])
            missing_in_db = result.missing_accounts.get('in_genesis_not_db', [])
            if missing_in_genesis:
                print(f"\n  In Database but NOT in Genesis: {len(missing_in_genesis):,}")
                # Show all if output file specified or verbose, otherwise limit to 10
                show_all = config.verbose or config.output_file
                limit = None if show_all else 10
                items_to_show = missing_in_genesis if show_all else missing_in_genesis[:10]

                for addr in items_to_show:
                    print(f"    - {addr}")
                if not show_all and len(missing_in_genesis) > 10:
                    print(f"    ... and {len(missing_in_genesis) - 10} more")
            if missing_in_db:
                print(f"\n  In Genesis but NOT in Database: {len(missing_in_db):,}")
                show_all = config.verbose or config.output_file
                items_to_show = missing_in_db if show_all else missing_in_db[:10]

                for addr in items_to_show:
                    print(f"    - {addr}")
                if not show_all and len(missing_in_db) > 10:
                    print(f"    ... and {len(missing_in_db) - 10} more")

        if config.check_db and config.check_network:
            missing_in_network = result.missing_accounts.get('in_db_not_network', [])
            missing_in_db_net = result.missing_accounts.get('in_network_not_db', [])
            if missing_in_network:
                print(f"\n  In Database but NOT on Network: {len(missing_in_network):,}")
                show_all = config.verbose or config.output_file
                items_to_show = missing_in_network if show_all else missing_in_network[:10]

                for addr in items_to_show:
                    print(f"    - {addr}")
                if not show_all and len(missing_in_network) > 10:
                    print(f"    ... and {len(missing_in_network) - 10} more")
            if missing_in_db_net:
                print(f"\n  On Network but NOT in Database: {len(missing_in_db_net):,}")
                show_all = config.verbose or config.output_file
                items_to_show = missing_in_db_net if show_all else missing_in_db_net[:10]

                for addr in items_to_show:
                    print(f"    - {addr}")
                if not show_all and len(missing_in_db_net) > 10:
                    print(f"    ... and {len(missing_in_db_net) - 10} more")

        if config.check_genesis and config.check_network:
            missing_in_network_gen = result.missing_accounts.get('in_genesis_not_network', [])
            missing_in_genesis_net = result.missing_accounts.get('in_network_not_genesis', [])
            if missing_in_network_gen:
                print(f"\n  In Genesis but NOT on Network: {len(missing_in_network_gen):,}")
                show_all = config.verbose or config.output_file
                items_to_show = missing_in_network_gen if show_all else missing_in_network_gen[:10]

                for addr in items_to_show:
                    print(f"    - {addr}")
                if not show_all and len(missing_in_network_gen) > 10:
                    print(f"    ... and {len(missing_in_network_gen) - 10} more")
            if missing_in_genesis_net:
                print(f"\n  On Network but NOT in Genesis: {len(missing_in_genesis_net):,}")
                print(f"    (These are accounts created after genesis)")
                show_all = config.verbose or config.output_file
                items_to_show = missing_in_genesis_net if show_all else missing_in_genesis_net[:10]

                for addr in items_to_show:
                    print(f"    - {addr}")
                if not show_all and len(missing_in_genesis_net) > 10:
                    print(f"    ... and {len(missing_in_genesis_net) - 10} more")
    else:
        print("\n✅ No missing accounts - all accounts exist in all checked sources")

    # Balance mismatches
    if result.balance_mismatches:
        print(f"\n⚠️  Balance Mismatches: {len(result.balance_mismatches):,}")

        # Group by type
        by_type = {}
        for mismatch in result.balance_mismatches:
            mtype = mismatch['type']
            if mtype not in by_type:
                by_type[mtype] = []
            by_type[mtype].append(mismatch)

        for mtype, mismatches in by_type.items():
            print(f"\n  {mtype.replace('_', ' ').upper()} ({len(mismatches):,} mismatches):")
            # Show all if output file specified or verbose, otherwise limit to 5
            show_all = config.verbose or config.output_file
            items_to_show = mismatches if show_all else mismatches[:5]

            for mismatch in items_to_show:
                print(f"    Address: {mismatch['address']}")
                for key, value in mismatch.items():
                    if key not in ['address', 'type']:
                        if value is None:
                            print(f"      {key}: None (not available)")
                        elif 'balance' in key.lower() or 'difference' in key.lower() or 'reward' in key.lower():
                            if isinstance(value, (int, float)):
                                print(f"      {key}: {value:,} ashm ({value/10**18:.6f} SHM)")
                            else:
                                print(f"      {key}: {value}")
                        else:
                            print(f"      {key}: {value}")
            if not show_all and len(mismatches) > 5:
                print(f"    ... and {len(mismatches) - 5} more")

    # Nonce mismatches
    if config.check_nonce and result.nonce_mismatches:
        print(f"\n⚠️  Nonce Mismatches: {len(result.nonce_mismatches):,}")
        # Show all if output file specified or verbose, otherwise limit to 5
        show_all = config.verbose or config.output_file
        items_to_show = result.nonce_mismatches if show_all else result.nonce_mismatches[:5]

        for mismatch in items_to_show:
            print(f"  {mismatch['type']}: {mismatch['address']}")
            for key, value in mismatch.items():
                if key not in ['address', 'type']:
                    print(f"    {key}: {value}")
        if not show_all and len(result.nonce_mismatches) > 5:
            print(f"  ... and {len(result.nonce_mismatches) - 5} more")

    # Verification Summary
    print("\n" + "="*80)
    print("VERIFICATION SUMMARY")
    print("="*80)

    # Calculate what was actually compared
    summary_lines = []

    if config.check_db and config.check_genesis:
        db_gen_missing = len(result.missing_accounts.get('in_db_not_genesis', [])) + len(result.missing_accounts.get('in_genesis_not_db', []))
        db_gen_balance_mismatches = len([m for m in result.balance_mismatches if m.get('type') == 'db_vs_genesis'])
        summary_lines.append(f"Database ↔ Genesis:")
        summary_lines.append(f"  - Accounts matched: {result.db_genesis_balance_match:,}")
        summary_lines.append(f"  - Missing accounts: {db_gen_missing:,}")
        summary_lines.append(f"  - Balance mismatches: {db_gen_balance_mismatches:,}")
        if config.check_nonce:
            db_gen_nonce_mismatches = len([m for m in result.nonce_mismatches if m.get('type') == 'db_vs_genesis'])
            summary_lines.append(f"  - Nonce mismatches: {db_gen_nonce_mismatches:,}")

    if config.check_db and config.check_network:
        db_net_missing = len(result.missing_accounts.get('in_db_not_network', [])) + len(result.missing_accounts.get('in_network_not_db', []))
        db_net_balance_mismatches = len([m for m in result.balance_mismatches if m.get('type') == 'db_vs_network'])
        summary_lines.append(f"Database ↔ Network:")
        summary_lines.append(f"  - Accounts matched: {result.db_network_balance_match:,}")
        summary_lines.append(f"  - Missing accounts: {db_net_missing:,}")
        summary_lines.append(f"  - Balance mismatches: {db_net_balance_mismatches:,}")
        if config.check_nonce:
            db_net_nonce_mismatches = len([m for m in result.nonce_mismatches if m.get('type') == 'db_vs_network'])
            summary_lines.append(f"  - Nonce mismatches: {db_net_nonce_mismatches:,}")

    if config.check_genesis and config.check_network:
        gen_net_missing = len(result.missing_accounts.get('in_genesis_not_network', [])) + len(result.missing_accounts.get('in_network_not_genesis', []))
        gen_net_balance_mismatches = len([m for m in result.balance_mismatches if m.get('type') == 'genesis_vs_network'])
        summary_lines.append(f"Genesis ↔ Network:")
        summary_lines.append(f"  - Accounts matched: {result.genesis_network_balance_match:,}")
        summary_lines.append(f"  - Missing accounts: {gen_net_missing:,}")
        summary_lines.append(f"  - Balance mismatches: {gen_net_balance_mismatches:,}")
        if config.check_nonce:
            gen_net_nonce_mismatches = len([m for m in result.nonce_mismatches if m.get('type') == 'genesis_vs_network'])
            summary_lines.append(f"  - Nonce mismatches: {gen_net_nonce_mismatches:,}")

    print("\n" + "\n".join(summary_lines))

    # Overall status
    has_issues = (len(result.balance_mismatches) > 0 or
                  (config.check_nonce and len(result.nonce_mismatches) > 0) or
                  any(len(v) > 0 for v in result.missing_accounts.values()))

    print("\n" + "="*80)
    if has_issues:
        print("STATUS: ⚠️  VERIFICATION COMPLETED WITH ISSUES")
        print("\nNote: Balance/nonce mismatches between Genesis and Network are normal")
        print("due to transactions, fees, and staking rewards after genesis.")
    else:
        print("STATUS: ✅ VERIFICATION PASSED - ALL CHECKS SUCCESSFUL")

    print("="*80 + "\n")


async def run_verification(config: VerificationConfig) -> int:
    """
    Run comprehensive verification

    Returns:
        Exit code (0 for success, 1 for issues found)
    """
    accounts = {}
    unclaimed_rewards = {}
    manual_transfers = {}
    genesis_supply = 0

    try:
        # Load database accounts
        if config.check_db and config.db_file:
            db_accounts = extract_db_accounts(
                str(config.db_file),
                config.account_types,
                config.balance_multiplier
            )
            accounts.update(db_accounts)

        # Load genesis accounts
        if config.check_genesis and config.genesis_file:
            genesis_accounts, genesis_supply = load_genesis_accounts(
                str(config.genesis_file),
                config.balance_multiplier
            )

            # Merge with existing accounts
            for addr, gen_acc in genesis_accounts.items():
                if addr in accounts:
                    accounts[addr].genesis_balance = gen_acc.genesis_balance
                    accounts[addr].genesis_nonce = gen_acc.genesis_nonce
                    accounts[addr].in_genesis = True
                else:
                    accounts[addr] = gen_acc

        # Load unclaimed rewards
        if config.unclaimed_rewards_file:
            unclaimed_rewards = load_unclaimed_rewards(str(config.unclaimed_rewards_file))

        # Load manual transfers
        if config.manual_transfers_file:
            manual_transfers = load_manual_transfers(
                str(config.manual_transfers_file),
                config.balance_multiplier
            )

        # Query network
        if config.check_network and config.rest_api_url:
            accounts = await verify_network_accounts(accounts, config.rest_api_url, config)

        # Analyze results
        result = analyze_results(accounts, unclaimed_rewards, manual_transfers, config)

        # Redirect output to file if specified
        original_stdout = sys.stdout
        output_file_handle = None

        try:
            if config.output_file:
                output_file_handle = open(config.output_file, 'w', encoding='utf-8')
                sys.stdout = output_file_handle
                # Print a notification to stderr so user knows where output went
                print(f"Writing verification report to: {config.output_file}", file=sys.stderr)

            # Print report (to file or stdout depending on redirection)
            print_report(result, config)

        finally:
            # Restore original stdout
            sys.stdout = original_stdout
            if output_file_handle:
                output_file_handle.close()
                print(f"Verification report saved to: {config.output_file}", file=sys.stderr)

        # Determine exit code
        has_issues = (len(result.balance_mismatches) > 0 or
                     (config.check_nonce and len(result.nonce_mismatches) > 0) or
                     any(len(v) > 0 for v in result.missing_accounts.values()))

        return 1 if has_issues else 0

    except Exception as e:
        print(f"\nError during verification: {e}", file=sys.stderr)
        if config.verbose:
            import traceback
            traceback.print_exc()
        return 1


def main():
    parser = argparse.ArgumentParser(
        description='Comprehensive Shardeum account verification across DB, Genesis, and Network',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Genesis vs Network only (no DB required)
  %(prog)s --genesis genesis.json --rest http://localhost:1317

  # Database vs Genesis only (no network required)
  %(prog)s --db accounts.sqlite3 --genesis genesis.json \\
           --balance-multiplier 240

  # All sources with unclaimed rewards
  %(prog)s --db accounts.sqlite3 --genesis genesis.json --rest http://localhost:1317 \\
           --balance-multiplier 240 --unclaimed-rewards genesis-addresses.json

  # Include contracts and check nonces
  %(prog)s --genesis genesis.json --rest http://localhost:1317 \\
           --account-types 0 1 --check-nonce

  # Database vs Network only
  %(prog)s --db accounts.sqlite3 --rest http://localhost:1317 \\
           --balance-multiplier 240 --check-genesis false
        """
    )

    parser.add_argument('--db', '--db-file', dest='db_file', type=Path,
                       help='Path to accounts.sqlite3 database (optional)')
    parser.add_argument('--genesis', '--genesis-file', dest='genesis_file', type=Path,
                       help='Path to genesis JSON file (optional)')
    parser.add_argument('--rest', '--rest-api', dest='rest_api', type=str,
                       help='REST API URL, e.g., http://localhost:1317 (optional)')
    parser.add_argument('--unclaimed-rewards', type=Path,
                       help='Path to unclaimed rewards JSON file - genesis-addresses.json (optional)')
    parser.add_argument('--manual-transfers', type=Path,
                       help='Path to manual transfers JSON file (optional)')
    parser.add_argument('--output', '-o', type=Path,
                       help='Output file to save verification results (optional, default: print to stdout)')
    parser.add_argument('--account-types', type=int, nargs='+', default=[0],
                       help='Account types to check (0=EOA, 1=Contract). Default: 0')
    parser.add_argument('--check-nonce', action='store_true',
                       help='Also verify nonces/sequences')
    parser.add_argument('--balance-multiplier', type=int, default=1,
                       help='Balance multiplier for verification (default: 1)')
    parser.add_argument('--concurrency', type=int, default=20,
                       help='Concurrent network requests (default: 20)')
    parser.add_argument('--timeout', type=int, default=10,
                       help='Network request timeout in seconds (default: 10)')
    parser.add_argument('--verbose', '-v', action='store_true',
                       help='Verbose output')
    parser.add_argument('--check-db', type=lambda x: x.lower() != 'false', default=True,
                       help='Check database (default: true)')
    parser.add_argument('--check-genesis', type=lambda x: x.lower() != 'false', default=True,
                       help='Check genesis (default: true)')
    parser.add_argument('--check-network', type=lambda x: x.lower() != 'false', default=True,
                       help='Check network (default: true)')

    args = parser.parse_args()

    # Validate at least one source is enabled
    if not (args.check_db or args.check_genesis or args.check_network):
        print("Error: At least one check must be enabled (--check-db, --check-genesis, --check-network)")
        return 1

    # Validate required files for enabled checks
    if args.check_db and not args.db_file:
        print("Error: --db required when --check-db is enabled")
        return 1
    if args.check_genesis and not args.genesis_file:
        print("Error: --genesis required when --check-genesis is enabled")
        return 1
    if args.check_network and not args.rest_api:
        print("Error: --rest required when --check-network is enabled")
        return 1

    # Create config
    config = VerificationConfig(
        genesis_file=args.genesis_file,
        db_file=args.db_file,
        rest_api_url=args.rest_api,
        unclaimed_rewards_file=args.unclaimed_rewards,
        manual_transfers_file=args.manual_transfers,
        output_file=args.output,
        account_types=args.account_types,
        check_nonce=args.check_nonce,
        balance_multiplier=args.balance_multiplier,
        concurrency=args.concurrency,
        timeout=args.timeout,
        verbose=args.verbose,
        check_db=args.check_db,
        check_genesis=args.check_genesis,
        check_network=args.check_network
    )

    # Run verification
    exit_code = asyncio.run(run_verification(config))
    return exit_code


if __name__ == '__main__':
    sys.exit(main())
