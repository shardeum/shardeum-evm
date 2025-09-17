package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/shardeum/shardeum-evm/tests/integration/rpc/backend"
)

func TestBackend(t *testing.T) {
	s := backend.NewTestSuite(CreateShardeum)
	suite.Run(t, s)
}
