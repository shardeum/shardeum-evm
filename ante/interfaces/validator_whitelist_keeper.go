package interfaces

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	vwkeeper "github.com/shardeum/shardeum-evm/x/validatorwhitelist/keeper"
)

// OnChainValidatorWhitelistKeeper adapts the on-chain module keeper to the ante interface
type OnChainValidatorWhitelistKeeper struct {
    Keeper vwkeeper.Keeper
}

func NewOnChainValidatorWhitelistKeeper(k vwkeeper.Keeper) ValidatorWhitelistKeeper {
    return &OnChainValidatorWhitelistKeeper{Keeper: k}
}

func (k *OnChainValidatorWhitelistKeeper) IsWhitelistEnabled(ctx sdk.Context) bool {
    params := k.Keeper.GetParams(ctx)
    return params.Enabled
}

func (k *OnChainValidatorWhitelistKeeper) CanValidatorReceiveDelegation(ctx sdk.Context, validatorAddr string) bool {
    return k.Keeper.IsValidatorAllowed(ctx, validatorAddr)
}

// AppConfigValidatorWhitelistKeeper implements ValidatorWhitelistKeeper using app configuration
type AppConfigValidatorWhitelistKeeper struct {
	enabled  bool
	allowset map[string]struct{}
}

// NewAppConfigValidatorWhitelistKeeper creates a new config-based validator whitelist keeper
func NewAppConfigValidatorWhitelistKeeper(enabled bool, allowlist []string) ValidatorWhitelistKeeper {
	allowset := make(map[string]struct{})
	for _, addr := range allowlist {
		addr = strings.TrimSpace(addr)
		if addr != "" {
			allowset[addr] = struct{}{}
		}
	}
	return &AppConfigValidatorWhitelistKeeper{
		enabled:  enabled,
		allowset: allowset,
	}
}

// IsWhitelistEnabled returns whether the whitelist is currently enabled
func (k *AppConfigValidatorWhitelistKeeper) IsWhitelistEnabled(ctx sdk.Context) bool {
	return k.enabled
}

// CanValidatorReceiveDelegation checks if a validator can receive delegations
// This controls which validators can join consensus through delegation
func (k *AppConfigValidatorWhitelistKeeper) CanValidatorReceiveDelegation(ctx sdk.Context, validatorAddr string) bool {
	if !k.enabled {
		return true // If disabled, all validators can receive delegations
	}
	
	_, ok := k.allowset[validatorAddr]
	return ok
}
