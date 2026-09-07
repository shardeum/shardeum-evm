package bank

import (
	"testing"

	"github.com/shardeum/shardeum-evm/shardeumd/tests/integration"
	"github.com/shardeum/shardeum-evm/tests/integration/precompiles/bank"
	"github.com/stretchr/testify/suite"
)

func TestBankPrecompileTestSuite(t *testing.T) {
	s := bank.NewPrecompileTestSuite(integration.CreateShardeum)
	suite.Run(t, s)
}

func TestBankPrecompileIntegrationTestSuite(t *testing.T) {
	bank.TestIntegrationSuite(t, integration.CreateShardeum)
}
