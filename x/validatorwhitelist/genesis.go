package validatorwhitelist

import (
	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/shardeum/shardeum-evm/x/validatorwhitelist/keeper"
	"github.com/shardeum/shardeum-evm/x/validatorwhitelist/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes genesis state based on exported genesis
func InitGenesis(
	ctx sdk.Context,
	k keeper.Keeper,
	data types.GenesisState,
) []abci.ValidatorUpdate {
	k.SetParams(ctx, &data.Params)

	return []abci.ValidatorUpdate{}
}

// ExportGenesis exports genesis state of the validator whitelist module
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	params := k.GetParams(ctx)
	return &types.GenesisState{
		Params: *params,
	}
}
