package shardeumd

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// UpgradeName defines the on-chain upgrade name for the Shardeum resync onto
// cosmos/evm v0.6.3.
//
// NOTE: the live chains are v0.4.0-era. They never executed the v0.4.0->v0.5.0
// upgrade (that handler was dead reference scaffolding), so this upgrade jumps
// straight from v0.4.0-era state to v0.6.3, skipping v0.5.0 and v0.6.0-v0.6.2.
// The name must match the name used in the MsgSoftwareUpgrade proposal and in
// each node's cosmovisor upgrades/<name>/bin directory.
const UpgradeName = "v0.4.0-to-v0.6.3"

func (app ShardeumApp) RegisterUpgradeHandlers() {
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeName,
		func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			sdkCtx := sdk.UnwrapSDKContext(ctx)

			// v0.5.0 moved EVM coin info (denom/decimals) from app-creation-time
			// config into x/vm's own KVStore, populated only by InitGenesis. Chains
			// upgrading from pre-v0.5.0 (as this one is) never had InitGenesis run
			// with that logic, so the store entry must be backfilled here or every
			// EVM operation panics with "denom not registered". See
			// docs/migrations/v0.4.0_to_v0.5.0.md, "UpgradeHandler" section.
			if err := app.EVMKeeper.InitEvmCoinInfo(sdkCtx); err != nil {
				return nil, err
			}

			return app.ModuleManager.RunMigrations(ctx, app.Configurator(), fromVM)
		},
	)

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(err)
	}

	if upgradeInfo.Name == UpgradeName && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{},
		}
		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}
