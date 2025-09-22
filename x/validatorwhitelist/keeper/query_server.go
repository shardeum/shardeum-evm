package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/shardeum/shardeum-evm/x/validatorwhitelist/types"
)

var _ types.QueryServer = &Keeper{}

// Params queries the parameters of the validator whitelist module
func (k Keeper) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	return &types.QueryParamsResponse{
		Params: *params,
	}, nil
}