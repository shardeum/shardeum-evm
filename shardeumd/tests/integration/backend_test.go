package integration

import (
	"testing"

	"github.com/shardeum/shardeum-evm/tests/integration/rpc/backend"
	"github.com/stretchr/testify/suite"
)

func TestBackend(t *testing.T) {
	s := backend.NewTestSuite(CreateShardeum)
	suite.Run(t, s)
}
