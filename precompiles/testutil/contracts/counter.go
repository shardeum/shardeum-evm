package contracts

import (
	contractutils "github.com/shardeum/shardeum-evm/contracts/utils"
	evmtypes "github.com/shardeum/shardeum-evm/x/vm/types"
)

func LoadCounterContract() (evmtypes.CompiledContract, error) {
	return contractutils.LoadContractFromJSONFile("Counter.json")
}
