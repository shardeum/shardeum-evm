package erc20

import (
	"testing"

	"github.com/shardeum/shardeum-evm/shardeumd/tests/integration"
	erc21 "github.com/shardeum/shardeum-evm/tests/integration/precompiles/erc20"
	"github.com/stretchr/testify/suite"
)

func TestErc20PrecompileTestSuite(t *testing.T) {
	s := erc21.NewPrecompileTestSuite(integration.CreateShardeum)
	suite.Run(t, s)
}

func TestErc20IntegrationTestSuite(t *testing.T) {
	erc21.TestIntegrationTestSuite(t, integration.CreateShardeum)
}
