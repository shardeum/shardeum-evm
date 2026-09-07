package contracts

import (
	contractutils "github.com/shardeum/shardeum-evm/contracts/utils"
	evmtypes "github.com/shardeum/shardeum-evm/x/vm/types"
)

func LoadSequentialOperationsTester() (evmtypes.CompiledContract, error) {
	return contractutils.LoadContractFromJSONFile("solidity/SequentialOperationsTester.json")
}
