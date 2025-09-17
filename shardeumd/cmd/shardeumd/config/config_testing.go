//go:build test
// +build test

package config

import (
	evmconfig "github.com/shardeum/shardeum-evm/config"
	testconfig "github.com/shardeum/shardeum-evm/testutil/config"
)

// EvmAppOptions allows to setup the global configuration
// for the Cosmos EVM chain.
func EvmAppOptions(chainID uint64) error {
	return evmconfig.EvmAppOptionsWithConfigWithReset(chainID, testconfig.TestChainsCoinInfo, cosmosEVMActivators, true)
}
