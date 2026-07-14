//go:build test
// +build test

package config

import (
	evmtypes "github.com/shardeum/shardeum-evm/x/vm/types"
)

// TestChainsCoinInfo is a map of the chain id and its corresponding EvmCoinInfo
// that allows initializing the app with different coin info based on the
// chain id
var TestChainsCoinInfo = map[uint64]evmtypes.EvmCoinInfo{
	EighteenDecimalsChainID: {
		Denom:         ShardeumChainDenom,
		ExtendedDenom: ShardeumChainDenom,
		DisplayDenom:  ShardeumDisplayDenom,
		Decimals:      evmtypes.EighteenDecimals.Uint32(),
	},
	SixDecimalsChainID: {
		Denom:         "utest",
		ExtendedDenom: "atest",
		DisplayDenom:  "test",
		Decimals:      evmtypes.SixDecimals.Uint32(),
	},
	TwelveDecimalsChainID: {
		Denom:         "ptest2",
		ExtendedDenom: "atest2",
		DisplayDenom:  "test2",
		Decimals:      evmtypes.TwelveDecimals.Uint32(),
	},
	TwoDecimalsChainID: {
		Denom:         "ctest3",
		ExtendedDenom: "atest3",
		DisplayDenom:  "test3",
		Decimals:      evmtypes.TwoDecimals.Uint32(),
	},
	TestChainID1: {
		Denom:         ShardeumChainDenom,
		ExtendedDenom: ShardeumChainDenom,
		DisplayDenom:  ShardeumChainDenom,
		Decimals:      evmtypes.EighteenDecimals.Uint32(),
	},
	TestChainID2: {
		Denom:         ShardeumChainDenom,
		ExtendedDenom: ShardeumChainDenom,
		DisplayDenom:  ShardeumChainDenom,
		Decimals:      evmtypes.EighteenDecimals.Uint32(),
	},
	EVMChainID: {
		Denom:         ShardeumChainDenom,
		ExtendedDenom: ShardeumChainDenom,
		DisplayDenom:  ShardeumDisplayDenom,
		Decimals:      evmtypes.EighteenDecimals.Uint32(),
	},
}
