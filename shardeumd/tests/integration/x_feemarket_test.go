package integration

import (
	"testing"

	"github.com/shardeum/shardeum-evm/tests/integration/x/feemarket"
	"github.com/stretchr/testify/suite"
)

func TestFeeMarketKeeperTestSuite(t *testing.T) {
	s := feemarket.NewTestKeeperTestSuite(CreateShardeum)
	suite.Run(t, s)
}
