# Commission & Rewards Withdrawal Guide

End-to-end reference for inspecting and withdrawing validator commission and delegator rewards on the local Shardeum (Cosmos-SDK fork) network. Includes verification and troubleshooting—tailored to the local test setup where each node has its own home under `.local/nodeN` and uses keyring backend `test`.

---
## 1. Terminology Recap
| Term | Meaning |
|------|---------|
| Account / Delegator Address | Bech32 prefix `shardeum` (e.g. `shardeum1...`). Holds liquid balance, pays fees, is the delegator (even for its own validator). |
| Validator Operator Address | Bech32 prefix `shardeumvaloper` (e.g. `shardeumvaloper1...`). Used in staking & distribution (commission, delegations). |
| Commission | Validator’s skimmed portion of block/distribution rewards, accumulated separately until withdrawn. |
| Delegator Rewards | The remaining rewards owed to each delegator (including the validator’s self‑delegation). |
| Self-Delegation | The validator operator’s own delegation (usually 1 token in local setups). |

Commission and delegator rewards live off-balance (distribution module state) until claimed. Claiming transfers coins into the delegator account (the account behind `--from`).

---
## 2. Quick Identification Commands
Set helper variables (node index examples: node0 = genesis validator, node1 = promoted validator):
```bash
# For node1 example
ACC_NODE1=$(shardeumd keys show validator-node1 --home .local/node1 --keyring-backend test -a)
VALOPER_NODE1=$(shardeumd keys show validator-node1 --home .local/node1 --keyring-backend test --bech val -a)
echo "Account : $ACC_NODE1"
echo "Valoper : $VALOPER_NODE1"
```

Check validator object:
```bash
shardeumd query staking validator "$VALOPER_NODE1" --home .local/node0 -o json | jq '.validator | {operator_address,status,tokens,delegator_shares,commission: .commission.commission_rates.rate}'
```

Check self-delegation (shares & balance):
```bash
shardeumd query staking delegation "$ACC_NODE1" "$VALOPER_NODE1" --home .local/node0 -o json | jq
```

---
## 3. Inspecting Commission & Rewards
Commission (unwithdrawn):
```bash
shardeumd query distribution commission "$VALOPER_NODE1" --home .local/node0 -o json | jq
```
Delegator rewards (self):
```bash
shardeumd query distribution rewards "$ACC_NODE1" "$VALOPER_NODE1" --home .local/node0 -o json | jq
```
Liquid bank balance:
```bash
shardeumd query bank balances "$ACC_NODE1" --home .local/node0 -o json | jq
```

Using REST (optional):
```bash
curl -s http://localhost:1317/cosmos/distribution/v1beta1/validators/$VALOPER_NODE1/commission | jq
curl -s "http://localhost:1317/cosmos/bank/v1beta1/balances/$ACC_NODE1" | jq
```

---
## 4. Withdrawing Commission
### Important (Fork-Specific Behavior)
This fork’s CLI expects the *key name* (not the valoper address) as the positional argument to `withdraw-validator-commission` (usage shows `[validator-addr]` but internally it looks up a key). For node1 the key name is `validator-node1`.

Standard pattern (for node1):
```bash
shardeumd tx distribution withdraw-validator-commission validator-node1 \
  --from validator-node1 \
  --home .local/node1 \
  --keyring-backend test \
  --chain-id shardeum_8119-1 \
  --gas auto --gas-adjustment 1.3 \
  --fees 699684826939954723453ashm \
  -y -o json | jq
```

Genesis validator (node0) key name is simply `validator`:
```bash
shardeumd tx distribution withdraw-validator-commission validator \
  --from validator \
  --home .local/node0 \
  --keyring-backend test \
  --chain-id shardeum_8119-1 \
  --gas auto --gas-adjustment 1.3 \
  --fees 699684826939954723453ashm -y
```

If you accidentally pass the valoper address instead of the key name and see:
```
failed to convert address field to address: <valoper>.info: key not found
```
retry with the key name positional.

---
## 5. Withdrawing Delegator Rewards (Self‑Delegation)
Depending on CLI wiring, this usually *does* expect the valoper address (confirm via `--help`). Try valoper first:
```bash
shardeumd tx distribution withdraw-rewards "$VALOPER_NODE1" \
  --from validator-node1 \
  --home .local/node1 \
  --keyring-backend test \
  --chain-id shardeum_8119-1 \
  --gas auto --gas-adjustment 1.3 \
  --fees 699684826939954723453ashm -y
```
If it errors similarly about key not found, your binary may also expect the *key name* positional:
```bash
shardeumd tx distribution withdraw-rewards validator-node1 \
  --from validator-node1 ... (same flags)
```

After rewards withdrawal, re-query:
```bash
shardeumd query distribution rewards "$ACC_NODE1" "$VALOPER_NODE1" --home .local/node0 -o json | jq
```

---
## 6. Post-Withdrawal Verification Checklist
1. Commission query shows reduced or zero amount.
2. Delegator rewards query shows reduced or zero (if you withdrew rewards).
3. Bank balance of `ACC_NODE1` increases by (commission + rewards − fees).
4. Staking shares (`delegator_shares`) remain unchanged (withdrawals do not affect stake).

Handy combined check:
```bash
VALOPER=$VALOPER_NODE1
ACC=$ACC_NODE1
echo "--- Commission" && shardeumd query distribution commission "$VALOPER" --home .local/node0 -o json | jq '.commission'
echo "--- Rewards" && shardeumd query distribution rewards "$ACC" "$VALOPER" --home .local/node0 -o json | jq '.rewards'
echo "--- Balance" && shardeumd query bank balances "$ACC" --home .local/node0 -o json | jq '.balances'
```

---
## 7. Estimating Expected Commission
Per block commission ≈ `TotalBlockRewards * (ValidatorProportion) * CommissionRate`.
Where ValidatorProportion ≈ `self_vote_power / total_power` (if only two validators, roughly each share). Over many blocks, inflation + fees accumulate; large commission numbers in test networks often reflect artificially high inflation settings.

---
## 8. Common Errors & Fixes
| Error Snippet | Cause | Fix |
|---------------|-------|-----|
| `key not found ... valoper... .info` | Positional arg interpreted as key name | Pass the key name (`validator` / `validator-nodeN`) instead of valoper |
| `accepts 1 arg(s), received 0` | Command requires positional key name | Add the key name after subcommand |
| `insufficient funds` | Fee > liquid balance | Lower fee or fund account |
| `account sequence mismatch` | Concurrent tx or stale sequence | Query account (`query auth account`) and retry with correct sequence if using manual sign |
| Rewards still non-zero immediately after withdraw | New block produced between withdraw and query | Re-query after a short delay |

---
## 9. Automation Scripts
Existing helper scripts:
* `scripts/withdraw_commission.sh` – Auto-detects command style (may still need adjustment for key-name positional quirk; pass KEY=validator-node1 HOME_DIR=...).
* `scripts/list_local_rewards.sh` – Summarizes balances, commission, rewards (set `--human` for token scaling).

Example automation invocation:
```bash
./scripts/withdraw_commission.sh KEY=validator-node1 HOME_DIR=$(pwd)/.local/node1
```

---
## 10. Raw JSON Fallback (Advanced)
If CLI parsing keeps failing, craft and sign a raw tx.
```bash
VALOPER=$VALOPER_NODE1
cat > withdraw_commission_unsigned.json <<EOF
{
  "body": {"messages": [{"@type": "/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommission","validator_address": "$VALOPER"}],"memo": "","timeout_height": "0","extension_options": [],"non_critical_extension_options": []},
  "auth_info": {"signer_infos": [],"fee": {"amount": [{"denom": "ashm","amount": "699684826939954723453"}],"gas_limit": "500000","payer": "","granter": ""}},
  "signatures": []
}
EOF
```
Sign & broadcast:
```bash
shardeumd tx sign withdraw_commission_unsigned.json \
  --from validator-node1 --home .local/node1 --keyring-backend test \
  --chain-id shardeum_8119-1 -o json > withdraw_commission_signed.json
shardeumd tx broadcast withdraw_commission_signed.json -o json | jq
```

---
## 11. Summary Flow (Node1)
```bash
# Prepare
ACC_NODE1=$(shardeumd keys show validator-node1 --home .local/node1 --keyring-backend test -a)
VALOPER_NODE1=$(shardeumd keys show validator-node1 --home .local/node1 --keyring-backend test --bech val -a)

# Inspect
shardeumd query distribution commission "$VALOPER_NODE1" --home .local/node0 -o json | jq
shardeumd query distribution rewards "$ACC_NODE1" "$VALOPER_NODE1" --home .local/node0 -o json | jq

# Withdraw commission
shardeumd tx distribution withdraw-validator-commission validator-node1 \
  --from validator-node1 --home .local/node1 --keyring-backend test \
  --chain-id shardeum_8119-1 --gas auto --gas-adjustment 1.3 \
  --fees 699684826939954723453ashm -y

# (Optional) Withdraw delegator rewards
shardeumd tx distribution withdraw-rewards "$VALOPER_NODE1" \
  --from validator-node1 --home .local/node1 --keyring-backend test \
  --chain-id shardeum_8119-1 --gas auto --gas-adjustment 1.3 \
  --fees 699684826939954723453ashm -y

# Verify
shardeumd query distribution commission "$VALOPER_NODE1" --home .local/node0 -o json | jq
shardeumd query distribution rewards "$ACC_NODE1" "$VALOPER_NODE1" --home .local/node0 -o json | jq
shardeumd query bank balances "$ACC_NODE1" --home .local/node0 -o json | jq
```

---
## 12. Next Enhancements (Optional)
* Script to withdraw commission for all local validators automatically.
* Add staking status (`tokens`, `commission rate`) columns to `list_local_rewards.sh`.
* Alert or cron when commission exceeds threshold.

---
*Document version: v1 (local operator guide – commission & rewards).* 
