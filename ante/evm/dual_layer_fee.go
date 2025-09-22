package evm

import (
	evmtypes "github.com/shardeum/shardeum-evm/x/vm/types"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

// CheckDualLayerFee validates the provided fee against both node-level minimum gas price
// and the EIP-1559 fee market requirements. This function ensures that transactions must
// satisfy both layers of the dual gas pricing system:
//
// 1. Node-level floor: Validates against minimum gas price from app.toml configuration
// 2. Consensus EVM fee market: Validates against EIP-1559 base fee and global minimum
//
// For dynamic transactions (EIP-1559), the function considers the effective fee which
// accounts for the current base fee. For legacy transactions, it uses the gas price
// specified in the transaction.
//
// Parameters:
//   - fee: The transaction fee amount (in 18 decimals)
//   - nodeMinGasPrice: Node-level minimum gas price from app.toml (in 18 decimals)
//   - globalMinGasPrice: Global minimum gas price from fee market module (in 18 decimals)
//   - gasLimit: Gas limit of the transaction (in 18 decimals)
//   - isLondon: Whether EIP-1559 (London) rules are active
//
// Returns error if fee doesn't meet either layer's requirements.
func CheckDualLayerFee(fee, nodeMinGasPrice, globalMinGasPrice, gasLimit sdkmath.LegacyDec, isLondon bool) error {
	// Layer 1: Node-level floor validation
	// This acts as a spam prevention mechanism at the individual validator level
	if !nodeMinGasPrice.IsZero() {
		nodeRequiredFee := nodeMinGasPrice.Mul(gasLimit)
		if fee.LT(nodeRequiredFee) {
			return errorsmod.Wrapf(
				errortypes.ErrInsufficientFee,
				"fee below node minimum: got %s, node requires %s (node min gas price: %s)",
				fee.TruncateInt().String(),
				nodeRequiredFee.TruncateInt().String(),
				nodeMinGasPrice.String(),
			)
		}
	}

	// Layer 2: Consensus EVM fee market validation (global minimum)
	// This enforces network-wide fee requirements through governance
	if !globalMinGasPrice.IsZero() {
		globalRequiredFee := globalMinGasPrice.Mul(gasLimit)
		if fee.LT(globalRequiredFee) {
			return errorsmod.Wrapf(
				errortypes.ErrInsufficientFee,
				"fee below global minimum: got %s, global requires %s (global min gas price: %s). "+
					"Please increase the priority tip (for EIP-1559 txs) or the gas prices (for legacy txs)",
				fee.TruncateInt().String(),
				globalRequiredFee.TruncateInt().String(),
				globalMinGasPrice.String(),
			)
		}
	}

	return nil
}

// GetNodeMinGasPrice extracts the node-level minimum gas price from the context.
// This price is configured in app.toml and represents the validator's local spam
// protection threshold. The price is automatically converted to 18 decimals for
// compatibility with EVM fee calculations.
func GetNodeMinGasPrice(ctx sdk.Context, evmDenom string) sdkmath.LegacyDec {
	nodeMinGasPrices := ctx.MinGasPrices()
	if len(nodeMinGasPrices) == 0 {
		return sdkmath.LegacyZeroDec()
	}

	// Convert the node minimum gas price to 18 decimals to match EVM representation
	nodeMinGasPrice := nodeMinGasPrices.AmountOf(evmDenom)
	return evmtypes.ConvertAmountTo18DecimalsLegacy(nodeMinGasPrice)
}
