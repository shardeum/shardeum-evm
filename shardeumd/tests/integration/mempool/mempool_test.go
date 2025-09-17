package mempool

import (
	"testing"

	"github.com/shardeum/shardeum-evm/shardeumd/tests/integration"

	"github.com/stretchr/testify/suite"

	"github.com/shardeum/shardeum-evm/tests/integration/mempool"
)

func TestMempoolIntegrationTestSuite(t *testing.T) {
	suite.Run(t, mempool.NewMempoolIntegrationTestSuite(integration.CreateShardeum))
}
