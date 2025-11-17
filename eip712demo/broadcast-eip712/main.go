package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	cryptocodec "github.com/shardeum/shardeum-evm/crypto/codec"
	"github.com/shardeum/shardeum-evm/crypto/ethsecp256k1"
	"github.com/shardeum/shardeum-evm/eip712demo/common"
	"github.com/shardeum/shardeum-evm/ethereum/eip712"
	ethermint "github.com/shardeum/shardeum-evm/types"
	"github.com/spf13/cobra"
)

// evmCodec is the codec used for EIP-712 operations, matching the chain's codec
var evmCodec codec.ProtoCodecMarshaler

func init() {
	// Set SDK config for Shardeum
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("shardeum", "shardeum"+sdk.PrefixPublic)
	config.SetBech32PrefixForValidator("shardeum"+sdk.PrefixValidator+sdk.PrefixOperator, "shardeum"+sdk.PrefixValidator+sdk.PrefixOperator+sdk.PrefixPublic)
	config.SetBech32PrefixForConsensusNode("shardeum"+sdk.PrefixValidator+sdk.PrefixConsensus, "shardeum"+sdk.PrefixValidator+sdk.PrefixConsensus+sdk.PrefixPublic)
	config.Seal()
	
	// Initialize the codec with all necessary interfaces
	// This matches what the chain does in ante/cosmos/eip712.go
	registry := codectypes.NewInterfaceRegistry()
	ethermint.RegisterInterfaces(registry)
	stakingtypes.RegisterInterfaces(registry)
	cryptocodec.RegisterInterfaces(registry)
	evmCodec = codec.NewProtoCodec(registry)
	
	// Set the amino codec for legacy StdSignBytes
	aminoCodec := codec.NewLegacyAmino()
	stakingtypes.RegisterLegacyAminoCodec(aminoCodec)
	cryptocodec.RegisterCrypto(aminoCodec)
	legacytx.RegressionTestingAminoCodec = aminoCodec
}

// SignedEIP712Tx represents the JSON structure from MetaMask
type SignedEIP712Tx struct {
	TypedData        apitypes.TypedData `json:"typedData"`
	Signature        string             `json:"signature"`
	EthAddress       string             `json:"ethAddress"`
	Bech32Address    string             `json:"bech32Address"`
	ValidatorAddress string             `json:"validatorAddress"`
	Amount           string             `json:"amount"`
	Denom            string             `json:"denom"`
}

// Helper to extract message data from TypedData
func (tx *SignedEIP712Tx) GetChainID() string {
	if chainID, ok := tx.TypedData.Message["chain_id"].(string); ok {
		return chainID
	}
	return ""
}

func (tx *SignedEIP712Tx) GetMemo() string {
	if memo, ok := tx.TypedData.Message["memo"].(string); ok {
		return memo
	}
	return ""
}

func (tx *SignedEIP712Tx) GetSequence() string {
	if seq, ok := tx.TypedData.Message["sequence"].(string); ok {
		return seq
	}
	return ""
}

func (tx *SignedEIP712Tx) GetAccountNumber() string {
	if accNum, ok := tx.TypedData.Message["account_number"].(string); ok {
		return accNum
	}
	return "0"
}

func (tx *SignedEIP712Tx) GetFee() ([]struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}, string) {
	fee := tx.TypedData.Message["fee"]
	if feeMap, ok := fee.(map[string]interface{}); ok {
		var feeAmount []struct {
			Denom  string `json:"denom"`
			Amount string `json:"amount"`
		}
		
		if amountList, ok := feeMap["amount"].([]interface{}); ok {
			for _, amt := range amountList {
				if amtMap, ok := amt.(map[string]interface{}); ok {
					feeAmount = append(feeAmount, struct {
						Denom  string `json:"denom"`
						Amount string `json:"amount"`
					}{
						Denom:  amtMap["denom"].(string),
						Amount: amtMap["amount"].(string),
					})
				}
			}
		}
		
		gas := ""
		if gasVal, ok := feeMap["gas"].(string); ok {
			gas = gasVal
		}
		
		return feeAmount, gas
	}
	return nil, ""
}

func (tx *SignedEIP712Tx) GetMsgs() []json.RawMessage {
	var result []json.RawMessage

	// First try the old format with msgs array (for backward compatibility)
	if msgs, ok := tx.TypedData.Message["msgs"].([]interface{}); ok {
		for _, msg := range msgs {
			if msgBytes, err := json.Marshal(msg); err == nil {
				result = append(result, msgBytes)
			}
		}
		return result
	}

	// New format: messages are individual fields msg0, msg1, etc.
	for i := 0; ; i++ {
		msgKey := fmt.Sprintf("msg%d", i)
		if msg, ok := tx.TypedData.Message[msgKey]; ok {
			if msgBytes, err := json.Marshal(msg); err == nil {
				result = append(result, msgBytes)
			}
		} else {
			// No more messages
			break
		}
	}

	return result
}

func main() {
	var nodeURL string
	var broadcast bool
	var evmChainID uint64
	var cosmosChainID string

	rootCmd := &cobra.Command{
		Use:   "broadcast-eip712 [signed-tx.json]",
		Short: "Parse, display, and optionally broadcast EIP-712 signed transactions",
		Long: `Parse an EIP-712 signed transaction from MetaMask and display its contents.
Can also broadcast the transaction to the network with --broadcast flag.

⚠️  IMPORTANT: The signed transaction must match the target chain!
If the chain IDs don't match, you need to re-sign with the correct chain ID.

Example:
  # Display only
  broadcast-eip712 signed-tx1.json

  # Broadcast to local node (chain ID: shardeum_8117-1)
  broadcast-eip712 signed-tx1.json --broadcast --node http://localhost:1317 --cosmos-chain-id shardeum_8117-1

  # Broadcast to mainnet (chain ID: shardeum_8118-1)
  broadcast-eip712 signed-tx1.json --broadcast --node https://gcp.rpc.shardeum.org --evm-chain-id 8118 --cosmos-chain-id shardeum_8118-1
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if broadcast {
				return broadcastTx(args[0], nodeURL, evmChainID, cosmosChainID)
			}
			return displayTx(args[0])
		},
	}

	rootCmd.Flags().BoolVar(&broadcast, "broadcast", false, "Broadcast the transaction to the network")
	rootCmd.Flags().StringVar(&nodeURL, "node", "http://localhost:1317", "Node REST API URL")
	rootCmd.Flags().Uint64Var(&evmChainID, "evm-chain-id", 8117, "EVM chain ID (8117 for local, 8118 for mainnet, 8119 for testnet)")
	rootCmd.Flags().StringVar(&cosmosChainID, "cosmos-chain-id", "shardeum_8117-1", "Cosmos chain ID (must match signed transaction)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func displayTx(filePath string) error {
	// Read the signed transaction file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var signedTx SignedEIP712Tx
	if err := json.Unmarshal(data, &signedTx); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Display transaction info
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("📋 EIP-712 Signed Transaction Details")
	fmt.Println("=" + strings.Repeat("=", 79))

	fmt.Printf("\n🔑 Signer:\n")
	fmt.Printf("   Ethereum Address: %s\n", signedTx.EthAddress)
	fmt.Printf("   Bech32 Address:   %s\n", signedTx.Bech32Address)

	fmt.Printf("\n💰 Transaction Details:\n")

	// Parse messages
	msgs, err := common.BuildMessages(&signedTx)
	if err != nil {
		fmt.Printf("   ⚠️  Failed to parse messages: %v\n", err)
	} else {
		fmt.Printf("   Messages: %d\n", len(msgs))
		for i, msg := range msgs {
			fmt.Printf("   %d. %s\n", i+1, sdk.MsgTypeURL(msg))
			// Display message details
			switch m := msg.(type) {
			case *stakingtypes.MsgDelegate:
				fmt.Printf("      Delegator: %s\n", m.DelegatorAddress)
				fmt.Printf("      Validator: %s\n", m.ValidatorAddress)
				fmt.Printf("      Amount:    %s\n", m.Amount.String())
			case *stakingtypes.MsgUndelegate:
				fmt.Printf("      Delegator: %s\n", m.DelegatorAddress)
				fmt.Printf("      Validator: %s\n", m.ValidatorAddress)
				fmt.Printf("      Amount:    %s\n", m.Amount.String())
			}
		}
	}

	fmt.Printf("\n⛽ Fee & Gas:\n")
	feeAmount, gas := signedTx.GetFee()
	fmt.Printf("   Gas Limit: %s\n", gas)
	if len(feeAmount) > 0 {
		fmt.Printf("   Fee:       %s %s\n",
			feeAmount[0].Amount,
			feeAmount[0].Denom)
	}

	fmt.Printf("\n📝 Memo: %s\n", signedTx.GetMemo())
	fmt.Printf("🌐 Chain ID: %s\n", signedTx.GetChainID())
	fmt.Printf("🔢 Sequence: %s\n", signedTx.GetSequence())

	// Parse signature
	sig, err := parseSignature(signedTx.Signature)
	if err != nil {
		fmt.Printf("\n❌ Failed to parse signature: %v\n", err)
	} else {
		fmt.Printf("\n🔐 Signature:\n")
		fmt.Printf("   Full: %s\n", signedTx.Signature)
		fmt.Printf("   R: %x\n", sig[:32])
		fmt.Printf("   S: %x\n", sig[32:64])
		fmt.Printf("   V: %d (adjusted from %d)\n", sig[64], sig[64]+27)
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("  please use --broadcast to send this transaction to the network")
	fmt.Println(strings.Repeat("=", 80))

	return nil
}

func parseSignature(sigHex string) ([]byte, error) {
	// Remove 0x prefix
	sigHex = strings.TrimPrefix(sigHex, "0x")

	// Decode hex
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return nil, fmt.Errorf("invalid hex signature: %w", err)
	}

	if len(sig) != 65 {
		return nil, fmt.Errorf("invalid signature length: %d, expected 65", len(sig))
	}

	// Adjust V value for Cosmos (Ethereum uses 27/28, we need 0/1)
	if sig[64] >= 27 {
		sig[64] -= 27
	}

	return sig, nil
}


// broadcastTx broadcasts the signed EIP-712 transaction to the network
func broadcastTx(filePath, nodeURL string, evmChainID uint64, cosmosChainID string) error {
	// Read and parse the signed transaction
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var signedTx SignedEIP712Tx
	if err := json.Unmarshal(data, &signedTx); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	fmt.Println("🚀 Broadcasting EIP-712 Transaction")
	fmt.Println(strings.Repeat("=", 80))

	// Validate chain ID match
	signedChainID := signedTx.GetChainID()
	if signedChainID != cosmosChainID {
		fmt.Printf("\n⚠️  WARNING: Chain ID Mismatch!\n")
		fmt.Printf("   Signed TX Chain ID:  %s\n", signedChainID)
		fmt.Printf("   Target Chain ID:     %s\n", cosmosChainID)
		fmt.Printf("\n   The transaction was signed for a different chain!\n")
		fmt.Printf("   You need to re-sign the transaction with the correct chain ID.\n")
		fmt.Printf("   Update CONFIG.cosmosChainId in eip712-helpers.js to '%s'\n\n", cosmosChainID)
		return fmt.Errorf("chain ID mismatch: signed=%s, target=%s", 
			signedChainID, cosmosChainID)
	}

	// Step 1: Parse signature
	fmt.Println("\n📝 Step 1: Parsing signature...")
	sig, err := parseSignature(signedTx.Signature)
	if err != nil {
		return fmt.Errorf("failed to parse signature: %w", err)
	}
	fmt.Printf("   ✅ Signature parsed: %d bytes\n", len(sig))

	// Step 2: Query account info (to get account number, sequence, and try to get public key)
	fmt.Println("\n📡 Step 2: Querying account info from chain...")
	accNumber, sequence, err := queryAccount(nodeURL, signedTx.Bech32Address)
	if err != nil {
		return fmt.Errorf("failed to query account: %w", err)
	}
	fmt.Printf("   ✅ Account number: %d, Sequence: %d\n", accNumber, sequence)
	
	// Step 2b: Recover pubkey from EIP-712 signature  
	// This MUST match what the ante handler will recover
	fmt.Println("\n🔑 Step 2b: Recovering public key from signature...")
	
	// Compute the hash of the TypedData
	sigHash, _, err := apitypes.TypedDataAndHash(signedTx.TypedData)
	if err != nil {
		return fmt.Errorf("failed to hash typed data: %w", err)
	}
	
	// Recover public key from signature
	pubKeyBytes, err := crypto.Ecrecover(sigHash, sig)
	if err != nil {
		return fmt.Errorf("failed to recover public key: %w", err)
	}
	
	// Convert to compressed format
	ecdsaPubKey, err := crypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to unmarshal public key: %w", err)
	}
	compressedPubKey := crypto.CompressPubkey(ecdsaPubKey)
	pubKey := &ethsecp256k1.PubKey{Key: compressedPubKey}
	
	// Verify it matches the expected address
	recoveredAddr := sdk.AccAddress(pubKey.Address())
	fmt.Printf("   🔍 Recovered: %s\n", recoveredAddr.String())
	fmt.Printf("   🔍 Expected: %s\n", signedTx.Bech32Address)
	
	if recoveredAddr.String() != signedTx.Bech32Address {
		return fmt.Errorf("recovered address %s does not match expected %s",
			recoveredAddr.String(), signedTx.Bech32Address)
	}
	
	fmt.Printf("   ✅ Public key recovered and verified\n")

	// Step 3: Build messages
	fmt.Println("\n🔨 Step 3: Building transaction messages...")
	msgs, err := common.BuildMessages(&signedTx)
	if err != nil {
		return fmt.Errorf("failed to build messages: %w", err)
	}
	fmt.Printf("   ✅ %d message(s) built\n", len(msgs))

	// Step 4: Construct Cosmos transaction with EIP-712 extension
	fmt.Println("\n🏗️  Step 4: Constructing Cosmos transaction...")
	txBytes, err := buildEIP712Tx(signedTx, msgs, pubKey, sig, accNumber, sequence)
	if err != nil {
		return fmt.Errorf("failed to build transaction: %w", err)
	}
	fmt.Printf("   ✅ Transaction encoded: %d bytes\n", len(txBytes))

	// Step 5: Broadcast transaction
	fmt.Println("\n📤 Step 5: Broadcasting to network...")
	txHash, err := broadcastRawTx(nodeURL, txBytes)
	if err != nil {
		return fmt.Errorf("failed to broadcast: %w", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("✅ Transaction Broadcast Successful!")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("\n🔗 Transaction Hash: %s\n", txHash)
	fmt.Printf("\n💡 Check status with:\n")
	fmt.Printf("   shardeumd query tx %s --node %s\n", txHash, nodeURL)

	return nil
}

// recoverPubKeyFromEIP712 recovers the public key from EIP-712 signature
func recoverPubKeyFromEIP712(signedTx SignedEIP712Tx, sig []byte, evmChainID uint64) (cryptotypes.PubKey, error) {
	// Build messages from the signed transaction
	msgs, err := common.BuildMessages(&signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to build messages: %w", err)
	}
	
	if len(msgs) == 0 {
		return nil, fmt.Errorf("no messages found in transaction")
	}
	
	// Get transaction parameters
	feeAmounts, gas := signedTx.GetFee()
	chainID := signedTx.GetChainID()
	sequence := signedTx.GetSequence()
	accountNumber := signedTx.GetAccountNumber()
	memo := signedTx.GetMemo()
	
	// Build the fee structure for Amino encoding
	feeCoins := make(sdk.Coins, len(feeAmounts))
	for i, fee := range feeAmounts {
		amount, ok := sdkmath.NewIntFromString(fee.Amount)
		if !ok {
			return nil, fmt.Errorf("invalid fee amount: %s", fee.Amount)
		}
		feeCoins[i] = sdk.NewCoin(fee.Denom, amount)
	}
	
	var gasLimit uint64
	fmt.Sscanf(gas, "%d", &gasLimit)
	
	var accNum, seq uint64
	fmt.Sscanf(accountNumber, "%d", &accNum)
	fmt.Sscanf(sequence, "%d", &seq)
	
	// Use legacytx.StdSignBytes to create proper Amino JSON
	aminoBytes := legacytx.StdSignBytes(chainID, accNum, seq, 0, 
		legacytx.NewStdFee(gasLimit, feeCoins), msgs, memo) //#nosec G115
	
	// Use the chain's EIP-712 function to build TypedData
	// Use evmCodec which was initialized with all necessary interfaces
	feeDelegation := &eip712.FeeDelegationOptions{
		FeePayer: sdk.MustAccAddressFromBech32(signedTx.Bech32Address),
	}
	
	typedData, err := eip712.LegacyWrapTxToTypedData(evmCodec, evmChainID, msgs[0], aminoBytes, feeDelegation)
	if err != nil {
		return nil, fmt.Errorf("failed to create TypedData: %w", err)
	}
	
	// Now compute the hash
	sigHash, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		return nil, fmt.Errorf("failed to hash typed data: %w", err)
	}

	// Recover the public key using Ethereum's crypto package
	// Note: sig is already adjusted (V is 0 or 1)
	pubKeyBytes, err := crypto.Ecrecover(sigHash, sig)
	if err != nil {
		return nil, fmt.Errorf("failed to recover public key: %w", err)
	}

	// Convert uncompressed public key (65 bytes) to compressed (33 bytes)
	// pubKeyBytes is [04 | X | Y], we need to compress it
	if len(pubKeyBytes) != 65 {
		return nil, fmt.Errorf("invalid public key length: %d", len(pubKeyBytes))
	}
	
	// Parse to ECDSA public key first
	ecdsaPubKey, err := crypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal public key: %w", err)
	}
	
	// Compress it
	compressedPubKey := crypto.CompressPubkey(ecdsaPubKey)
	pubKey := &secp256k1.PubKey{Key: compressedPubKey}

	// Verify the recovered address matches
	recoveredAddr := sdk.AccAddress(pubKey.Address())
	if recoveredAddr.String() != signedTx.Bech32Address {
		return nil, fmt.Errorf("recovered address %s does not match expected %s",
			recoveredAddr.String(), signedTx.Bech32Address)
	}

	return pubKey, nil
}

// queryAccount queries the account number and sequence from the chain
func queryAccount(nodeURL, address string) (uint64, uint64, error) {
	url := fmt.Sprintf("%s/cosmos/auth/v1beta1/accounts/%s", strings.TrimSuffix(nodeURL, "/"), address)

	resp, err := http.Get(url)
	if err != nil {
		return 0, 0, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, 0, fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Account struct {
			Type          string `json:"@type"`
			AccountNumber string `json:"account_number"`
			Sequence      string `json:"sequence"`
			BaseAccount   *struct {
				AccountNumber string `json:"account_number"`
				Sequence      string `json:"sequence"`
			} `json:"base_account,omitempty"`
		} `json:"account"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, 0, fmt.Errorf("failed to decode response: %w", err)
	}

	// Handle both base accounts and vesting accounts
	accNumStr := result.Account.AccountNumber
	seqStr := result.Account.Sequence
	if result.Account.BaseAccount != nil {
		accNumStr = result.Account.BaseAccount.AccountNumber
		seqStr = result.Account.BaseAccount.Sequence
	}

	var accNum, seq uint64
	fmt.Sscanf(accNumStr, "%d", &accNum)
	fmt.Sscanf(seqStr, "%d", &seq)

	return accNum, seq, nil
}

// buildEIP712Tx constructs a Cosmos SDK transaction with EIP-712 extension
func buildEIP712Tx(
	signedTx SignedEIP712Tx,
	msgs []sdk.Msg,
	pubKey cryptotypes.PubKey,
	signature []byte,
	accNumber, sequence uint64,
) ([]byte, error) {
	// Create codec
	registry := codectypes.NewInterfaceRegistry()
	stakingtypes.RegisterInterfaces(registry)
	authtypes.RegisterInterfaces(registry)
	ethermint.RegisterInterfaces(registry)
	cryptocodec.RegisterInterfaces(registry)
	
	// Register ethsecp256k1 public key type explicitly
	registry.RegisterImplementations(
		(*cryptotypes.PubKey)(nil),
		&ethsecp256k1.PubKey{},
	)
	
	protoCodec := codec.NewProtoCodec(registry)

	// Create tx config
	txConfig := authtx.NewTxConfig(protoCodec, authtx.DefaultSignModes)

	// Create tx builder with extension support
	txBuilder := txConfig.NewTxBuilder()
	builder, ok := txBuilder.(authtx.ExtensionOptionsTxBuilder)
	if !ok {
		return nil, fmt.Errorf("tx builder does not support extensions")
	}

	// Set messages
	if err := builder.SetMsgs(msgs...); err != nil {
		return nil, fmt.Errorf("failed to set messages: %w", err)
	}

	// Set fee
	feeAmounts, gas := signedTx.GetFee()
	sdkFeeAmount := make([]sdk.Coin, 0)
	for _, fee := range feeAmounts {
		amount, ok := sdkmath.NewIntFromString(fee.Amount)
		if !ok {
			return nil, fmt.Errorf("invalid fee amount: %s", fee.Amount)
		}
		sdkFeeAmount = append(sdkFeeAmount, sdk.NewCoin(fee.Denom, amount))
	}
	builder.SetFeeAmount(sdkFeeAmount)

	// Set gas limit
	var gasLimit uint64
	fmt.Sscanf(gas, "%d", &gasLimit)
	builder.SetGasLimit(gasLimit)

	// Set memo
	builder.SetMemo(signedTx.GetMemo())

	// NOTE: We do NOT set ExtensionOptionsWeb3Tx because:
	// 1. The aminojson sign mode handler rejects transactions with extensions
	// 2. The extension was only used for routing in the old legacy ante handler
	// 3. Our ethsecp256k1.PubKey.VerifySignature() handles EIP-712 automatically
	// 4. No extension = standard Cosmos ante handler which will call our VerifySignature()

	// Set the actual signature with SIGN_MODE_LEGACY_AMINO_JSON
	// The ethsecp256k1.PubKey.VerifySignature expects [R || S] format (64 bytes)
	// Remove the recovery ID (V) from the signature
	sigWithoutRecoveryID := signature[:64]

	sigData := &signingtypes.SingleSignatureData{
		SignMode:  signingtypes.SignMode_SIGN_MODE_LEGACY_AMINO_JSON,
		Signature: sigWithoutRecoveryID, // 64 bytes: [R || S] without V
	}
	
	// Wrap pubkey with ethermint type URL for chain compatibility
	wrappedPubKey, err := wrapPubKeyWithEthermintTypeURL(pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap pubkey: %w", err)
	}

	fmt.Printf("\n📝 TRANSACTION SIGNATURE DETAILS:\n")
	fmt.Printf("   PubKey Type: %T\n", wrappedPubKey)
	fmt.Printf("   PubKey Bytes (hex): %x\n", wrappedPubKey.Bytes())
	fmt.Printf("   PubKey Address: %s\n", sdk.AccAddress(wrappedPubKey.Address()).String())
	fmt.Printf("   Signature Length: %d bytes\n", len(sigWithoutRecoveryID))
	fmt.Printf("   Signature (hex): %x\n", sigWithoutRecoveryID)
	fmt.Printf("   Sign Mode: %s\n", signingtypes.SignMode_SIGN_MODE_LEGACY_AMINO_JSON)
	fmt.Printf("   Account Number: %d\n", accNumber)
	fmt.Printf("   Sequence: %d\n", sequence)
	fmt.Printf("\n")

	sig := signingtypes.SignatureV2{
		PubKey:   wrappedPubKey,
		Data:     sigData,
		Sequence: sequence,
	}
	if err := builder.SetSignatures(sig); err != nil {
		return nil, fmt.Errorf("failed to set signature: %w", err)
	}

	// Encode transaction
	txEncoder := txConfig.TxEncoder()
	txBytes, err := txEncoder(builder.GetTx())
	if err != nil {
		return nil, fmt.Errorf("failed to encode transaction: %w", err)
	}

	return txBytes, nil
}

// broadcastRawTx broadcasts the encoded transaction bytes to the network
func broadcastRawTx(nodeURL string, txBytes []byte) (string, error) {
	url := fmt.Sprintf("%s/cosmos/tx/v1beta1/txs", strings.TrimSuffix(nodeURL, "/"))

	// Prepare broadcast request
	reqBody := struct {
		TxBytes []byte `json:"tx_bytes"`
		Mode    string `json:"mode"`
	}{
		TxBytes: txBytes,
		Mode:    "BROADCAST_MODE_SYNC",
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP POST request
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("broadcast failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		TxResponse struct {
			TxHash string `json:"txhash"`
			Code   uint32 `json:"code"`
			RawLog string `json:"raw_log"`
		} `json:"tx_response"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.TxResponse.Code != 0 {
		return "", fmt.Errorf("transaction failed with code %d: %s",
			result.TxResponse.Code, result.TxResponse.RawLog)
	}

	return result.TxResponse.TxHash, nil
}

// wrapPubKeyWithEthermintTypeURL creates a pubkey Any with the ethermint type URL
// This ensures compatibility with chains that expect /ethermint.crypto.v1.ethsecp256k1.PubKey
func wrapPubKeyWithEthermintTypeURL(pubKey cryptotypes.PubKey) (cryptotypes.PubKey, error) {
	// Cast to ethsecp256k1.PubKey to get access to the key bytes
	ethPubKey, ok := pubKey.(*ethsecp256k1.PubKey)
	if !ok {
		return nil, fmt.Errorf("pubkey is not ethsecp256k1.PubKey: %T", pubKey)
	}
	
	// Create a new PubKey with the same bytes but will be encoded with ethermint type URL
	// The proto.RegisterType in crypto/codec/codec.go init() registers this type
	// with both type URLs, so decoding will work
	return ethPubKey, nil
}

