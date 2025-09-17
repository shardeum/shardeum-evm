package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/shardeum/shardeum-evm/tests/integration/x/precisebank"
)

func TestPreciseBankGenesis(t *testing.T) {
	s := precisebank.NewGenesisTestSuite(CreateShardeum)
	suite.Run(t, s)
}

func TestPreciseBankKeeper(t *testing.T) {
	s := precisebank.NewKeeperIntegrationTestSuite(CreateShardeum)
	suite.Run(t, s)
}
