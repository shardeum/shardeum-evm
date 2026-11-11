package shardeumd

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/types/module"
)

// UpgradeName defines the on-chain upgrade name for the resync onto
// cosmos/evm v0.6.4 (security fixes GHSA-p8x8-4gg6-prcc, GHSA-7xxp-gq2c-x47x,
// GHSA-c6xv-pppp-rjw6, GHSA-wm4f-qpxp-rch4 and #1277). The release is
// consensus-breaking (EVM execution semantics change), so it must go through
// a MsgSoftwareUpgrade governance proposal.
//
// Only the current upgrade's handler is registered, matching upstream evmd.
// x/upgrade requires the running binary to have a handler for the last
// applied upgrade; on a chain still on v1.1.0 that is "v0.4.0-to-v0.6.3",
// which this binary does not register, so it errors on its first block until
// the "v0.6.3-to-v0.6.4" height is reached. Nodes replaying history from
// genesis must therefore step through binaries: the pre-v1.1.0 release up to
// the "v0.4.0-to-v0.6.3" height (mainnet 7,927,860), v1.1.0 up to the
// "v0.6.3-to-v0.6.4" height, then this binary. State sync avoids this.
//
// The name must match the name used in the MsgSoftwareUpgrade proposal and in
// each node's cosmovisor upgrades/<name>/bin directory.
const UpgradeName = "v0.6.3-to-v0.6.4"

func (app ShardeumApp) RegisterUpgradeHandlers() {
	// v0.6.3 -> v0.6.4: no store changes and no new state to backfill. The
	// upgrade exists only to coordinate the binary switch at a single height.
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeName,
		func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
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
