package cosmos

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	anteinterfaces "github.com/shardeum/shardeum-evm/ante/interfaces"
)

// ValidatorWhitelistDecorator enforces an allowlist for validator delegation messages
// using the ValidatorWhitelistKeeper interface. Controls which validators can receive
// delegations and join consensus. When disabled, it is a no-op.
type ValidatorWhitelistDecorator struct {
	whitelistKeeper anteinterfaces.ValidatorWhitelistKeeper
}

func NewValidatorWhitelistDecorator(whitelistKeeper anteinterfaces.ValidatorWhitelistKeeper) sdk.AnteDecorator {
	return &ValidatorWhitelistDecorator{
		whitelistKeeper: whitelistKeeper,
	}
}

func (d ValidatorWhitelistDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// If whitelist keeper is nil or disabled, allow all delegation operations
	if d.whitelistKeeper == nil || !d.whitelistKeeper.IsWhitelistEnabled(ctx) {
		return next(ctx, tx, simulate)
	}

	// Check each message in the transaction for delegation restrictions
	msgs := tx.GetMsgs()
	for _, msg := range msgs {
		switch m := msg.(type) {
		case *stakingtypes.MsgCreateValidator:
			// Block self-delegation at creation time for non-whitelisted validators
			if !d.whitelistKeeper.CanValidatorReceiveDelegation(ctx, m.ValidatorAddress) {
				zeroGasCtx := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
				return zeroGasCtx, errorsmod.Wrap(
					sdkerrors.ErrUnauthorized,
					fmt.Sprintf("validator %s is not whitelisted for consensus (create validator)", m.ValidatorAddress),
				)
			}
		case *stakingtypes.MsgDelegate:
			if !d.whitelistKeeper.CanValidatorReceiveDelegation(ctx, m.ValidatorAddress) {
				// Return with zero gas consumed for failed transactions
				zeroGasCtx := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
				return zeroGasCtx, errorsmod.Wrap(sdkerrors.ErrUnauthorized, 
					fmt.Sprintf("validator %s is not whitelisted for consensus", m.ValidatorAddress))
			}
		case *stakingtypes.MsgBeginRedelegate:
			if !d.whitelistKeeper.CanValidatorReceiveDelegation(ctx, m.ValidatorDstAddress) {
				// Return with zero gas consumed for failed transactions
				zeroGasCtx := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
				return zeroGasCtx, errorsmod.Wrap(sdkerrors.ErrUnauthorized, 
					fmt.Sprintf("destination validator %s is not whitelisted for consensus", m.ValidatorDstAddress))
			}
		}
	}

	return next(ctx, tx, simulate)
}