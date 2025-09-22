### Validator Whitelist End-to-End Runbook

This guide walks through bootstrapping a 1-node network, enabling the validator whitelist via governance, verifying fail-fast behavior, adding a new validator to the allowlist, and confirming two validators in consensus.

## Quick Reference

**Prereqs:**

- You are on branch `main-validatior-whitelists` and `make build` works
- jq installed (mac: `brew install jq`)

**Environment notes:**

- Repo root: replace $REPO with your absolute path if not running from repo
- Testnet home dirs are under `.testnet/nodeN`
- **Balanced stake**: This runbook uses 10K ashm for all delegations to ensure both validators have competitive stake amounts

---

## 1) Start a 1-node network

From repo root:

```bash
./scripts/start_network.sh 1
```

Watch logs (optional):

```bash
tail -f .testnet/node0/node.log | sed -n 's/.*NewBlock.*/&/p'
```

Confirm blocks:

```bash
curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height'
```

Get node0 valoper address (used later):

```bash
VALOPER0=$(./build/shardeumd keys show validator --bech val --home .testnet/node0 -a)
echo $VALOPER0
```

**CRITICAL**: The validator needs voting power to participate in governance. Delegate to the validator first:

```bash
./build/shardeumd tx staking delegate $VALOPER0 10000ashm \
  --home .testnet/node0 --keyring-backend test --from dev0 \
  --yes --broadcast-mode sync \
  --gas auto --gas-adjustment 1.3 --gas-prices 0.000006ashm
```

Note: The validator whitelist is disabled at genesis per config (`enabled=false`), so node0 produces blocks.

Check current whitelist status:

```bash
./build/shardeumd q validatorwhitelist params --node tcp://localhost:26657 -o json
```

---

## 2) Add a second node (full node)

Start a second node that connects to node0 (seed):

```bash
./scripts/add_node.sh node1 http://localhost:26657
```

Optionally, create a key on node1 for later validator creation:

```bash
./build/shardeumd keys add validator --home .testnet/node1 --keyring-backend test
ADDR1=$(./build/shardeumd keys show validator --home .testnet/node1 --keyring-backend test -a)
echo $ADDR1
```

Fund node1 address from node0 (so node1 can self-delegate and compete with node0):

```bash
./build/shardeumd tx bank send dev0 $ADDR1 30000ashm \
  --home .testnet/node0 --keyring-backend test --yes --broadcast-mode sync \
  --gas auto --gas-adjustment 1.3 --gas-prices 0.000006ashm
```

---

## 3) Enable whitelist via governance (empty allowlist)

Fetch gov authority and create proposal JSON (Gov v1 messages):

```bash
GOV_AUTH=$(./build/shardeumd q auth module-accounts -o json \
  | jq -r '.accounts[] | select(.value.name=="gov") | .value.address')
cat > /tmp/enable_whitelist.json << EOF
{
  "messages": [
    {
      "@type": "/shardeum.validatorwhitelist.v1.MsgUpdateWhitelist",
      "authority": "$GOV_AUTH",
      "enabled": true,
      "allowlist": []
    }
  ],
  "metadata": "",
  "title": "Enable whitelist",
  "summary": "Enable consensus validator allowlist with empty set",
  "deposit": "10000000ashm"
}
EOF
```

Submit proposal from node0:

```bash
./build/shardeumd tx gov submit-proposal /tmp/enable_whitelist.json \
  --home .testnet/node0 --keyring-backend test --from dev0 \
  --yes --broadcast-mode sync \
  --gas auto --gas-adjustment 1.3 --gas-prices 0.000006ashm
```

Get proposal ID:

```bash
./build/shardeumd q gov proposals --node tcp://localhost:26657 -o json | jq -r '.proposals[-1].id // empty'
```

Vote YES:

```bash
PROPOSAL_ID=$(./build/shardeumd q gov proposals --node tcp://localhost:26657 -o json | jq -r '.proposals[-1].id // empty')
./build/shardeumd tx gov vote $PROPOSAL_ID yes \
  --home .testnet/node0 --keyring-backend test --from validator \
  --yes --broadcast-mode sync \
  --gas auto --gas-adjustment 1.3 --gas-prices 0.000006ashm
```

Wait until the voting period ends (testnet has 2-minute voting period), then verify proposal status:

```bash
./build/shardeumd q gov proposal $PROPOSAL_ID --node tcp://localhost:26657 -o json | jq '.proposal.status'
```

Verify whitelist is now enabled:

```bash
./build/shardeumd q validatorwhitelist params --node tcp://localhost:26657 -o json
```

**Note**: The testnet has a 2-minute voting period. Wait ~2.5 minutes before checking the status.

At this point, whitelist is enabled and the allowlist is empty.

---

## 4) Verify fail-fast delegation block (allowlist empty)

Attempt to delegate to the existing validator (node0). This should fail with unauthorized error and consume zero gas:

```bash
./build/shardeumd tx staking delegate $VALOPER0 1000000ashm \
  --home .testnet/node0 --keyring-backend test --from dev0 \
  --yes --broadcast-mode sync \
  --gas auto --gas-adjustment 1.3 --gas-prices 0.000006ashm || true
```

Expect an error like:
"validator <valoper> is not whitelisted for consensus"

---

## 5) Add node0 and node1 to allowlist via governance

First add node0 (so further delegations to node0 are allowed):

```bash
cat > /tmp/add_node0.json << EOF
{
  "messages": [
    {
      "@type": "/shardeum.validatorwhitelist.v1.MsgAddValidator",
      "authority": "${GOV_AUTH}",
      "validator": "${VALOPER0}"
    }
  ],
  "metadata": "",
  "title": "Add node0 to allowlist",
  "summary": "Permit delegations to node0 (bootstrap validator)",
  "deposit": "10000000ashm"
}
EOF

./build/shardeumd tx gov submit-proposal /tmp/add_node0.json \
  --home .testnet/node0 --keyring-backend test --from dev0 \
  --yes --broadcast-mode sync \
  --gas auto --gas-adjustment 1.3 --gas-prices 0.000006ashm

PID0=$(./build/shardeumd q gov proposals --node tcp://localhost:26657 -o json | jq -r '.proposals[-1].id // empty')
./build/shardeumd tx gov vote $PID0 yes --home .testnet/node0 --keyring-backend test --from validator --yes --broadcast-mode sync \
  --gas auto --gas-adjustment 1.3 --gas-prices 0.000006ashm

# Wait for voting period to complete (~2.5 minutes)
echo "Waiting for voting period to complete..."
sleep 150

# Check proposal status
./build/shardeumd q gov proposal $PID0 --node tcp://localhost:26657 -o json | jq '.proposal.status'
```

Now create node1 validator. First, we need to whitelist node1's valoper address:

```bash
# Step 1: Get node1 valoper address from its key (before creating validator)
VALOPER1=$(./build/shardeumd keys show validator --bech val --home .testnet/node1 -a)
echo "Node1 valoper: $VALOPER1"

# Step 2: Add node1 to whitelist (before creating validator)
cat > /tmp/add_node1.json << EOF
{
  "messages": [
    {
      "@type": "/shardeum.validatorwhitelist.v1.MsgAddValidator",
      "authority": "${GOV_AUTH}",
      "validator": "${VALOPER1}"
    }
  ],
  "metadata": "",
  "title": "Add node1 to allowlist",
  "summary": "Allow node1 to create validator and receive delegations",
  "deposit": "10000000ashm"
}
EOF

# Submit the proposal
./build/shardeumd tx gov submit-proposal /tmp/add_node1.json \
  --home .testnet/node0 --keyring-backend test --from dev0 \
  --yes --broadcast-mode sync \
  --gas auto --gas-adjustment 1.3 --gas-prices 0.000006ashm

# Get the proposal ID and vote
PID1=$(./build/shardeumd q gov proposals --node tcp://localhost:26657 -o json | jq -r '.proposals[-1].id // empty')
./build/shardeumd tx gov vote $PID1 yes \
  --home .testnet/node0 --keyring-backend test --from validator \
  --yes --broadcast-mode sync \
  --gas auto --gas-adjustment 1.3 --gas-prices 0.000006ashm

# Wait for voting period to complete (~2.5 minutes)
echo "Waiting for voting period to complete..."
sleep 150

# Check proposal status
./build/shardeumd q gov proposal $PID1 --node tcp://localhost:26657 -o json | jq '.proposal.status'

# Step 3: Create node1 validator (now that it's whitelisted)
MONIKER1=node1
VALCONS_PUBKEY1=$(./build/shardeumd cometbft show-validator --home .testnet/node1)

# Create validator JSON file
cat > /tmp/validator1.json << EOF
{
	"pubkey": $VALCONS_PUBKEY1,
	"amount": "20000ashm",
	"moniker": "$MONIKER1",
	"identity": "",
	"website": "",
	"security": "",
	"details": "",
	"commission-rate": "0.1",
	"commission-max-rate": "0.2",
	"commission-max-change-rate": "0.01",
	"min-self-delegation": "1"
}
EOF

# Create validator using JSON file
./build/shardeumd tx staking create-validator /tmp/validator1.json \
  --from validator \
  --chain-id shardeum-testnet \
  --home .testnet/node1 --keyring-backend test \
  --yes --broadcast-mode sync \
  --gas auto --gas-adjustment 1.3 --gas-prices 0.000006ashm

# Step 4: Verify node1 is now whitelisted
./build/shardeumd q validatorwhitelist params --node tcp://localhost:26657 -o json | jq '.params.allowlist'
```

---

## 6) Delegate again to node0 (now allowed)

First, verify node0 is in the whitelist:

```bash
./build/shardeumd q validatorwhitelist params --node tcp://localhost:26657 -o json
```

Now delegate to node0 (should succeed):

```bash
./build/shardeumd tx staking delegate $VALOPER0 10000ashm \
  --home .testnet/node0 --keyring-backend test --from dev0 \
  --yes --broadcast-mode sync \
  --gas auto --gas-adjustment 1.3 --gas-prices 0.000006ashm
```

Check delegation:

```bash
./build/shardeumd q staking delegations $(./build/shardeumd keys show dev0 --home .testnet/node0 --keyring-backend test -a) \
  --node tcp://localhost:26657 -o json | jq
```

---

## 7) Test delegation to node1 (now whitelisted)

Now that node1 is both a validator and whitelisted, delegation should work:

```bash
# Delegate to node1 (should succeed now)
./build/shardeumd tx staking delegate $VALOPER1 10000ashm \
  --home .testnet/node0 --keyring-backend test --from dev0 \
  --yes --broadcast-mode sync \
  --gas auto --gas-adjustment 1.3 --gas-prices 0.000006ashm

# Check delegations
./build/shardeumd q staking delegations $(./build/shardeumd keys show dev0 --home .testnet/node0 --keyring-backend test -a) \
  --node tcp://localhost:26657 -o json | jq
```

**Expected result**: Delegation should succeed.

---

## 8) Test removing validator from whitelist

Let's test removing node1 from the whitelist and verify that delegation fails:

Create proposal to remove node1 from whitelist:

```bash
cat > /tmp/remove_node1.json << EOF
{
  "messages": [
    {
      "@type": "/shardeum.validatorwhitelist.v1.MsgRemoveValidator",
      "authority": "${GOV_AUTH}",
      "validator": "${VALOPER1}"
    }
  ],
  "metadata": "",
  "title": "Remove node1 from allowlist",
  "summary": "Remove node1 from validator whitelist to test delegation blocking",
  "deposit": "10000000ashm"
}
EOF

./build/shardeumd tx gov submit-proposal /tmp/remove_node1.json \
  --home .testnet/node0 --keyring-backend test --from dev0 \
  --yes --broadcast-mode sync \
  --gas auto --gas-adjustment 1.3 --gas-prices 0.000006ashm
```

Vote on the proposal:

```bash
PROPOSAL_ID=$(./build/shardeumd q gov proposals --node tcp://localhost:26657 -o json | jq -r '.proposals[-1].id // empty')
./build/shardeumd tx gov vote $PROPOSAL_ID yes \
  --home .testnet/node0 --keyring-backend test --from validator \
  --yes --broadcast-mode sync \
  --gas auto --gas-adjustment 1.3 --gas-prices 0.000006ashm
```

Wait for voting period and verify whitelist status:

```bash
# Wait ~2.5 minutes for voting period
./build/shardeumd q validatorwhitelist params --node tcp://localhost:26657 -o json
```

Test delegation to removed validator (should fail):

```bash
./build/shardeumd tx staking delegate $VALOPER1 1000ashm \
  --home .testnet/node0 --keyring-backend test --from dev0 \
  --yes --broadcast-mode sync \
  --gas auto --gas-adjustment 1.3 --gas-prices 0.000006ashm
```

**Expected result**: Delegation should fail with "validator not whitelisted for consensus" error.

**Note**: Removing a validator from the whitelist only prevents new delegations. The validator itself remains in the validator set and can still produce blocks if it has enough stake. To completely remove a validator, you would need to slash it via governance.

---

## 8.1) Understanding Validator Whitelist Behavior

**Important**: The validator whitelist controls **delegation access**, not validator existence:

- ✅ **Prevents new delegations** to non-whitelisted validators
- ✅ **Blocks validator creation** for non-whitelisted addresses
- ❌ **Does NOT remove existing validators** from the active set
- ❌ **Does NOT prevent existing validators** from producing blocks

**This is the correct behavior** for a validator whitelist system. The goal is to control who can receive new stake, not to remove existing validators.

**If you need to remove a validator completely**, you would need to:

1. **Slash the validator** (via governance or slashing conditions)
2. **Wait for unbonding period** to complete
3. **Or reduce its stake** below the minimum threshold

**For testing purposes**, the whitelist behavior you observed is working correctly - it prevents new delegations while allowing existing validators to continue operating.

### **Important: Slashing via Governance is NOT Available**

**❌ `MsgSlashValidator` does not exist in Cosmos SDK v0.53.4**

The Cosmos SDK does not natively support slashing validators through governance proposals. The only slashing-related message available is `MsgUnjail`, which is for unjailing a validator, not slashing it.

### **Alternative Approaches to Remove a Validator:**

#### **Option 1: Wait for Natural Slashing (Recommended)**

Validators can be slashed automatically for:

- **Double signing** (5% slash)
- **Downtime** (1% slash)
- **Missing blocks** (jailed, then can be unjailed)

#### **Option 2: Reduce Validator Stake Below Minimum**

```bash
# Check current validator stake
./build/shardeumd q staking validator $VALOPER1 --node tcp://localhost:26657 -o json | jq '.validator.tokens'

# If the validator has delegations, you could redelegate away from it
# This would reduce its stake and potentially remove it from the active set
```

**Note**: This approach is only feasible in test environments where you control the validator. In production, you won't have control over the validator's stake.

#### **Option 3: Custom Implementation (Advanced)**

You would need to implement a custom governance proposal type in your chain's codebase to enable slashing via governance.

### **Production Considerations:**

#### **Real-World Scenarios:**

- **You don't control the validator** - Can't reduce its stake
- **Validator continues operating** - Still produces blocks and earns rewards
- **Delegators are protected** - New delegations are blocked (main goal achieved)
- **Existing delegators unaffected** - Their stake remains safe

#### **What the Whitelist Actually Achieves:**

1. **Prevents new stake** from flowing to non-whitelisted validators
2. **Controls validator growth** - Non-whitelisted validators can't grow their stake
3. **Protects delegators** - Prevents accidental delegation to unwanted validators
4. **Maintains network stability** - Existing validators continue operating normally

#### **Long-term Strategy:**

- **Gradual transition** - Over time, whitelisted validators will grow while non-whitelisted ones stagnate
- **Natural attrition** - Non-whitelisted validators may eventually become less competitive
- **Community pressure** - Validators may choose to comply with whitelist requirements

### **Current Whitelist Behavior is Correct:**

- ✅ **Prevents new delegations** to non-whitelisted validators
- ✅ **Blocks validator creation** for non-whitelisted addresses
- ❌ **Does NOT remove existing validators** (this is by design)

**The whitelist is working as intended** - it controls who can receive new stake, not who can remain as a validator.

---

## 9) Verify two validators in consensus

Check active validators:

```bash
./build/shardeumd q staking validators --node tcp://localhost:26657 -o json | jq -r '.validators[] | "\(.description.moniker) | \(.operator_address) | status=\(.status)"'
```

---

## Troubleshooting

### Governance Proposals Failing

**Problem**: Proposals are rejected with "not enough votes" even after voting.

**Solution**:

1. Ensure the validator has voting power by delegating first (Step 1)
2. Use `--from validator` (not `--from dev0`) when voting
3. Wait for the full voting period (2 minutes) before checking status

### Vote Count Shows 0

**Problem**: Vote count shows 0 even after successful vote submission.

**Solution**: This is normal during the voting period. The `final_tally_result` is only updated after the voting period ends.

### Validator Has No Voting Power

**Problem**: Validator shows `voting_power: null`.

**Solution**: The validator needs delegations to have voting power. Delegate tokens to the validator first.

### JSON Parse Errors

**Problem**: `jq` commands fail with parse errors.

**Solution**: Always use `-o json` flag with `shardeumd q` commands to ensure JSON output.

Check CometBFT validator set height and hashes (should include 2 entries as powers grow):

```bash
curl -s http://localhost:26657/validators | jq '.result.validators | length'
```

You should see 2 validators after the next end-block updates once node1 bonds enough stake.

---

## Additional Troubleshooting

### Critical Issues and Fixes

**1. "unknown field 'validator_address'" error:**

- **Issue**: Proposal JSON uses wrong field name
- **Fix**: Use `"validator"` not `"validator_address"` in proposal JSON
- **Example**:
  ```json
  {
    "@type": "/shardeum.validatorwhitelist.v1.MsgAddValidator",
    "authority": "shardeum10d07y265gmmuvt4z0w9aw880jnsr700jzj92zh",
    "validator": "shardeumvaloper1..." // ✅ Correct
    // "validator_address": "..."      // ❌ Wrong
  }
  ```

**2. "unknown flag: --amount" error for create-validator:**

- **Issue**: Using command-line flags instead of JSON file
- **Fix**: Create a JSON file with validator details
- **Example**:

  ```bash
  # Create validator JSON file
  cat > /tmp/validator1.json << EOF
  {
    "pubkey": {"@type":"/cosmos.crypto.ed25519.PubKey","key":"..."},
    "amount": "20000ashm",
    "moniker": "node1",
    "commission-rate": "0.1",
    "commission-max-rate": "0.2",
    "commission-max-change-rate": "0.01",
    "min-self-delegation": "1"
  }
  EOF

  # Use JSON file
  ./build/shardeumd tx staking create-validator /tmp/validator1.json --from validator
  ```

**3. Delegation succeeds when it should fail:**

- **Issue**: ValidatorWhitelistDecorator not wired into ante handler
- **Fix**: Ensure the decorator is added to `newCosmosAnteHandler` in `shardeumd/ante/cosmos_handler.go`
- **Check**: Look for this line in the ante handler chain:
  ```go
  cosmosante.NewValidatorWhitelistDecorator(options.ValidatorWhitelistKeeper),
  ```

**4. "validator does not exist" error:**

- **Issue**: Network was restarted, validator address changed
- **Fix**: Get current validator address:
  ```bash
  VALOPER0=$(./build/shardeumd keys show validator --bech val --home .testnet/node0 -a)
  echo $VALOPER0
  ```

**5. "service does not have cosmos.msg.v1.service proto annotation" warning:**

- **Issue**: Missing service annotation in protobuf
- **Fix**: Add to `tx.proto`:
  ```protobuf
  service Msg {
    option (cosmos.msg.v1.service) = true;
    // ... rest of service definition
  }
  ```

**6. "unable to resolve type URL" error:**

- **Issue**: Protobuf types not registered
- **Fix**: Ensure `RegisterInterfaces` is called in `x/validatorwhitelist/types/codec.go`
- **Check**: Look for registration of `MsgUpdateWhitelist`, `MsgAddValidator`, `MsgRemoveValidator`

**7. "parse error: Invalid numeric literal" when querying validators:**

- **Issue**: Missing `-o json` flag causes jq to parse non-JSON output
- **Fix**: Always add `-o json` to `shardeumd q` commands
- **Example**:

  ```bash
  # ❌ Wrong - causes parse error
  VALOPER1=$(./build/shardeumd q staking validators --node tcp://localhost:26657 | jq -r '.validators[] | select(.description.moniker=="node1") | .operator_address')

  # ✅ Correct - works properly
  VALOPER1=$(./build/shardeumd q staking validators --node tcp://localhost:26657 -o json | jq -r '.validators[] | select(.description.moniker=="node1") | .operator_address')
  ```

### Expected Behavior

**When whitelist is disabled:**

- All delegations should succeed
- All validator creation should succeed

**When whitelist is enabled but empty:**

- All delegations should fail with "validator not whitelisted for consensus"
- All validator creation should fail with same error
- Gas used should be 0 (fail-fast behavior)

**When whitelist is enabled with validators:**

- Only whitelisted validators can receive delegations
- Only whitelisted validators can be created
- Non-whitelisted validators fail with clear error message

---

Notes:

- With whitelist enabled and empty allowlist, all new delegations are rejected fail-fast with zero gas; existing validator keeps producing blocks.
- Creating a new validator (MsgCreateValidator) is blocked if the destination valoper is not allowlisted.
- For quicker testing, you can shorten gov voting period in genesis before starting.
