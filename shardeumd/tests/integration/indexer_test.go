package integration

import (
	"testing"

	"github.com/shardeum/shardeum-evm/tests/integration/indexer"
)

func TestKVIndexer(t *testing.T) {
	indexer.TestKVIndexer(t, CreateShardeum)
}
