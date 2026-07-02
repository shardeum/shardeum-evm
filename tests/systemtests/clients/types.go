package clients

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shardeum/shardeum-evm/crypto/ethsecp256k1"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type EthAccount struct {
	Address common.Address
	PrivKey *ecdsa.PrivateKey
}

type CosmosAccount struct {
	AccAddress    sdk.AccAddress
	AccountNumber uint64
	PrivKey       *ethsecp256k1.PrivKey
}

type TxPoolResult struct {
	Pending map[string]map[string]*EthRPCTransaction `json:"pending"`
	Queued  map[string]map[string]*EthRPCTransaction `json:"queued"`
}

type EthRPCTransaction struct {
	Hash common.Hash `json:"hash"`
}
