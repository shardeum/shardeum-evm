package main

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

func main() {
	fmt.Println("Testing ChainID Format Differences\n")
	fmt.Println("=" + string(make([]byte, 60)) + "=")

	// Test what HexOrDecimal256 produces
	chainID := uint64(8117)
	hexOrDec := math.NewHexOrDecimal256(int64(chainID))

	// Marshal to JSON
	jsonBytes, _ := json.Marshal(hexOrDec)
	fmt.Printf("\n1. HexOrDecimal256 marshals to JSON as: %s\n", string(jsonBytes))

	// Create domain with HexOrDecimal256
	domainHexOrDec := apitypes.TypedDataDomain{
		Name:              "Cosmos Web3",
		Version:           "1.0.0",
		ChainId:           hexOrDec,
		VerifyingContract: "cosmos",
		Salt:              "0",
	}

	domainHexOrDecJSON, _ := json.MarshalIndent(domainHexOrDec, "", "  ")
	fmt.Printf("\n2. Domain with HexOrDecimal256:\n%s\n", string(domainHexOrDecJSON))

	// Test with decimal string in map
	domainDecimal := map[string]interface{}{
		"name":              "Cosmos Web3",
		"version":           "1.0.0",
		"chainId":           "8117",
		"verifyingContract": "cosmos",
		"salt":              "0",
	}
	decimalJSON, _ := json.MarshalIndent(domainDecimal, "", "  ")
	fmt.Printf("\n3. Domain with decimal string \"8117\":\n%s\n", string(decimalJSON))

	// Test with number in map
	domainNumber := map[string]interface{}{
		"name":              "Cosmos Web3",
		"version":           "1.0.0",
		"chainId":           8117,
		"verifyingContract": "cosmos",
		"salt":              "0",
	}
	numberJSON, _ := json.MarshalIndent(domainNumber, "", "  ")
	fmt.Printf("\n4. Domain with number 8117:\n%s\n", string(numberJSON))

	// Test with hex string in map
	domainHex := map[string]interface{}{
		"name":              "Cosmos Web3",
		"version":           "1.0.0",
		"chainId":           "0x1fb5",
		"verifyingContract": "cosmos",
		"salt":              "0",
	}
	hexJSON, _ := json.MarshalIndent(domainHex, "", "  ")
	fmt.Printf("\n5. Domain with hex string \"0x1fb5\":\n%s\n", string(hexJSON))

	// Now test what happens when we hash these
	fmt.Printf("\n" + string(make([]byte, 60)) + "\n")
	fmt.Println("Hash Testing (if available):")
	fmt.Println("Note: Full hash testing would require creating complete TypedData structures")

	fmt.Printf("\nConclusion:\n")
	fmt.Printf("- The 'legacy' handler uses HexOrDecimal256 which likely marshals as: %s\n", string(jsonBytes))
	fmt.Printf("- Your web page sends: \"8117\" (decimal string)\n")
	fmt.Printf("- To match the non-legacy handler, you may need to send: %s\n", string(jsonBytes))
}
