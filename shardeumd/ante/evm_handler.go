package ante

import (
	"github.com/shardeum/shardeum-evm/ante"
	evmante "github.com/shardeum/shardeum-evm/ante/evm"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// newMonoEVMAnteHandler creates the sdk.AnteHandler implementation for the EVM transactions.
func newMonoEVMAnteHandler(options ante.HandlerOptions) sdk.AnteHandler {
	decorators := []sdk.AnteDecorator{
		evmante.NewEVMMonoDecorator(
			options.AccountKeeper,
			options.FeeMarketKeeper,
			options.EvmKeeper,
			options.MaxTxGasWanted,
		),
		ante.NewTxListenerDecorator(options.PendingTxListener),
	}

	return sdk.ChainAnteDecorators(decorators...)
}
