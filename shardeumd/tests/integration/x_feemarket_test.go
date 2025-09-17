package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/shardeum/shardeum-evm/tests/integration/x/feemarket"
)

func TestFeeMarketKeeperTestSuite(t *testing.T) {
	s := feemarket.NewTestKeeperTestSuite(CreateShardeum)
	suite.Run(t, s)
}
