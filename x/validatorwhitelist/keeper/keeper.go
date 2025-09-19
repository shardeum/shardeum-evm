package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/shardeum/shardeum-evm/x/validatorwhitelist/types"
)

type Keeper struct {
	cdc      codec.Codec
	storeKey storetypes.StoreKey
	types.UnimplementedMsgServer
	types.UnimplementedQueryServer
}

// NewKeeper creates a new validator whitelist Keeper instance
func NewKeeper(cdc codec.Codec, storeKey storetypes.StoreKey) Keeper {
	return Keeper{
		cdc:      cdc,
		storeKey: storeKey,
	}
}

// GetParams returns the current validator whitelist parameters
func (k Keeper) GetParams(ctx sdk.Context) (params *types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	params = &types.Params{}
	k.cdc.MustUnmarshal(bz, params)
	return params
}

// SetParams sets the validator whitelist parameters
func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(params)
	store.Set(types.ParamsKey, bz)
}

// IsValidatorAllowed checks if a validator is allowed to receive delegations
func (k Keeper) IsValidatorAllowed(ctx sdk.Context, validatorAddr string) bool {
	params := k.GetParams(ctx)
	return params.IsValidatorAllowed(validatorAddr)
}
