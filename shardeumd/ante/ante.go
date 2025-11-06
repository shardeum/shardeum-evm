package ante

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"github.com/shardeum/shardeum-evm/ante"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
)

// NewAnteHandler returns an ante handler responsible for attempting to route an
// Ethereum or SDK transaction to an internal ante handler for performing
// transaction-level processing (e.g. fee payment, signature verification) before
// being passed onto it's respective handler.
func NewAnteHandler(options ante.HandlerOptions) sdk.AnteHandler {
	return func(
		ctx sdk.Context, tx sdk.Tx, sim bool,
	) (newCtx sdk.Context, err error) {
		var anteHandler sdk.AnteHandler

		fmt.Printf("\n🔵🔵🔵 ANTE HANDLER ENTRY (BUILD v3) 🔵🔵🔵\n")
		fmt.Printf("Transaction type: %T\n", tx)

		txWithExtensions, ok := tx.(authante.HasExtensionOptionsTx)
		if ok {
			opts := txWithExtensions.GetExtensionOptions()
			fmt.Printf("Has extension options: %d option(s)\n", len(opts))
			if len(opts) > 0 {
				typeURL := opts[0].GetTypeUrl()
				fmt.Printf("Extension type URL: %s\n", typeURL)
				switch typeURL {
				case "/cosmos.evm.vm.v1.ExtensionOptionsEthereumTx":
					// handle as *evmtypes.MsgEthereumTx
					fmt.Printf("→ Routing to EVM ante handler\n")
					anteHandler = newMonoEVMAnteHandler(options)
				case "/cosmos.evm.types.v1.ExtensionOptionDynamicFeeTx":
					// cosmos-sdk tx with dynamic fee extension
					fmt.Printf("→ Routing to Cosmos ante handler (DynamicFeeTx)\n")
					anteHandler = newCosmosAnteHandler(options)
				case "/cosmos.evm.types.v1.ExtensionOptionsWeb3Tx":
					// cosmos-sdk tx with EIP-712 signature in Web3Tx extension
					fmt.Printf("→ Routing to Cosmos ante handler (Web3Tx/EIP-712)\n🔵🔵🔵🔵🔵🔵🔵🔵🔵🔵🔵🔵🔵🔵🔵\n\n")
					anteHandler = newCosmosAnteHandler(options)
				default:
					fmt.Printf("❌ Unknown extension option: %s\n", typeURL)
					return ctx, errorsmod.Wrapf(
						errortypes.ErrUnknownExtensionOptions,
						"rejecting tx with unsupported extension option: %s", typeURL,
					)
				}

				return anteHandler(ctx, tx, sim)
			}
		} else {
			fmt.Printf("No extension options interface\n")
		}

		// handle as totally normal Cosmos SDK tx
		fmt.Printf("→ Routing to Cosmos ante handler (standard)\n🔵🔵🔵🔵🔵🔵🔵🔵🔵🔵🔵🔵🔵🔵🔵\n\n")
		switch tx.(type) {
		case sdk.Tx:
			anteHandler = newCosmosAnteHandler(options)
		default:
			return ctx, errorsmod.Wrapf(errortypes.ErrUnknownRequest, "invalid transaction type: %T", tx)
		}

		return anteHandler(ctx, tx, sim)
	}
}
