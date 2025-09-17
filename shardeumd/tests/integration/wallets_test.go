package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/shardeum/shardeum-evm/tests/integration/wallets"
)

func TestLedgerTestSuite(t *testing.T) {
	s := wallets.NewLedgerTestSuite(CreateShardeum)
	suite.Run(t, s)
}
