package cosmos

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
)

// Eip712SigVerifyWrapper wraps the standard Cosmos SigVerificationDecorator
// and skips it if EIP-712 signature was already verified
type Eip712SigVerifyWrapper struct {
	inner ante.SigVerificationDecorator
}

func NewEip712SigVerifyWrapper(inner ante.SigVerificationDecorator) Eip712SigVerifyWrapper {
	return Eip712SigVerifyWrapper{inner: inner}
}

func (svw Eip712SigVerifyWrapper) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// Check if EIP-712 signature was already verified
	if verified, ok := ctx.Value("eip712-verified").(bool); ok && verified {
		// Skip standard signature verification for EIP-712 txs
		return next(ctx, tx, simulate)
	}
	
	// Otherwise, run standard Cosmos signature verification
	return svw.inner.AnteHandle(ctx, tx, simulate, next)
}
