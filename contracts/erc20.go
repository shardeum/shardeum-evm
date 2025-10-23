package contracts

import (
	contractutils "github.com/shardeum/shardeum-evm/contracts/utils"
	evmtypes "github.com/shardeum/shardeum-evm/x/vm/types"

	_ "embed"
)

var (
	// ERC20MinterBurnerDecimalsJSON are the compiled bytes of the ERC20MinterBurnerDecimalsContract
	//
	//go:embed solidity/ERC20MinterBurnerDecimals.json
	ERC20MinterBurnerDecimalsJSON []byte

	// ERC20MinterBurnerDecimalsContract is the compiled erc20 contract
	ERC20MinterBurnerDecimalsContract evmtypes.CompiledContract
)

func init() {
	var err error
	if ERC20MinterBurnerDecimalsContract, err = contractutils.ConvertHardhatBytesToCompiledContract(
		ERC20MinterBurnerDecimalsJSON,
	); err != nil {
		panic(err)
	}
}
