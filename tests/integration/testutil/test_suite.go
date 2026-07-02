//go:build test

package testutil

import (
	"github.com/shardeum/shardeum-evm/testutil/integration/evm/network"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite

	create  network.CreateEvmApp
	options []network.ConfigOption
}

func NewTestSuite(create network.CreateEvmApp, options ...network.ConfigOption) *TestSuite {
	return &TestSuite{
		create:  create,
		options: options,
	}
}
