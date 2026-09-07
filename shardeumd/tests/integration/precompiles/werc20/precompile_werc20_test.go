package werc20

import (
	"testing"

	"github.com/shardeum/shardeum-evm/shardeumd/tests/integration"
	"github.com/shardeum/shardeum-evm/tests/integration/precompiles/werc20"
	"github.com/stretchr/testify/suite"
)

func TestWERC20PrecompileUnitTestSuite(t *testing.T) {
	s := werc20.NewPrecompileUnitTestSuite(integration.CreateShardeum)
	suite.Run(t, s)
}

func TestWERC20PrecompileIntegrationTestSuite(t *testing.T) {
	werc20.TestPrecompileIntegrationTestSuite(t, integration.CreateShardeum)
}
