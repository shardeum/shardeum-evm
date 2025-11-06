package eip712

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// TypedDataAndHashWithDecimalChainID calculates EIP-712 hash using decimal chainId format.
// This matches MetaMask's expectation where chainId is a decimal string like "8117" instead of "0x1fb5".
func TypedDataAndHashWithDecimalChainID(typedData apitypes.TypedData) ([]byte, string, error) {
	// Convert chainId to decimal string for the domain
	var chainIdStr string
	if typedData.Domain.ChainId != nil {
		chainIdBigInt := (*big.Int)(typedData.Domain.ChainId)
		chainIdStr = chainIdBigInt.String() // Decimal representation
	}

	// Create domain map with decimal chainId
	domainMap := map[string]interface{}{
		"name":              typedData.Domain.Name,
		"version":           typedData.Domain.Version,
		"chainId":           chainIdStr, // Use decimal string instead of HexOrDecimal256
		"verifyingContract": typedData.Domain.VerifyingContract,
		"salt":              typedData.Domain.Salt,
	}

	// Hash the domain using EIP-712 encoding
	domainSeparator, err := typedData.HashStruct("EIP712Domain", domainMap)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash domain: %w", err)
	}

	// Hash the primary message
	messageHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash message: %w", err)
	}

	// Calculate final EIP-712 hash: keccak256("\x19\x01" || domainSeparator || messageHash)
	rawData := append([]byte{0x19, 0x01}, append(domainSeparator, messageHash...)...)
	hash := crypto.Keccak256(rawData)

	// Create JSON representation with decimal chainId for debugging
	customData := map[string]interface{}{
		"types":       typedData.Types,
		"primaryType": typedData.PrimaryType,
		"domain":      domainMap,
		"message":     typedData.Message,
	}
	jsonData, err := json.Marshal(customData)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal typed data: %w", err)
	}

	return hash, string(jsonData), nil
}


