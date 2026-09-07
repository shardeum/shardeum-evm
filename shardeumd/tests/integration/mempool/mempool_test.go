package mempool

import (
	"testing"

	"github.com/shardeum/shardeum-evm/shardeumd/tests/integration"
	"github.com/shardeum/shardeum-evm/tests/integration/mempool"
	"github.com/stretchr/testify/suite"
)

func TestMempoolIntegrationTestSuite(t *testing.T) {
	suite.Run(t, mempool.NewMempoolIntegrationTestSuite(integration.CreateShardeum))
}
