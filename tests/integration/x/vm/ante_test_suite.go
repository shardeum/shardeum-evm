package vm

import (
	"github.com/shardeum/shardeum-evm/testutil/integration/evm/network"
	"github.com/stretchr/testify/suite"
)

type EvmAnteTestSuite struct {
	suite.Suite

	create  network.CreateEvmApp
	options []network.ConfigOption
}

func NewEvmAnteTestSuite(create network.CreateEvmApp, opts ...network.ConfigOption) *EvmAnteTestSuite {
	return &EvmAnteTestSuite{
		create:  create,
		options: opts,
	}
}
