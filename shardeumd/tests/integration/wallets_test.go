package integration

import (
	"testing"

	"github.com/shardeum/shardeum-evm/tests/integration/wallets"
	"github.com/stretchr/testify/suite"
)

func TestLedgerTestSuite(t *testing.T) {
	s := wallets.NewLedgerTestSuite(CreateShardeum)
	suite.Run(t, s)
}
