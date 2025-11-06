package codec

import (
	"github.com/shardeum/shardeum-evm/crypto/ethsecp256k1"

	"github.com/cosmos/gogoproto/proto"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

// RegisterInterfaces register the Cosmos EVM key concrete types.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*cryptotypes.PubKey)(nil), &ethsecp256k1.PubKey{})
	registry.RegisterImplementations((*cryptotypes.PrivKey)(nil), &ethsecp256k1.PrivKey{})
}

func init() {
	// Register the same concrete types with Ethermint URLs for Keplr compatibility
	proto.RegisterType((*ethsecp256k1.PubKey)(nil), "ethermint.crypto.v1.ethsecp256k1.PubKey")
	proto.RegisterType((*ethsecp256k1.PrivKey)(nil), "ethermint.crypto.v1.ethsecp256k1.PrivKey")
}
