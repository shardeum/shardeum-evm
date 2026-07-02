package integration

import (
	"testing"

	"github.com/shardeum/shardeum-evm/tests/integration/x/precisebank"
	"github.com/stretchr/testify/suite"
)

func TestPreciseBankGenesis(t *testing.T) {
	s := precisebank.NewGenesisTestSuite(CreateShardeum)
	suite.Run(t, s)
}

func TestPreciseBankKeeper(t *testing.T) {
	s := precisebank.NewKeeperIntegrationTestSuite(CreateShardeum)
	suite.Run(t, s)
}
