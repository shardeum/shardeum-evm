#!/usr/bin/env python3
"""
Convert Shardeum bech32 address to Ethereum hex address

This script converts Shardeum bech32 addresses (format: shardeum1...) to their 
corresponding Ethereum hex addresses (format: 0x...). This is useful for:
- Interacting with EVM-compatible contracts on Shardeum
- Making JSON-RPC calls that require hex addresses
- Converting between Cosmos SDK and Ethereum address formats

Usage:
    python generate_ethereum_address.py <shardeum_bech32_address>
    
Example:
    python generate_ethereum_address.py shardeum1zkwykx6dwzhkv4hspyu9pns34sx03u702a4d7k
    Output: 0x159c4b1b4d70af6656f0093850ce11ac0cf8f3cf

The conversion uses the same bech32 encoding/decoding logic as the Shardeum
genesis importer to ensure consistency across the ecosystem.
"""

import sys

# Bech32 encoding/decoding constants
# These are the standard bech32 character set and generator polynomial
CHARSET = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"  # Base32 character set for bech32
GEN = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3]  # Generator polynomial coefficients

def bech32_polymod(values):
    """
    Calculate bech32 polymod checksum
    
    This function implements the polymod checksum algorithm used in bech32 encoding.
    It's used to verify the integrity of bech32 addresses and generate checksums.
    
    Args:
        values: List of 5-bit values to process
        
    Returns:
        int: The polymod checksum value
    """
    chk = 1  # Initialize checksum accumulator
    for value in values:
        top = chk >> 25  # Extract top 5 bits
        chk = (chk & 0x1ffffff) << 5 ^ value  # Shift left 5 bits and XOR with value
        for i in range(5):  # Apply generator polynomial
            chk ^= GEN[i] if ((top >> i) & 1) else 0
    return chk

def convertbits(data, frombits, tobits, pad=True):
    """
    Convert between different bit group sizes
    
    This function converts data from one bit group size to another. For example,
    converting from 5-bit groups (used in bech32) to 8-bit groups (used in hex).
    
    Args:
        data: List of values in the source bit group size
        frombits: Source bit group size (e.g., 5 for bech32)
        tobits: Target bit group size (e.g., 8 for hex)
        pad: Whether to pad incomplete groups
        
    Returns:
        List of values in the target bit group size, or None if conversion fails
    """
    acc = 0  # Accumulator for bits
    bits = 0  # Number of bits currently in accumulator
    ret = []  # Result list
    maxv = (1 << tobits) - 1  # Maximum value for target bit group
    max_acc = (1 << (frombits + tobits - 1)) - 1  # Maximum accumulator value
    
    for value in data:
        # Validate input value
        if value < 0 or (value >> frombits):
            return None
        
        # Add new bits to accumulator
        acc = ((acc << frombits) | value) & max_acc
        bits += frombits
        
        # Extract complete groups
        while bits >= tobits:
            bits -= tobits
            ret.append((acc >> bits) & maxv)
    
    # Handle remaining bits
    if pad:
        if bits:
            ret.append((acc << (tobits - bits)) & maxv)
    elif bits >= frombits or ((acc << (tobits - bits)) & maxv):
        return None
    
    return ret

def bech32_decode(bech):
    """
    Decode a bech32 string into human-readable part and data
    
    This function parses a bech32 address and extracts the human-readable part (HRP)
    and the data portion, while validating the format and checksum.
    
    Args:
        bech: The bech32 string to decode (e.g., "shardeum1abc...")
        
    Returns:
        Tuple of (hrp, data) if successful, (None, None) if failed
    """
    # Validate character set and case consistency
    if ((any(ord(x) < 33 or ord(x) > 126 for x in bech)) or
            (bech.lower() != bech and bech.upper() != bech)):
        return (None, None)
    
    bech = bech.lower()  # Convert to lowercase for processing
    pos = bech.rfind('1')  # Find the separator character '1'
    
    # Validate separator position and minimum length
    if pos < 1 or pos + 7 > len(bech) or pos + 1 + 6 > len(bech):
        return (None, None)
    
    # Validate that all characters after separator are in the charset
    if not all(x in CHARSET for x in bech[pos+1:]):
        return (None, None)
    
    # Extract human-readable part and data
    hrp = bech[:pos]  # Everything before the '1'
    data = [CHARSET.find(x) for x in bech[pos+1:]]  # Convert characters to 5-bit values
    
    # Verify checksum
    if not bech32_verify_checksum(hrp, data):
        return (None, None)
    
    # Return HRP and data without checksum (last 6 values)
    return (hrp, data[:-6])

def bech32_verify_checksum(hrp, data):
    """
    Verify a bech32 string checksum
    
    This function verifies that the checksum in a bech32 address is valid
    by computing the polymod of the HRP + data and checking if it equals 1.
    
    Args:
        hrp: Human-readable part (e.g., "shardeum")
        data: Data portion including checksum
        
    Returns:
        bool: True if checksum is valid, False otherwise
    """
    return bech32_polymod(bech32_hrp_expand(hrp) + data) == 1

def bech32_hrp_expand(hrp):
    """
    Expand the HRP into values for checksum computation
    
    This function converts the human-readable part into a list of 5-bit values
    that can be used in the polymod checksum calculation.
    
    Args:
        hrp: Human-readable part (e.g., "shardeum")
        
    Returns:
        List of 5-bit values representing the HRP
    """
    return [ord(x) >> 5 for x in hrp] + [0] + [ord(x) & 31 for x in hrp]

def bech32_to_hex(bech32_addr):
    """
    Convert bech32 address to Ethereum hex address
    
    This is the main conversion function that takes a Shardeum bech32 address
    and converts it to the corresponding Ethereum hex address format.
    
    Args:
        bech32_addr: Shardeum bech32 address (e.g., "shardeum1abc...")
        
    Returns:
        str: Ethereum hex address (e.g., "0x123...") or None if conversion fails
    """
    try:
        # Decode the bech32 address
        hrp, data = bech32_decode(bech32_addr)
        if hrp is None or data is None:
            return None
        
        # Convert from 5-bit groups (bech32) to 8-bit groups (hex)
        conv = convertbits(data, 5, 8, False)
        if conv is None:
            return None
        
        # Convert bytes to hex string with 0x prefix
        return '0x' + ''.join(f'{b:02x}' for b in conv)
    except Exception as e:
        print(f"Error decoding address {bech32_addr}: {e}", file=sys.stderr)
        return None

if __name__ == "__main__":
    """
    Main execution block - handles command line arguments and performs conversion
    """
    # Validate command line arguments
    if len(sys.argv) != 2:
        print("Usage: python generate_ethereum_address.py <shardeum_bech32_address>", file=sys.stderr)
        print("", file=sys.stderr)
        print("Example:", file=sys.stderr)
        print("  python generate_ethereum_address.py shardeum1zkwykx6dwzhkv4hspyu9pns34sx03u702a4d7k", file=sys.stderr)
        print("  Output: 0x159c4b1b4d70af6656f0093850ce11ac0cf8f3cf", file=sys.stderr)
        sys.exit(1)
    
    # Get the bech32 address from command line
    bech32_addr = sys.argv[1]
    
    # Perform the conversion
    hex_addr = bech32_to_hex(bech32_addr)
    
    # Output the result
    if hex_addr:
        print(hex_addr)
    else:
        print("Failed to convert address", file=sys.stderr)
        print("Please ensure the address is a valid Shardeum bech32 address", file=sys.stderr)
        sys.exit(1)
