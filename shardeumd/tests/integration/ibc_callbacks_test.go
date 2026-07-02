package integration

import (
	"testing"

	"github.com/shardeum/shardeum-evm/tests/integration/x/ibc/callbacks"
	"github.com/stretchr/testify/suite"
)

func TestIBCCallback(t *testing.T) {
	suite.Run(t, callbacks.NewKeeperTestSuite(CreateShardeum))
}
