package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/shardeum/shardeum-evm/tests/integration/eip712"
)

func TestEIP712TestSuite(t *testing.T) {
	s := eip712.NewTestSuite(CreateShardeum, false)
	suite.Run(t, s)

	// Note that we don't test the Legacy EIP-712 Extension, since that case
	// is sufficiently covered by the AnteHandler tests.
	s = eip712.NewTestSuite(CreateShardeum, true)
	suite.Run(t, s)
}
