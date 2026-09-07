package contracts

import (
	contractutils "github.com/shardeum/shardeum-evm/contracts/utils"
	evmtypes "github.com/shardeum/shardeum-evm/x/vm/types"
)

func LoadContractCreationTester() (evmtypes.CompiledContract, error) {
	return contractutils.LoadContractFromJSONFile("solidity/ContractCreationTester.json")
}
