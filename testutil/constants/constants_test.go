package constants_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/shardeum/shardeum-evm/testutil/config"
	"github.com/shardeum/shardeum-evm/testutil/constants"
)

func TestRequireSameTestDenom(t *testing.T) {
	require.Equal(t,
		constants.ShardeumAttoDenom,
		config.ShardeumChainDenom,
		"test denoms should be the same across the repo",
	)
}

func TestRequireSameTestBech32Prefix(t *testing.T) {
	require.Equal(t,
		constants.ExampleBech32Prefix,
		config.Bech32Prefix,
		"bech32 prefixes should be the same across the repo",
	)
}

func TestRequireSameWEVMOSMainnet(t *testing.T) {
	require.Equal(t,
		constants.ShardeumChainID.EVMChainID,
		uint64(config.ShardeumChainID),
		"EVM chain IDs should be the same across the repo",
	)
}
