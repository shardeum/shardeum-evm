package integration

import (
	"testing"

	"github.com/shardeum/shardeum-evm/tests/integration/x/ibc"
	"github.com/stretchr/testify/suite"
)

func TestIBCKeeperTestSuite(t *testing.T) {
	s := ibc.NewKeeperTestSuite(CreateShardeum)
	suite.Run(t, s)
}
