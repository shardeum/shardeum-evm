package ante

import (
	"testing"

	"github.com/shardeum/shardeum-evm/shardeumd/tests/integration"
	"github.com/shardeum/shardeum-evm/tests/integration/ante"
)

func TestAnte_Integration(t *testing.T) {
	ante.TestIntegrationAnteHandler(t, integration.CreateShardeum)
}
