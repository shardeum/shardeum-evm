package contracts

import (
	contractutils "github.com/shardeum/shardeum-evm/contracts/utils"
	evmtypes "github.com/shardeum/shardeum-evm/x/vm/types"
)

func LoadSequentialICS20Sender() (evmtypes.CompiledContract, error) {
	return contractutils.LoadContractFromJSONFile("solidity/SequentialICS20Sender.json")
}
