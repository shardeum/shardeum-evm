package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// HasDynamicFeeExtensionOption returns true if the tx implements the `ExtensionOptionDynamicFeeTx` extension option.
func HasDynamicFeeExtensionOption(anyType *codectypes.Any) bool {
	_, ok := anyType.GetCachedValue().(*ExtensionOptionDynamicFeeTx)
	return ok
}

// HasWeb3ExtensionOption returns true if the tx implements the `ExtensionOptionsWeb3Tx` extension option.
func HasWeb3ExtensionOption(anyType *codectypes.Any) bool {
	_, ok := anyType.GetCachedValue().(*ExtensionOptionsWeb3Tx)
	return ok
}

// HasDynamicFeeOrWeb3ExtensionOption returns true if the tx implements either the `ExtensionOptionDynamicFeeTx` or `ExtensionOptionsWeb3Tx` extension option.
func HasDynamicFeeOrWeb3ExtensionOption(anyType *codectypes.Any) bool {
	return HasDynamicFeeExtensionOption(anyType) || HasWeb3ExtensionOption(anyType)
}
