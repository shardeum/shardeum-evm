package integration

import (
	"encoding/json"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/shardeum/shardeum-evm"
	shardeumd "github.com/shardeum/shardeum-evm/shardeumd"
	testconfig "github.com/shardeum/shardeum-evm/testutil/config"
	"github.com/shardeum/shardeum-evm/testutil/constants"
	feemarkettypes "github.com/shardeum/shardeum-evm/x/feemarket/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"

	clienthelpers "cosmossdk.io/client/v2/helpers"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/baseapp"
	simutils "github.com/cosmos/cosmos-sdk/testutil/sims"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// CreateShardeum creates an evm app for regular integration tests (non-mempool)
// This version uses a noop mempool to avoid state issues during transaction processing
func CreateShardeum(chainID string, evmChainID uint64, customBaseAppOptions ...func(*baseapp.BaseApp)) evm.EvmApp {
	defaultNodeHome, err := clienthelpers.GetNodeHomeDirectory(".shardeumd")
	if err != nil {
		panic(err)
	}

	db := dbm.NewMemDB()
	logger := log.NewNopLogger()
	loadLatest := true
	appOptions := simutils.NewAppOptionsWithFlagHome(defaultNodeHome)

	baseAppOptions := append(customBaseAppOptions, baseapp.SetChainID(chainID))

	return shardeumd.NewShardeumApp(
		logger,
		db,
		nil,
		loadLatest,
		appOptions,
		evmChainID,
		testconfig.EvmAppOptions,
		baseAppOptions...,
	)
}

// SetupShardeum initializes a new shardeumd app with default genesis state.
// It is used in IBC integration tests to create a new shardeumd app instance.
func SetupShardeum() (ibctesting.TestingApp, map[string]json.RawMessage) {
	app := shardeumd.NewShardeumApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		simutils.EmptyAppOptions{},
		constants.ShardeumEIP155ChainID,
		testconfig.EvmAppOptions,
	)
	// disable base fee for testing
	genesisState := app.DefaultGenesis()
	fmGen := feemarkettypes.DefaultGenesisState()
	fmGen.Params.NoBaseFee = true
	genesisState[feemarkettypes.ModuleName] = app.AppCodec().MustMarshalJSON(fmGen)
	stakingGen := stakingtypes.DefaultGenesisState()
	stakingGen.Params.BondDenom = constants.ShardeumAttoDenom
	genesisState[stakingtypes.ModuleName] = app.AppCodec().MustMarshalJSON(stakingGen)
	mintGen := minttypes.DefaultGenesisState()
	mintGen.Params.MintDenom = constants.ShardeumAttoDenom
	genesisState[minttypes.ModuleName] = app.AppCodec().MustMarshalJSON(mintGen)

	return app, genesisState
}
