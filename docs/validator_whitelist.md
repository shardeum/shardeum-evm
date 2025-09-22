# Validator Consensus Whitelisting

## Overview

The validator consensus whitelist controls which validators can participate in consensus by restricting delegation to them. This is implemented as an on-chain governance module that allows network participants to control which validators can receive delegations and join consensus.

## Behavior

### ✅ Allowed Operations

- **Validator Creation**: `MsgCreateValidator` works for whitelisted validators only
- **Delegation to Whitelisted Validators**: `MsgDelegate` to whitelisted validators works normally
- **Redelegation to Whitelisted Validators**: `MsgBeginRedelegate` to whitelisted validators works normally
- **Governance Proposals**: Network participants can vote to add/remove validators from the whitelist

### ❌ Blocked Operations

- **Validator Creation for Non-Whitelisted**: `MsgCreateValidator` for non-whitelisted validators fails with zero gas
- **Delegation to Non-Whitelisted Validators**: `MsgDelegate` to non-whitelisted validators fails with zero gas
- **Redelegation to Non-Whitelisted Validators**: `MsgBeginRedelegate` to non-whitelisted validators fails with zero gas

## Configuration

### On-Chain Governance Module

The validator whitelist is controlled through the `x/validatorwhitelist` module with governance proposals:

#### Genesis Configuration

```json
{
  "app_state": {
    "validatorwhitelist": {
      "params": {
        "enabled": false,
        "allowlist": []
      }
    }
  }
}
```

#### Governance Proposals

**Enable/Disable Whitelist:**

```json
{
  "messages": [
    {
      "@type": "/shardeum.validatorwhitelist.v1.MsgUpdateWhitelist",
      "authority": "shardeum1...",
      "enabled": true,
      "allowlist": ["shardeumvaloper1...", "shardeumvaloper2..."]
    }
  ],
  "title": "Enable validator whitelist",
  "summary": "Enable validator consensus whitelisting",
  "deposit": "10000000ashm"
}
```

**Add Validator:**

```json
{
  "messages": [
    {
      "@type": "/shardeum.validatorwhitelist.v1.MsgAddValidator",
      "authority": "shardeum1...",
      "validator": "shardeumvaloper1..."
    }
  ],
  "title": "Add validator to whitelist",
  "summary": "Allow validator to receive delegations",
  "deposit": "10000000ashm"
}
```

**Remove Validator:**

```json
{
  "messages": [
    {
      "@type": "/shardeum.validatorwhitelist.v1.MsgRemoveValidator",
      "authority": "shardeum1...",
      "validator": "shardeumvaloper1..."
    }
  ],
  "title": "Remove validator from whitelist",
  "summary": "Prevent validator from receiving new delegations",
  "deposit": "10000000ashm"
}
```

### Configuration Options

| Parameter   | Type         | Default | Description                                                |
| ----------- | ------------ | ------- | ---------------------------------------------------------- |
| `enabled`   | boolean      | `false` | Enable/disable validator consensus whitelisting            |
| `allowlist` | string array | `[]`    | List of validator addresses allowed to receive delegations |

## Error Messages

When operations with non-whitelisted validators are attempted, users receive clear error messages:

### Validator Creation Errors

```
Error: validator shardeumvaloper1xyz789 is not whitelisted for consensus (create validator)
```

### Delegation Errors

```
Error: validator shardeumvaloper1xyz789 is not whitelisted for consensus
```

### Redelegation Errors

```
Error: destination validator shardeumvaloper1xyz789 is not whitelisted for consensus
```

## Gas Behavior

### Zero Gas for Failed Transactions

- **Non-whitelisted delegations**: 0 gas consumed (fail-fast with zero cost)
- **Whitelisted delegations**: Normal gas consumption (~50,000+ gas)

### Gas Comparison

| Scenario                                   | Gas Consumed  | Explanation                            |
| ------------------------------------------ | ------------- | -------------------------------------- |
| **Whitelisted Validator Creation**         | ~100,000+ gas | Full validator creation logic          |
| **Non-Whitelisted Validator Creation**     | **0 gas**     | Immediate rejection with zero gas      |
| **Whitelisted Validator Delegation**       | ~50,000+ gas  | Full staking logic + validation        |
| **Non-Whitelisted Validator Delegation**   | **0 gas**     | Immediate rejection with zero gas      |
| **Mixed Transaction with Non-Whitelisted** | **0 gas**     | Entire transaction fails with zero gas |

## Use Cases

### 1. Controlled Validator Onboarding

- Validators must be whitelisted before they can be created
- Network participants vote on which validators can join
- Prevents unauthorized validators from entering the network

### 2. Consensus Control

- Only whitelisted validators can participate in consensus
- Network participants control which validators can receive stake
- Prevents unwanted validators from joining consensus

### 3. Governance-Based Management

- Whitelist changes require governance proposals
- Network participants vote on validator additions/removals
- Transparent and democratic validator management

### 4. Production Network Security

- Prevents malicious validators from being created
- Ensures only trusted validators can participate
- Provides network-level security controls

## Implementation Details

### Interface Pattern

The implementation follows Cosmos SDK best practices using the interface pattern:

```go
type ValidatorWhitelistKeeper interface {
    IsWhitelistEnabled(ctx sdk.Context) bool
    CanValidatorReceiveDelegation(ctx sdk.Context, validatorAddr string) bool
}
```

### Ante Handler Integration

The whitelist is enforced at the ante handler level, providing:

- **Fail-fast behavior**: Immediate rejection before expensive staking logic
- **Zero gas consumption**: No gas wasted on failed transactions
- **Clear error messages**: Specific feedback for users

### Message Types Handled

- `MsgCreateValidator`: Controls validator creation (must be whitelisted)
- `MsgDelegate`: Controls delegation to validators
- `MsgBeginRedelegate`: Controls redelegation to validators

### Module Architecture

The `x/validatorwhitelist` module provides:

#### **Message Types (MsgServer)**

- **`MsgUpdateWhitelist`**: Enable/disable whitelist and set complete allowlist
- **`MsgAddValidator`**: Add a single validator to the allowlist
- **`MsgRemoveValidator`**: Remove a single validator from the allowlist

#### **Query Types (QueryServer)**

- **`QueryParams`**: Get current whitelist parameters (enabled status and allowlist)

#### **Keeper Methods**

- **`GetParams()`**: Retrieve current whitelist parameters
- **`SetParams()`**: Update whitelist parameters
- **`IsValidatorAllowed()`**: Check if a validator is in the allowlist

#### **Events**

- **`update_whitelist`**: Emitted when whitelist is updated
- **`add_validator`**: Emitted when a validator is added to allowlist
- **`remove_validator`**: Emitted when a validator is removed from allowlist

#### **Genesis Support**

- **`InitGenesis`**: Initialize whitelist state from genesis
- **`ExportGenesis`**: Export current whitelist state for genesis

## File Structure & Code Organization

### **Manually Written Files (Business Logic)**

#### **Core Module Files**

- **`x/validatorwhitelist/module.go`** - Main module definition and lifecycle management
- **`x/validatorwhitelist/genesis.go`** - Genesis initialization and export logic
- **`x/validatorwhitelist/keeper/keeper.go`** - Core state management and business logic
- **`x/validatorwhitelist/keeper/msg_server.go`** - Message handling and validation logic
- **`x/validatorwhitelist/keeper/query_server.go`** - Query handling logic

#### **Type Definitions**

- **`x/validatorwhitelist/types/keys.go`** - Store keys and module constants
- **`x/validatorwhitelist/types/events.go`** - Event type definitions
- **`x/validatorwhitelist/types/codec.go`** - Codec registration and interface definitions
- **`x/validatorwhitelist/types/params.go`** - Parameter validation and business rules
- **`x/validatorwhitelist/types/genesis.go`** - Genesis state management

#### **Integration Files**

- **`ante/cosmos/validator_whitelist.go`** - Ante handler decorator (enforcement logic)
- **`ante/interfaces/cosmos.go`** - Interface definitions for modularity
- **`ante/interfaces/validator_whitelist_keeper.go`** - Keeper interface implementations

### **Auto-Generated Files (Protobuf)**

#### **Protobuf Definitions**

- **`proto/shardeum/validatorwhitelist/v1/tx.proto`** - Message type definitions
- **`proto/shardeum/validatorwhitelist/v1/query.proto`** - Query service definitions
- **`proto/shardeum/validatorwhitelist/v1/genesis.proto`** - Genesis state definitions
- **`proto/shardeum/validatorwhitelist/v1/params.proto`** - Parameter definitions

#### **Generated Go Code**

- **`x/validatorwhitelist/types/tx.pb.go`** - Generated message types
- **`x/validatorwhitelist/types/query.pb.go`** - Generated query types
- **`x/validatorwhitelist/types/genesis.pb.go`** - Generated genesis types
- **`x/validatorwhitelist/types/params.pb.go`** - Generated parameter types

### **Business Logic Locations**

#### **1. Core Business Logic**

**File**: `x/validatorwhitelist/keeper/keeper.go`

```go
// Main business logic methods:
- GetParams()           // Retrieve whitelist parameters
- SetParams()           // Update whitelist parameters
- IsValidatorAllowed()  // Check if validator is whitelisted
```

#### **2. Message Processing Logic**

**File**: `x/validatorwhitelist/keeper/msg_server.go`

```go
// Governance message handlers:
- UpdateWhitelist()     // Enable/disable + set allowlist
- AddValidator()        // Add single validator to allowlist
- RemoveValidator()     // Remove single validator from allowlist
```

#### **3. Validation & Business Rules**

**File**: `x/validatorwhitelist/types/params.go`

```go
// Parameter validation:
- Validate()            // Validate whitelist parameters
- IsValidatorAllowed()  // Check if validator is in allowlist
- DefaultParams()       // Default parameter values
```

#### **4. Enforcement Logic**

**File**: `ante/cosmos/validator_whitelist.go`

```go
// Transaction enforcement:
- AnteHandle()          // Block non-whitelisted operations
- Zero gas enforcement  // Fail-fast with zero gas consumption
```

#### **5. Module Lifecycle**

**File**: `x/validatorwhitelist/module.go`

```go
// Module management:
- RegisterServices()    // Register gRPC services
- InitGenesis()         // Initialize from genesis
- ExportGenesis()       // Export for genesis
```

### **Code Generation Process**

#### **Protobuf Generation**

```bash
# Generate Go code from .proto files
make proto-gen

# This creates:
# - x/validatorwhitelist/types/*.pb.go files
# - Message types, query types, genesis types
```

#### **What Gets Generated**

- **Message structs** from `tx.proto`
- **Query structs** from `query.proto`
- **Genesis structs** from `genesis.proto`
- **Parameter structs** from `params.proto`
- **gRPC service interfaces** and method signatures

#### **What's Manually Written**

- **Business logic implementation** in keeper methods
- **Validation rules** in params.go
- **Event emission** in msg_server.go
- **Module wiring** in module.go
- **Ante handler enforcement** in validator_whitelist.go

### **Key Design Patterns**

#### **1. Interface-Based Design**

- **`ante/interfaces/cosmos.go`** defines the `ValidatorWhitelistKeeper` interface
- **`ante/interfaces/validator_whitelist_keeper.go`** provides implementations
- Enables modularity and testability

#### **2. Keeper Pattern**

- **`x/validatorwhitelist/keeper/keeper.go`** manages all state operations
- Encapsulates business logic and state access
- Provides clean API for other modules

#### **3. Event-Driven Architecture**

- **`x/validatorwhitelist/types/events.go`** defines event types
- **`x/validatorwhitelist/keeper/msg_server.go`** emits events on state changes
- Enables monitoring and integration

#### **4. Genesis Integration**

- **`x/validatorwhitelist/genesis.go`** handles initialization
- **`x/validatorwhitelist/types/genesis.go`** manages genesis state
- Supports network bootstrap and state export

## Testing

### Unit Tests

Comprehensive test coverage includes:

#### **Types Tests (`x/validatorwhitelist/types/`)**

- **`params_test.go`**: Parameter validation and business logic

  - Default parameter values
  - Parameter validation rules
  - Validator allowlist checking logic
  - Case-insensitive validator address matching
  - Disabled whitelist behavior (all validators allowed)

- **`genesis_test.go`**: Genesis state management
  - Default genesis state creation
  - Custom genesis state creation
  - Genesis state validation

#### **Keeper Tests (`x/validatorwhitelist/keeper/`)**

- **`keeper_test.go`**: Core keeper functionality and interface compliance

  - Keeper constructor validation
  - Interface implementation verification
  - Method signature validation

- **`msg_server_test.go`**: Message server functionality

  - `UpdateWhitelist` message handling
  - `AddValidator` message handling
  - `RemoveValidator` message handling
  - Interface compliance verification

- **`query_server_test.go`**: Query server functionality

  - `Params` query handling
  - Interface compliance verification

- **`integration_test.go`**: Integration testing framework
  - Keeper functionality integration
  - State management testing
  - End-to-end workflow validation

#### **Interface Tests (`ante/interfaces/`)**

- **`validator_whitelist_keeper_test.go`**: Interface and decorator testing
  - Config-based keeper implementation
  - Mock keeper for unit testing
  - Delegation control logic
  - Case sensitivity handling
  - Empty allowlist scenarios

### Test Coverage

The validator whitelist module has comprehensive test coverage for:

- ✅ **Parameter validation** - All parameter validation rules
- ✅ **Business logic** - Validator allowlist checking
- ✅ **Genesis management** - State initialization and export
- ✅ **Keeper functionality** - Core keeper methods and state management
- ✅ **Message server** - All governance message handling
- ✅ **Query server** - All query functionality
- ✅ **Interface implementations** - All keeper implementations
- ✅ **Integration testing** - End-to-end workflow validation
- ✅ **Edge cases** - Empty allowlists, case sensitivity, disabled state
- ✅ **Error handling** - Invalid parameters and edge cases

### Running Tests

```bash
# Run all validator whitelist tests
go test -v ./x/validatorwhitelist/...

# Run specific test suites
go test -v ./x/validatorwhitelist/types/...
go test -v ./x/validatorwhitelist/keeper/...
go test -v ./ante/interfaces/

# Run with coverage
go test -v -cover ./x/validatorwhitelist/...
```

## Security Considerations

### Fail-Fast Security

- Immediate rejection prevents partial state changes
- Zero gas consumption prevents gas-based attacks
- Clear error messages help users understand restrictions

### Configuration Security

- Whitelist is managed through on-chain governance
- Changes require governance proposals and voting
- Transparent and auditable whitelist management

## Future Enhancements

### Advanced Features

- Time-based whitelist expiration
- Validator reputation-based whitelisting
- Emergency validator removal procedures
- Batch validator operations

### Integration Features

- Integration with validator monitoring systems
- Automated whitelist management based on performance metrics
- Cross-chain validator whitelist synchronization

## Troubleshooting

### Common Issues

1. **"validator not whitelisted for consensus"**

   - Check if validator address is in the whitelist: `shardeumd q validatorwhitelist params`
   - Verify whitelist is enabled: `shardeumd q validatorwhitelist params`
   - Ensure validator address format is correct (`shardeumvaloper1...`)

2. **All delegations failing**

   - Check if whitelist is disabled: `shardeumd q validatorwhitelist params`
   - Verify allowlist is not empty when enabled
   - Check if validator is in the allowlist

3. **Validator creation failing**

   - Validator must be whitelisted before creation
   - Submit governance proposal to add validator to whitelist first
   - Get valoper address from key: `shardeumd keys show validator --bech val -a`

4. **Governance proposal failing**
   - Ensure sufficient deposit (10M ashm minimum)
   - Check voting power of proposer
   - Verify proposal format and authority address

### Configuration Validation

- Validator addresses should be in `shardeumvaloper1...` format
- Addresses are case-sensitive
- Use `shardeumd keys show validator --bech val -a` to get correct format

## Examples

### Query Whitelist Status

```bash
# Check current whitelist parameters
shardeumd q validatorwhitelist params --node tcp://localhost:26657 -o json

# Check if specific validator is whitelisted
shardeumd q validatorwhitelist params --node tcp://localhost:26657 -o json | jq '.params.allowlist'
```

### Enable Whitelist via Governance

```bash
# Create proposal to enable whitelist
cat > /tmp/enable_whitelist.json << EOF
{
  "messages": [
    {
      "@type": "/shardeum.validatorwhitelist.v1.MsgUpdateWhitelist",
      "authority": "$GOV_AUTH",
      "enabled": true,
      "allowlist": ["shardeumvaloper1...", "shardeumvaloper2..."]
    }
  ],
  "title": "Enable validator whitelist",
  "summary": "Enable validator consensus whitelisting",
  "deposit": "10000000ashm"
}
EOF

# Submit proposal
shardeumd tx gov submit-proposal /tmp/enable_whitelist.json \
  --from proposer --yes --broadcast-mode sync
```

### Add Validator to Whitelist

```bash
# Get validator address
VALOPER=$(shardeumd keys show validator --bech val -a)

# Create proposal to add validator
cat > /tmp/add_validator.json << EOF
{
  "messages": [
    {
      "@type": "/shardeum.validatorwhitelist.v1.MsgAddValidator",
      "authority": "$GOV_AUTH",
      "validator": "$VALOPER"
    }
  ],
  "title": "Add validator to whitelist",
  "summary": "Allow validator to receive delegations",
  "deposit": "10000000ashm"
}
EOF

# Submit proposal
shardeumd tx gov submit-proposal /tmp/add_validator.json \
  --from proposer --yes --broadcast-mode sync
```

### Complete Workflow

1. **Get validator valoper address**: `shardeumd keys show validator --bech val -a`
2. **Submit governance proposal** to add validator to whitelist
3. **Vote on proposal** and wait for it to pass
4. **Create validator** (now that it's whitelisted)
5. **Delegate to validator** (now that it's whitelisted)

## Available Operations

### Query Operations

```bash
# Get current whitelist parameters
shardeumd q validatorwhitelist params --node tcp://localhost:26657 -o json

# Check if whitelist is enabled
shardeumd q validatorwhitelist params --node tcp://localhost:26657 -o json | jq '.params.enabled'

# Get current allowlist
shardeumd q validatorwhitelist params --node tcp://localhost:26657 -o json | jq '.params.allowlist'

# Check if specific validator is whitelisted
shardeumd q validatorwhitelist params --node tcp://localhost:26657 -o json | jq '.params.allowlist[] | select(. == "shardeumvaloper1...")'
```

### Governance Operations

#### 1. Enable/Disable Whitelist

```bash
# Enable whitelist with initial allowlist
shardeumd tx gov submit-proposal /tmp/enable_whitelist.json --from proposer --yes

# Disable whitelist
shardeumd tx gov submit-proposal /tmp/disable_whitelist.json --from proposer --yes
```

#### 2. Add Validator to Whitelist

```bash
# Add single validator
shardeumd tx gov submit-proposal /tmp/add_validator.json --from proposer --yes
```

#### 3. Remove Validator from Whitelist

```bash
# Remove single validator
shardeumd tx gov submit-proposal /tmp/remove_validator.json --from proposer --yes
```

#### 4. Vote on Proposals

```bash
# Vote yes on proposal
shardeumd tx gov vote $PROPOSAL_ID yes --from validator --yes

# Vote no on proposal
shardeumd tx gov vote $PROPOSAL_ID no --from validator --yes
```

### Validator Operations

#### 1. Get Validator Address

```bash
# Get valoper address from key
shardeumd keys show validator --bech val -a

# Get valoper address from existing validator
shardeumd q staking validators --node tcp://localhost:26657 -o json | jq '.validators[] | select(.description.moniker=="node1") | .operator_address'
```

#### 2. Create Validator (Must be Whitelisted)

```bash
# Create validator using JSON file
shardeumd tx staking create-validator /tmp/validator.json --from validator --yes
```

#### 3. Delegate to Validator (Must be Whitelisted)

```bash
# Delegate to whitelisted validator
shardeumd tx staking delegate $VALOPER 10000ashm --from delegator --yes
```

### Event Monitoring

```bash
# Monitor whitelist events
shardeumd q txs --events "update_whitelist" --node tcp://localhost:26657 -o json

# Monitor validator addition events
shardeumd q txs --events "add_validator" --node tcp://localhost:26657 -o json

# Monitor validator removal events
shardeumd q txs --events "remove_validator" --node tcp://localhost:26657 -o json
```
