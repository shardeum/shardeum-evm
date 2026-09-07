package gov

import (
	"testing"

	"github.com/shardeum/shardeum-evm/shardeumd/tests/integration"
	"github.com/shardeum/shardeum-evm/tests/integration/precompiles/gov"
	"github.com/stretchr/testify/suite"
)

func TestGovPrecompileTestSuite(t *testing.T) {
	s := gov.NewPrecompileTestSuite(integration.CreateShardeum)
	suite.Run(t, s)
}

func TestGovPrecompileIntegrationTestSuite(t *testing.T) {
	gov.TestPrecompileIntegrationTestSuite(t, integration.CreateShardeum)
}
