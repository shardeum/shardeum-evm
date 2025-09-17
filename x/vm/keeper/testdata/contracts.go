package testdata

import (
	contractutils "github.com/shardeum/shardeum-evm/contracts/utils"
	evmtypes "github.com/shardeum/shardeum-evm/x/vm/types"
)

func LoadERC20Contract() (evmtypes.CompiledContract, error) {
	return contractutils.LegacyLoadContractFromJSONFile("ERC20Contract.json")
}

func LoadMessageCallContract() (evmtypes.CompiledContract, error) {
	return contractutils.LegacyLoadContractFromJSONFile("MessageCallContract.json")
}
