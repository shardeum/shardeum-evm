package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/shardeum/shardeum-evm/tests/integration/x/ibc"
)

func TestIBCKeeperTestSuite(t *testing.T) {
	s := ibc.NewKeeperTestSuite(CreateShardeum)
	suite.Run(t, s)
}
