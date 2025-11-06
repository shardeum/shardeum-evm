# Broadcast EIP-712 Tool

A command-line tool to parse and display EIP-712 signed transactions from MetaMask.

## Installation

The tool is built automatically with the project:

```bash
make build
```

This creates `./build/broadcast-eip712`

## Usage

```bash
./build/broadcast-eip712 <signed-tx.json>
```

### Example

```bash
# Display information about a signed transaction
./build/broadcast-eip712 signed-tx1.json
```

## What It Does

✅ **Validates** the signed transaction JSON format  
✅ **Parses** transaction messages (MsgDelegate, etc.)  
✅ **Displays** all transaction details in a readable format  
✅ **Extracts** signature components (r, s, v)  
✅ **Shows** delegator, validator, amounts, fees, and gas  

## What It Doesn't Do (Yet)

❌ **Broadcasting** - Requires public key recovery and proper Cosmos SDK transaction construction  
❌ **Account querying** - Needs chain connection to verify account state  
❌ **Signature verification** - EIP-712 signature recovery not fully implemented  

## Example Output

```
===============================================================================
📋 EIP-712 Signed Transaction Details
===============================================================================

🔑 Signer:
   Ethereum Address: 0x95d267738a8b16789538e5126b57b473ab7218e1
   Bech32 Address:   shardeum1...ab7218e1

💰 Transaction Details:
   Messages: 1
   1. /cosmos.staking.v1beta1.MsgDelegate
      Delegator: shardeum1...ab7218e1
      Validator: shardeumvaloper10hduntgw0xcla9wq7qhyxptsqss58mwz78uq87
      Amount:    1000000ashm

⛽ Fee & Gas:
   Gas Limit: 200000
   Fee:       2000 ashm

📝 Memo: Delegated via MetaMask
🌐 Chain ID: shardeum_8117-1
🔢 Sequence: 0

🔐 Signature:
   Full: 0x8d27a7e2...
   R: 8d27a7e2...
   S: 6b58dd0d...
   V: 1 (adjusted from 28)
```

## Workflow

### 1. Sign Transaction with MetaMask

```bash
# Open the demo page
python3 -m http.server 8000
open http://localhost:8000/examples/metamask-eip712-delegate.html
```

1. Connect MetaMask
2. Enter validator address
3. Sign the transaction
4. Save the JSON file (e.g., `signed-tx1.json`)

### 2. Inspect with This Tool

```bash
./build/broadcast-eip712 signed-tx1.json
```

### 3. Broadcasting (Future)

Currently, broadcasting requires additional implementation. See `docs/EIP712_BROADCAST_TODO.md` for details on:
- Public key recovery from EIP-712 signatures
- Cosmos SDK transaction construction with EIP-712 extension
- Account number and sequence querying

## Supported Message Types

Currently supported:
- ✅ `cosmos-sdk/MsgDelegate` - Stake delegation

Easy to add:
- `cosmos-sdk/MsgSend` - Token transfers
- `cosmos-sdk/MsgUndelegate` - Unstake
- `cosmos-sdk/MsgBeginRedelegate` - Redelegate
- `cosmos-sdk/MsgWithdrawDelegatorReward` - Claim rewards
- `cosmos-sdk/MsgVote` - Governance voting

## Technical Details

### Signature Format

The tool parses Ethereum signatures (65 bytes):
- **R**: 32 bytes (first component)
- **S**: 32 bytes (second component)  
- **V**: 1 byte (recovery ID, adjusted from 27/28 to 0/1)

### Message Parsing

The tool deserializes the EIP-712 TypedData and extracts:
- Message type and values
- Fee information (amount + denom)
- Gas limit
- Chain ID and sequence
- Memo

## Related Tools

- **Python script**: `examples/broadcast_eip712_tx.py` - Alternative parser
- **Web demo**: `examples/metamask-eip712-delegate.html` - Sign transactions with MetaMask
- **JS helpers**: `examples/eip712-helpers.js` - Build EIP-712 TypedData

## Development

To add support for more message types, edit the `buildMessages()` function in `cmd/broadcast-eip712/main.go`.

Example for MsgSend:

```go
case "cosmos-sdk/MsgSend":
    var sendValue struct {
        FromAddress string `json:"from_address"`
        ToAddress   string `json:"to_address"`
        Amount      []struct {
            Denom  string `json:"denom"`
            Amount string `json:"amount"`
        } `json:"amount"`
    }
    // ... parse and create MsgSend
```

## Implementation Status

This tool is a **partial implementation** of Option 2 from `docs/EIP712_BROADCAST_TODO.md`.

**Completed:**
- ✅ JSON parsing and validation
- ✅ Message deserialization
- ✅ Signature extraction
- ✅ Display formatting

**TODO:**
- ❌ Public key recovery from EIP-712 signature
- ❌ Cosmos SDK transaction builder with EIP-712 extension
- ❌ Account number and sequence querying
- ❌ Transaction encoding (protobuf)
- ❌ Broadcasting via node RPC

## References

- [EIP-712 Specification](https://eips.ethereum.org/EIPS/eip-712)
- [Cosmos SDK Documentation](https://docs.cosmos.network/)
- [Shardeum EIP-712 Support](../docs/EIP712_SUPPORT.md)
- [Broadcasting Guide](../docs/EIP712_BROADCAST_TODO.md)

## Troubleshooting

### "failed to read file"
- Check file path is correct
- Ensure file exists and is readable

### "failed to parse JSON"
- Verify JSON format from MetaMask export
- Check all required fields are present

### "unsupported message type"
- Only MsgDelegate is currently supported
- Add more message types in `buildMessages()` function

## Future Enhancements

1. **Full broadcasting support**
   - Implement public key recovery
   - Add Cosmos SDK transaction construction
   - Query chain for account info
   
2. **More message types**
   - MsgSend, MsgUndelegate, etc.
   
3. **Batch operations**
   - Process multiple signed transactions
   
4. **Verification mode**
   - Verify signature without broadcasting

## Contributing

To add full broadcasting support:

1. Implement `recoverPubKey()` function using `ethereum/eip712` package
2. Build Cosmos transaction using `testutil/tx/eip712.go` as reference
3. Add account querying from chain
4. Implement protobuf encoding and broadcasting

See existing tests in `tests/integration/eip712/` for examples (note: some tests may need updates for PreciseBankKeeper).
