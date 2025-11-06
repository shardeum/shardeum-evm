//go:build ignore

package main

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	shardeumeip712 "github.com/shardeum/shardeum-evm/ethereum/eip712"
)

type signedTx struct {
	TypedData       apitypes.TypedData `json:"typedData"`
	Signature       string             `json:"signature"`
	EthAddress      string             `json:"ethAddress"`
	Bech32Address   string             `json:"bech32Address"`
	ValidatorAddr   string             `json:"validatorAddress"`
	AccountNumber   string             `json:"accountNumber"`
	Sequence        string             `json:"sequence"`
	FeePayerAddress string             `json:"feePayer"`
}

type checkResult struct {
	name   string
	ok     bool
	detail string
}

func addCheck(results *[]checkResult, name string, ok bool, detailFmt string, args ...interface{}) {
	detail := detailFmt
	if len(args) > 0 {
		detail = fmt.Sprintf(detailFmt, args...)
	}
	*results = append(*results, checkResult{name: name, ok: ok, detail: detail})
}

func printSummary(results []checkResult) {
	if len(results) == 0 {
		return
	}

	failed := 0
	for _, r := range results {
		if !r.ok {
			failed++
		}
	}

	fmt.Printf("\n=== Check Summary ===\n")
	fmt.Printf("Total: %d, Passed: %d, Failed: %d\n", len(results), len(results)-failed, failed)
	for _, r := range results {
		status := "PASS"
		if !r.ok {
			status = "FAIL"
		}
		if r.detail != "" {
			fmt.Printf("[%s] %s - %s\n", status, r.name, r.detail)
		} else {
			fmt.Printf("[%s] %s\n", status, r.name)
		}
	}
	fmt.Println("=====================")
}

func main() {
	jsonPath := flag.String("signed", "signed-tx1.json", "path to JSON file with typedData and signature")
	showTyped := flag.Bool("typed", false, "print the reconstructed typed data JSON")
	flag.Parse()

	if err := run(*jsonPath, *showTyped); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(path string, showTyped bool) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	var tx signedTx
	if err := json.Unmarshal(raw, &tx); err != nil {
		return fmt.Errorf("unmarshal %s: %w", path, err)
	}

	sigBytes, err := hexutil.Decode(tx.Signature)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	var checks []checkResult
	defer func() {
		printSummary(checks)
	}()

	addCheck(&checks, "Signature length is 65 bytes", len(sigBytes) == ethcrypto.SignatureLength, "got %d bytes", len(sigBytes))
	if len(sigBytes) != ethcrypto.SignatureLength {
		return fmt.Errorf("signature length %d != %d", len(sigBytes), ethcrypto.SignatureLength)
	}

	hash, typedJSON, err := shardeumeip712.TypedDataAndHashWithDecimalChainID(tx.TypedData)
	if err != nil {
		return fmt.Errorf("hash typed data: %w", err)
	}

	standardDomainHash, standardMsgHash, standardFinal, err := computeStandardHashes(tx.TypedData)
	if err != nil {
		return fmt.Errorf("standard hashes: %w", err)
	}

	domainMapDecimal := decimalDomainMap(tx.TypedData)
	decimalDomainHash, err := tx.TypedData.HashStruct("EIP712Domain", domainMapDecimal)
	if err != nil {
		return fmt.Errorf("decimal domain hash: %w", err)
	}

	if chainID, ok := domainMapDecimal["chainId"].(string); ok {
		addCheck(&checks, "Domain chainId decimal string present", chainID != "", "%s", chainID)
	} else {
		addCheck(&checks, "Domain chainId decimal string present", false, "missing")
	}

	fmt.Printf("Loaded %s\n", path)
	fmt.Printf("Standard domain hash: %x\n", standardDomainHash)
	fmt.Printf("Decimal domain hash:  %x\n", decimalDomainHash)
	fmt.Printf("Message hash:        %x\n", standardMsgHash)
	fmt.Printf("Standard final hash: %x\n", standardFinal)
	fmt.Printf("Custom final hash:   %x\n", hash)

	hashesMatch := bytesEqual(hash, standardFinal)
	addCheck(&checks, "Custom and standard final hashes match", hashesMatch, "custom=%x standard=%x", hash, standardFinal)

	if showTyped {
		fmt.Printf("\nTyped data with decimal chainId:\n%s\n\n", typedJSON)
	}

	fmt.Printf("Raw signature (R|S|V): %x\n", sigBytes)
	fmt.Printf("Original V: %d\n", sigBytes[ethcrypto.RecoveryIDOffset])

	if sigBytes[ethcrypto.RecoveryIDOffset] == 27 || sigBytes[ethcrypto.RecoveryIDOffset] == 28 {
		sigBytes[ethcrypto.RecoveryIDOffset] -= 27
	}

	fmt.Printf("Adjusted V: %d\n", sigBytes[ethcrypto.RecoveryIDOffset])
	addCheck(&checks, "Recovery id is 0 or 1", sigBytes[ethcrypto.RecoveryIDOffset] == 0 || sigBytes[ethcrypto.RecoveryIDOffset] == 1, "v=%d", sigBytes[ethcrypto.RecoveryIDOffset])
	fmt.Printf("R: %x\n", sigBytes[:32])
	fmt.Printf("S: %x\n", sigBytes[32:64])

	r := new(big.Int).SetBytes(sigBytes[:32])
	s := new(big.Int).SetBytes(sigBytes[32:64])
	curveOrder := ethcrypto.S256().Params().N
	halfOrder := new(big.Int).Rsh(curveOrder, 1)

	addCheck(&checks, "R is non-zero", r.Sign() != 0, "r=%s", r.Text(16))
	addCheck(&checks, "S is non-zero", s.Sign() != 0, "s=%s", s.Text(16))
	addCheck(&checks, "R < curve order", r.Cmp(curveOrder) < 0, "r=%s", r.Text(16))
	addCheck(&checks, "S < curve order", s.Cmp(curveOrder) < 0, "s=%s", s.Text(16))
	addCheck(&checks, "S in lower half", s.Cmp(halfOrder) <= 0, "s=%s", s.Text(16))

	fmt.Printf("r >= N? %v\n", r.Cmp(curveOrder) >= 0)
	fmt.Printf("s >= N? %v\n", s.Cmp(curveOrder) >= 0)
	fmt.Printf("s > N/2? %v\n", s.Cmp(halfOrder) > 0)

	pubBytes, err := secp256k1.RecoverPubkey(hash, sigBytes)
	if err != nil {
		return fmt.Errorf("recover pubkey: %w", err)
	}
	validPub := len(pubBytes) == 65 && pubBytes[0] == 0x04
	addCheck(&checks, "Recovered pubkey has uncompressed format", validPub, "len=%d prefix=0x%x", len(pubBytes), func() byte {
		if len(pubBytes) > 0 {
			return pubBytes[0]
		}
		return 0
	}())
	if !validPub {
		return errors.New("unexpected recovered pubkey format")
	}

	fmt.Printf("Recovered pubkey: %x\n", pubBytes)

	signatureRS := sigBytes[:len(sigBytes)-1]

	pubKey, err := ethcrypto.UnmarshalPubkey(pubBytes)
	if err != nil {
		return fmt.Errorf("unmarshal pubkey: %w", err)
	}

	compressedPub := ethcrypto.CompressPubkey(pubKey)

	secpCompressedOK := secp256k1.VerifySignature(compressedPub, hash, signatureRS)
	secpUncompressedOK := secp256k1.VerifySignature(pubBytes, hash, signatureRS)
	cryptoOK := ethcrypto.VerifySignature(compressedPub, hash, signatureRS)

	ecdsaOK := ecdsa.Verify(pubKey, hash, r, s)

	fmt.Printf("secp256k1.VerifySignature (compressed pubkey): %v\n", secpCompressedOK)
	fmt.Printf("secp256k1.VerifySignature (uncompressed pubkey): %v\n", secpUncompressedOK)
	fmt.Printf("crypto.VerifySignature (compressed pubkey):   %v\n", cryptoOK)
	fmt.Printf("ecdsa.Verify:             %v\n", ecdsaOK)
	addCheck(&checks, "ecdsa.Verify passes", ecdsaOK, "")
	addCheck(&checks, "secp256k1.VerifySignature (compressed) passes", secpCompressedOK, "")
	addCheck(&checks, "secp256k1.VerifySignature (uncompressed) passes", secpUncompressedOK, "")
	addCheck(&checks, "crypto.VerifySignature passes", cryptoOK, "")

	standardSecp := secp256k1.VerifySignature(compressedPub, standardFinal, signatureRS)
	addCheck(&checks, "Standard hash verification matches custom result", standardSecp == hashesMatch, "standardSecp=%v hashesMatch=%v", standardSecp, hashesMatch)

	ethAddr := ethcrypto.PubkeyToAddress(*pubKey)
	fmt.Printf("Recovered Ethereum address: %s\n", ethAddr.Hex())
	if tx.EthAddress != "" {
		fmt.Printf("Signed Ethereum address:    %s\n", tx.EthAddress)
	}

	cosmosAddr, cosmosErr := bech32.ConvertAndEncode("cosmos", ethAddr.Bytes())
	addCheck(&checks, "Bech32 cosmos encoding", cosmosErr == nil, "error=%v", cosmosErr)
	if cosmosErr == nil {
		fmt.Printf("Recovered Cosmos address:   %s\n", cosmosAddr)
	}

	shardeumAddr, shardeumErr := bech32.ConvertAndEncode("shardeum", ethAddr.Bytes())
	addCheck(&checks, "Bech32 shardeum encoding", shardeumErr == nil, "error=%v", shardeumErr)
	if shardeumErr == nil {
		fmt.Printf("Recovered Shardeum address: %s\n", shardeumAddr)
		if tx.Bech32Address != "" {
			matches := strings.EqualFold(shardeumAddr, tx.Bech32Address)
			addCheck(&checks, "Recovered shardeum address matches signed", matches, "recovered=%s signed=%s", shardeumAddr, tx.Bech32Address)
		}
	}

	if tx.EthAddress != "" {
		addCheck(&checks, "Recovered Ethereum address matches signed", strings.EqualFold(ethAddr.Hex(), tx.EthAddress), "recovered=%s signed=%s", ethAddr.Hex(), tx.EthAddress)
	}

	if tx.ValidatorAddr != "" {
		addDelegationChecks(&checks, tx.TypedData, tx.ValidatorAddr, tx.Bech32Address)
	} else {
		addDelegationChecks(&checks, tx.TypedData, "", tx.Bech32Address)
	}

	return nil
}

func computeStandardHashes(td apitypes.TypedData) ([]byte, []byte, []byte, error) {
	domainMap := td.Domain.Map()

	domainHash, err := td.HashStruct("EIP712Domain", domainMap)
	if err != nil {
		return nil, nil, nil, err
	}

	msgHash, err := td.HashStruct(td.PrimaryType, td.Message)
	if err != nil {
		return nil, nil, nil, err
	}

	raw := append([]byte{0x19, 0x01}, append(domainHash, msgHash...)...)
	finalHash := ethcrypto.Keccak256(raw)

	return domainHash, msgHash, finalHash, nil
}

func decimalDomainMap(td apitypes.TypedData) map[string]interface{} {
	var chainID string
	if td.Domain.ChainId != nil {
		chainID = (*big.Int)(td.Domain.ChainId).String()
	}

	return map[string]interface{}{
		"name":              td.Domain.Name,
		"version":           td.Domain.Version,
		"chainId":           chainID,
		"verifyingContract": td.Domain.VerifyingContract,
		"salt":              td.Domain.Salt,
	}
}

func addDelegationChecks(checks *[]checkResult, td apitypes.TypedData, expectedValidator, expectedBech32 string) {
	message := map[string]interface{}(td.Message)
	ok := message != nil
	addCheck(checks, "Typed data message is object", ok, "")
	if !ok {
		return
	}

	if chainID, ok := stringFromInterface(message["chain_id"]); ok {
		addCheck(checks, "Message chain_id present", true, chainID)
	} else {
		addCheck(checks, "Message chain_id present", false, "missing")
	}

	if account, ok := stringFromInterface(message["account_number"]); ok {
		addCheck(checks, "Message account_number present", true, account)
	} else {
		addCheck(checks, "Message account_number present", false, "missing")
	}

	if sequence, ok := stringFromInterface(message["sequence"]); ok {
		addCheck(checks, "Message sequence present", true, sequence)
	} else {
		addCheck(checks, "Message sequence present", false, "missing")
	}

	fee, ok := mapFromInterface(message["fee"])
	addCheck(checks, "Fee object present", ok, "")
	if ok {
		if feePayer, ok := stringFromInterface(fee["feePayer"]); ok {
			addCheck(checks, "Fee feePayer present", true, feePayer)
			if expectedBech32 != "" {
				addCheck(checks, "Fee feePayer matches expected", strings.EqualFold(feePayer, expectedBech32), "feePayer=%s expected=%s", feePayer, expectedBech32)
			}
		} else {
			addCheck(checks, "Fee feePayer present", false, "missing")
		}

		if gas, ok := stringFromInterface(fee["gas"]); ok {
			addCheck(checks, "Fee gas present", true, gas)
		} else {
			addCheck(checks, "Fee gas present", false, "missing")
		}

		if amounts, ok := sliceFromInterface(fee["amount"]); ok && len(amounts) > 0 {
			if amountMap, ok := mapFromInterface(amounts[0]); ok {
				if denom, ok := stringFromInterface(amountMap["denom"]); ok {
					addCheck(checks, "Fee amount denom present", true, denom)
				} else {
					addCheck(checks, "Fee amount denom present", false, "missing")
				}
				if amt, ok := stringFromInterface(amountMap["amount"]); ok {
					addCheck(checks, "Fee amount value present", true, amt)
				} else {
					addCheck(checks, "Fee amount value present", false, "missing")
				}
			} else {
				addCheck(checks, "Fee amount entry is object", false, fmt.Sprintf("type=%T", amounts[0]))
			}
		} else {
			addCheck(checks, "Fee amount list present", false, "missing or empty")
		}
	}

	msgs, ok := sliceFromInterface(message["msgs"])
	addCheck(checks, "Msgs array present", ok && len(msgs) > 0, "len=%d", func() int {
		if !ok {
			return 0
		}
		return len(msgs)
	}())
	if !ok || len(msgs) == 0 {
		return
	}

	firstMsg, ok := mapFromInterface(msgs[0])
	addCheck(checks, "First msg is object", ok, fmt.Sprintf("type=%T", msgs[0]))
	if !ok {
		return
	}

	if msgType, ok := stringFromInterface(firstMsg["type"]); ok {
		addCheck(checks, "Msg type present", true, msgType)
	} else {
		addCheck(checks, "Msg type present", false, "missing")
	}

	valueMap, ok := mapFromInterface(firstMsg["value"])
	addCheck(checks, "Msg value object present", ok, "")
	if !ok {
		return
	}

	if delegator, ok := stringFromInterface(valueMap["delegator_address"]); ok {
		addCheck(checks, "Delegator address present", true, delegator)
		if expectedBech32 != "" {
			addCheck(checks, "Delegator matches expected", strings.EqualFold(delegator, expectedBech32), "delegator=%s expected=%s", delegator, expectedBech32)
		}
	} else {
		addCheck(checks, "Delegator address present", false, "missing")
	}

	if validator, ok := stringFromInterface(valueMap["validator_address"]); ok {
		addCheck(checks, "Validator address present", true, validator)
		if expectedValidator != "" {
			addCheck(checks, "Validator matches expected", strings.EqualFold(validator, expectedValidator), "validator=%s expected=%s", validator, expectedValidator)
		}
	} else {
		addCheck(checks, "Validator address present", false, "missing")
	}

	amountMap, ok := mapFromInterface(valueMap["amount"])
	addCheck(checks, "Msg amount object present", ok, "")
	if ok {
		if denom, ok := stringFromInterface(amountMap["denom"]); ok {
			addCheck(checks, "Msg amount denom present", true, denom)
		} else {
			addCheck(checks, "Msg amount denom present", false, "missing")
		}
		if amt, ok := stringFromInterface(amountMap["amount"]); ok {
			addCheck(checks, "Msg amount value present", true, amt)
		} else {
			addCheck(checks, "Msg amount value present", false, "missing")
		}
	}
}

func mapFromInterface(value interface{}) (map[string]interface{}, bool) {
	switch v := value.(type) {
	case map[string]interface{}:
		return v, true
	default:
		return nil, false
	}
}

func sliceFromInterface(value interface{}) ([]interface{}, bool) {
	switch v := value.(type) {
	case []interface{}:
		return v, true
	default:
		return nil, false
	}
}

func stringFromInterface(value interface{}) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case json.Number:
		return v.String(), true
	case fmt.Stringer:
		return v.String(), true
	case float64:
		return fmt.Sprintf("%.0f", v), true
	case int:
		return fmt.Sprintf("%d", v), true
	case int64:
		return fmt.Sprintf("%d", v), true
	case uint64:
		return fmt.Sprintf("%d", v), true
	case bool:
		if v {
			return "true", true
		}
		return "false", true
	default:
		return "", false
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
