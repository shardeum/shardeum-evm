package shardeumd

import (
	cmn "github.com/shardeum/shardeum-evm/precompiles/common"
	evmtypes "github.com/shardeum/shardeum-evm/x/vm/types"
)

type BankKeeper interface {
	evmtypes.BankKeeper
	cmn.BankKeeper
}
