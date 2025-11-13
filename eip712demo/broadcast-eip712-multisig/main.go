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
	cryptocodec "github.com/shardeum/shardeum-evm/crypto/codec"
	cryptomultisig "github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
	"github.com/cosmos/cosmos-sdk/crypto/types/multisig"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/shardeum/shardeum-evm/crypto/ethsecp256k1"
	ethermint "github.com/shardeum/shardeum-evm/types"
	"github.com/spf13/cobra"
	"sort"
)

// evmCodec is the codec used for EIP-712 operations, matching the chain's codec
var evmCodec codec.ProtoCodecMarshaler

// signerInfo holds information about a signer in a multisig transaction
type signerInfo struct {
	tx       SignedEIP712Tx
	pubKey   cryptotypes.PubKey
	sig      []byte
	keyIndex int
}

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
	banktypes.RegisterInterfaces(registry)
	stakingtypes.RegisterInterfaces(registry)
	cryptocodec.RegisterInterfaces(registry)
	evmCodec = codec.NewProtoCodec(registry)
	
	// Set the amino codec for legacy StdSignBytes
	aminoCodec := codec.NewLegacyAmino()
	banktypes.RegisterLegacyAminoCodec(aminoCodec)
	stakingtypes.RegisterLegacyAminoCodec(aminoCodec)
	cryptocodec.RegisterCrypto(aminoCodec)
	legacytx.RegressionTestingAminoCodec = aminoCodec
}

// SignedEIP712Tx represents the JSON structure from MetaMask (multisig version)
type SignedEIP712Tx struct {
	TypedData             apitypes.TypedData `json:"typedData"`
	Signature             string             `json:"signature"`
	EthAddressSigner      string             `json:"ethAddressSigner"`      // signer's ETH address
	Bech32Signer          string             `json:"bech32Signer"`          // signer's bech32 address
	MultisigDelegatorAddr string             `json:"multisigDelegatorAddr"` // multisig account address
	ValidatorAddress      string             `json:"validatorAddress"`
	Amount                string             `json:"amount"`
	Denom                 string             `json:"denom"`
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
	var thresholdFlag int

	rootCmd := &cobra.Command{
		Use:   "broadcast-eip712-multisig [signed-tx1.json] [signed-tx2.json] ...",
		Short: "Parse, display, and optionally broadcast EIP-712 multisig signed transactions",
		Long: `Parse EIP-712 signed transactions from MetaMask (multisig version) and display their contents.
Can also broadcast the combined multisig transaction to the network with --broadcast flag.

⚠️  IMPORTANT: All signed transactions must sign the SAME payload!
The tool will verify all TypedData hashes match before proceeding.

Example:
  # Display only (2 signers)
  broadcast-eip712-multisig sig1.json sig2.json

  # Display only (3 signers)
  broadcast-eip712-multisig sig1.json sig2.json sig3.json

  # Broadcast to local node (chain ID: shardeum_8117-1)
  broadcast-eip712-multisig sig1.json sig2.json sig3.json --broadcast --node http://localhost:1317 --cosmos-chain-id shardeum_8117-1

  # Broadcast to mainnet (chain ID: shardeum_8118-1)
  broadcast-eip712-multisig sig1.json sig2.json --broadcast --node https://gcp.rpc.shardeum.org --evm-chain-id 8118 --cosmos-chain-id shardeum_8118-1
`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if broadcast {
				return broadcastTx(args, nodeURL, evmChainID, cosmosChainID, thresholdFlag)
			}
			return displayTx(args)
		},
	}

	rootCmd.Flags().BoolVar(&broadcast, "broadcast", false, "Broadcast the transaction to the network")
	rootCmd.Flags().StringVar(&nodeURL, "node", "http://localhost:1317", "Node REST API URL")
	rootCmd.Flags().Uint64Var(&evmChainID, "evm-chain-id", 8117, "EVM chain ID (8117 for local, 8118 for mainnet, 8119 for testnet)")
	rootCmd.Flags().StringVar(&cosmosChainID, "cosmos-chain-id", "shardeum_8117-1", "Cosmos chain ID (must match signed transaction)")
	rootCmd.Flags().IntVar(&thresholdFlag, "threshold", 0, "Multisig threshold (required if account has no pubkey on chain, e.g., first transaction)")


	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func displayTx(filePaths []string) error {
	// Load and verify all signed transactions
	signedTxs, err := loadAndVerifySignedFiles(filePaths)
	if err != nil {
		return err
	}

	// Use the first transaction as the base for transaction details
	baseTx := signedTxs[0]

	// Display transaction info
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("📋 EIP-712 Multisig Transaction Details")
	fmt.Println("=" + strings.Repeat("=", 79))

	fmt.Printf("\n� Multisig Account:\n")
	fmt.Printf("   Multisig Address: %s\n", baseTx.MultisigDelegatorAddr)
	fmt.Printf("   Number of Signers: %d\n", len(signedTxs))

	fmt.Printf("\n✍️  Signers:\n")
	for i, tx := range signedTxs {
		fmt.Printf("   %d. ETH:     %s\n", i+1, tx.EthAddressSigner)
		fmt.Printf("      Bech32:  %s\n", tx.Bech32Signer)
		fmt.Printf("      Sig:     %s\n", tx.Signature[:20]+"...")
	}

	fmt.Printf("\n💰 Transaction Details:\n")

	// Parse messages
	msgs, err := buildMessages(baseTx)
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
			}
		}
	}

	fmt.Printf("\n⛽ Fee & Gas:\n")
	feeAmount, gas := baseTx.GetFee()
	fmt.Printf("   Gas Limit: %s\n", gas)
	if len(feeAmount) > 0 {
		fmt.Printf("   Fee:       %s %s\n",
			feeAmount[0].Amount,
			feeAmount[0].Denom)
	}

	fmt.Printf("\n📝 Memo: %s\n", baseTx.GetMemo())
	fmt.Printf("🌐 Chain ID: %s\n", baseTx.GetChainID())
	fmt.Printf("🔢 Sequence: %s\n", baseTx.GetSequence())

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("� To broadcast this transaction, run again with --broadcast flag")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("\nExample:\n")
	fmt.Printf("  broadcast-eip712-multisig %s --broadcast --node http://localhost:1317\n", 
		strings.Join(filePaths, " "))
	fmt.Println()

	return nil
}

// loadAndVerifySignedFiles loads N JSON files and ensures the typed data hash matches.
func loadAndVerifySignedFiles(paths []string) ([]SignedEIP712Tx, error) {
	out := make([]SignedEIP712Tx, 0, len(paths))
	var firstHash []byte
	var firstMultisigAddr string
	
	for i, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", p, err)
		}
		
		var tx SignedEIP712Tx
		if err := json.Unmarshal(data, &tx); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", p, err)
		}
		
		// Verify TypedData hash matches
		hash, _, err := apitypes.TypedDataAndHash(tx.TypedData)
		if err != nil {
			return nil, fmt.Errorf("failed to hash TypedData in %s: %w", p, err)
		}
		
		if i == 0 {
			firstHash = hash
			firstMultisigAddr = tx.MultisigDelegatorAddr
		} else {
			if !bytes.Equal(firstHash, hash) {
				return nil, fmt.Errorf("TypedData mismatch: %s has different payload than first file", p)
			}
			if tx.MultisigDelegatorAddr != firstMultisigAddr {
				return nil, fmt.Errorf("multisig address mismatch: %s has %s, expected %s", 
					p, tx.MultisigDelegatorAddr, firstMultisigAddr)
			}
		}
		
		out = append(out, tx)
	}
	
	return out, nil
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

func buildMessages(signedTx SignedEIP712Tx) ([]sdk.Msg, error) {
	var msgs []sdk.Msg

	for _, rawMsg := range signedTx.GetMsgs() {
		var msgWrapper struct {
			Type  string          `json:"type"`
			Value json.RawMessage `json:"value"`
		}

		if err := json.Unmarshal(rawMsg, &msgWrapper); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message wrapper: %w", err)
		}

		switch msgWrapper.Type {
		case "cosmos-sdk/MsgDelegate":
			var delegateValue struct {
				DelegatorAddress string `json:"delegator_address"`
				ValidatorAddress string `json:"validator_address"`
				Amount           struct {
					Denom  string `json:"denom"`
					Amount string `json:"amount"`
				} `json:"amount"`
			}
			if err := json.Unmarshal(msgWrapper.Value, &delegateValue); err != nil {
				return nil, fmt.Errorf("failed to unmarshal MsgDelegate: %w", err)
			}

			amount, ok := sdkmath.NewIntFromString(delegateValue.Amount.Amount)
			if !ok {
				return nil, fmt.Errorf("invalid amount: %s", delegateValue.Amount.Amount)
			}

			msg := stakingtypes.NewMsgDelegate(
				delegateValue.DelegatorAddress,
				delegateValue.ValidatorAddress,
				sdk.NewCoin(delegateValue.Amount.Denom, amount),
			)
			msgs = append(msgs, msg)

		default:
			return nil, fmt.Errorf("unsupported message type: %s", msgWrapper.Type)
		}
	}

	return msgs, nil
}

// broadcastTx broadcasts the multisig EIP-712 transaction to the network
func broadcastTx(filePaths []string, nodeURL string, evmChainID uint64, cosmosChainID string, thresholdFlag int) error {
	// Step 1: Load and verify all signed transactions
	fmt.Println("🚀 Broadcasting EIP-712 Multisig Transaction")
	fmt.Println(strings.Repeat("=", 80))
	
	fmt.Println("\n📂 Step 1: Loading and verifying signed files...")
	signedTxs, err := loadAndVerifySignedFiles(filePaths)
	if err != nil {
		return fmt.Errorf("failed to load signed files: %w", err)
	}
	fmt.Printf("   ✅ Loaded %d signatures\n", len(signedTxs))
	fmt.Printf("   ✅ All TypedData payloads match\n")
	
	// Use first transaction as base
	baseTx := signedTxs[0]
	multisigAddr := baseTx.MultisigDelegatorAddr

	// Validate chain ID match
	signedChainID := baseTx.GetChainID()
	if signedChainID != cosmosChainID {
		fmt.Printf("\n⚠️  WARNING: Chain ID Mismatch!\n")
		fmt.Printf("   Signed TX Chain ID:  %s\n", signedChainID)
		fmt.Printf("   Target Chain ID:     %s\n", cosmosChainID)
		fmt.Printf("\n   The transaction was signed for a different chain!\n")
		fmt.Printf("   You need to re-sign the transaction with the correct chain ID.\n")
		return fmt.Errorf("chain ID mismatch: signed=%s, target=%s", 
			signedChainID, cosmosChainID)
	}

	// Step 2: Query multisig account from chain (or recover pubkeys if new account)
	fmt.Println("\n📡 Step 2: Querying multisig account from chain...")
	multisigPubKey, accNumber, sequence, threshold, err := queryMultisigAccount(nodeURL, multisigAddr)
	
	// If account has no public key (first transaction), we'll construct it from signatures
	accountHasNoPubKey := false
	if err != nil && strings.Contains(err.Error(), "account has no public key") {
		fmt.Printf("   ⚠️  Account has no public key on chain (first transaction)\n")
		fmt.Printf("   ℹ️  Will construct multisig public key from signatures\n")
		accountHasNoPubKey = true
		
		// Still need account number and sequence, so query the account without requiring pubkey
		accNumber, sequence, err = queryAccountBasic(nodeURL, multisigAddr)
		if err != nil {
			return fmt.Errorf("failed to query account info: %w", err)
		}
		fmt.Printf("   ✅ Account Number: %d, Sequence: %d\n", accNumber, sequence)
	} else if err != nil {
		return fmt.Errorf("failed to query multisig account: %w", err)
	}
	
	var constituentKeys []cryptotypes.PubKey
	
	if !accountHasNoPubKey {
		// Account has pubkey on chain, extract it
		legacyMultisig, ok := multisigPubKey.(*cryptomultisig.LegacyAminoPubKey)
		if !ok {
			return fmt.Errorf("account pubkey is not LegacyAminoPubKey (got %T)", multisigPubKey)
		}
		
		constituentKeys = legacyMultisig.GetPubKeys()
		
		fmt.Printf("   ✅ Multisig Address: %s\n", multisigAddr)
		fmt.Printf("   ✅ Account Number: %d, Sequence: %d\n", accNumber, sequence)
		fmt.Printf("   ✅ Threshold: %d of %d\n", threshold, len(constituentKeys))
		fmt.Printf("   ✅ Constituent Keys:\n")
		for i, key := range constituentKeys {
			addr := sdk.AccAddress(key.Address())
			fmt.Printf("      %d. %s\n", i+1, addr.String())
		}
	}
	
	// Step 3: Recover public keys from signatures and match to multisig
	fmt.Println("\n🔑 Step 3: Recovering public keys from signatures...")
	
	signers := make([]signerInfo, 0, len(signedTxs))
	recoveredPubKeys := make([]cryptotypes.PubKey, 0, len(signedTxs))
	
	for _, tx := range signedTxs {
		// Parse signature
		sig, err := parseSignature(tx.Signature)
		if err != nil {
			return fmt.Errorf("failed to parse signature for %s: %w", tx.Bech32Signer, err)
		}
		
		// Recover public key
		sigHash, _, err := apitypes.TypedDataAndHash(tx.TypedData)
		if err != nil {
			return fmt.Errorf("failed to hash typed data: %w", err)
		}
		
		pubKeyBytes, err := crypto.Ecrecover(sigHash, sig)
		if err != nil {
			return fmt.Errorf("failed to recover public key for %s: %w", tx.Bech32Signer, err)
		}
		
		// Convert to compressed format
		ecdsaPubKey, err := crypto.UnmarshalPubkey(pubKeyBytes)
		if err != nil {
			return fmt.Errorf("failed to unmarshal public key: %w", err)
		}
		compressedPubKey := crypto.CompressPubkey(ecdsaPubKey)
		pubKey := &ethsecp256k1.PubKey{Key: compressedPubKey}
		
		// Verify recovered address matches expected
		recoveredAddr := sdk.AccAddress(pubKey.Address())
		if recoveredAddr.String() != tx.Bech32Signer {
			return fmt.Errorf("recovered address %s does not match expected %s",
				recoveredAddr.String(), tx.Bech32Signer)
		}
		
		fmt.Printf("   ✅ Recovered: %s\n", recoveredAddr.String())
		recoveredPubKeys = append(recoveredPubKeys, pubKey)
		
		signers = append(signers, signerInfo{
			tx:       tx,
			pubKey:   pubKey,
			sig:      sig,
			keyIndex: -1, // Will set below
		})
	}
	
	// If account has no pubkey, construct it from recovered keys
	if accountHasNoPubKey {
		// Use threshold from flag if provided, otherwise use number of signers
		if thresholdFlag > 0 {
			threshold = uint32(thresholdFlag)
		} else {
			threshold = uint32(len(recoveredPubKeys))
		}
		
		if threshold > uint32(len(recoveredPubKeys)) {
			return fmt.Errorf("threshold %d cannot be greater than number of signers %d", 
				threshold, len(recoveredPubKeys))
		}
		
		// Sort recovered pubkeys by address for deterministic multisig address
		sort.Slice(recoveredPubKeys, func(i, j int) bool {
			return bytes.Compare(recoveredPubKeys[i].Address(), recoveredPubKeys[j].Address()) < 0
		})
		
		multisigPubKey = cryptomultisig.NewLegacyAminoPubKey(int(threshold), recoveredPubKeys)
		constituentKeys = recoveredPubKeys
		
		// Verify the constructed multisig matches the expected address
		constructedAddr := sdk.AccAddress(multisigPubKey.Address())
		if constructedAddr.String() != multisigAddr {
			fmt.Printf("   ⚠️  WARNING: Constructed multisig address does not match!\n")
			fmt.Printf("      Constructed: %s\n", constructedAddr.String())
			fmt.Printf("      Expected:    %s\n", multisigAddr)
			fmt.Printf("\n")
			fmt.Printf("   This could be due to:\n")
			fmt.Printf("   1. Wrong threshold (you specified %d, try different values with --threshold)\n", threshold)
			fmt.Printf("   2. Different key ordering used when creating the multisig\n")
			fmt.Printf("\n")
			return fmt.Errorf("multisig address mismatch")
		}
		
		fmt.Printf("   ✅ Constructed multisig with threshold %d of %d\n", threshold, len(constituentKeys))
		fmt.Printf("   ✅ Multisig address verified: %s\n", constructedAddr.String())
	}
	
	// Now match signers to their index in the multisig
	for i := range signers {
		keyIndex := -1
		for j, key := range constituentKeys {
			if bytes.Equal(key.Address().Bytes(), signers[i].pubKey.Address().Bytes()) {
				keyIndex = j
				break
			}
		}
		
		if keyIndex < 0 {
			return fmt.Errorf("signer %s (%s) is not part of the multisig account",
				signers[i].tx.Bech32Signer, signers[i].tx.EthAddressSigner)
		}
		
		signers[i].keyIndex = keyIndex
		if !accountHasNoPubKey {
			fmt.Printf("   ✅ Matched: %s (index %d in multisig)\n", 
				sdk.AccAddress(signers[i].pubKey.Address()).String(), keyIndex)
		}
	}
	
	// Sort signers by their key index in the multisig
	sort.Slice(signers, func(i, j int) bool {
		return signers[i].keyIndex < signers[j].keyIndex
	})
	
	fmt.Printf("   ✅ All signers validated and sorted\n")
	
	// Verify we have enough signatures
	if len(signers) < int(threshold) {
		return fmt.Errorf("insufficient signatures: have %d, need %d", len(signers), threshold)
	}

	// Step 4: Build messages
	fmt.Println("\n� Step 4: Building transaction messages...")
	msgs, err := buildMessages(baseTx)
	if err != nil {
		return fmt.Errorf("failed to build messages: %w", err)
	}
	fmt.Printf("   ✅ %d message(s) built\n", len(msgs))

	// Step 5: Construct Cosmos transaction with multisig
	fmt.Println("\n🏗️  Step 5: Constructing multisig transaction...")
	txBytes, err := buildMultisigTx(baseTx, msgs, multisigPubKey, signers, accNumber, sequence)
	if err != nil {
		return fmt.Errorf("failed to build transaction: %w", err)
	}
	fmt.Printf("   ✅ Transaction encoded: %d bytes\n", len(txBytes))

	// Step 6: Broadcast transaction
	fmt.Println("\n📤 Step 6: Broadcasting to network...")
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

// queryMultisigAccount queries the multisig account from the chain and extracts its pubkey and threshold
func queryMultisigAccount(nodeURL, address string) (cryptotypes.PubKey, uint64, uint64, uint32, error) {
	url := fmt.Sprintf("%s/cosmos/auth/v1beta1/accounts/%s", strings.TrimSuffix(nodeURL, "/"), address)

	resp, err := http.Get(url)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, 0, 0, fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("failed to read response: %w", err)
	}

	// Create codec for unmarshaling
	registry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	cryptocodec.RegisterInterfaces(registry)
	ethermint.RegisterInterfaces(registry)
	
	// Register multisig types explicitly
	registry.RegisterImplementations(
		(*cryptotypes.PubKey)(nil),
		&cryptomultisig.LegacyAminoPubKey{},
	)
	
	cdc := codec.NewProtoCodec(registry)

	var accountResponse struct {
		Account json.RawMessage `json:"account"`
	}
	if err := json.Unmarshal(body, &accountResponse); err != nil {
		return nil, 0, 0, 0, fmt.Errorf("failed to unmarshal account response: %w", err)
	}

	var account authtypes.AccountI
	if err := cdc.UnmarshalInterfaceJSON(accountResponse.Account, &account); err != nil {
		return nil, 0, 0, 0, fmt.Errorf("failed to unmarshal account: %w", err)
	}

	pubKey := account.GetPubKey()
	if pubKey == nil {
		return nil, 0, 0, 0, fmt.Errorf("account has no public key")
	}

	// Extract threshold from LegacyAminoPubKey
	legacyMultisig, ok := pubKey.(*cryptomultisig.LegacyAminoPubKey)
	if !ok {
		return nil, 0, 0, 0, fmt.Errorf("account pubkey is not LegacyAminoPubKey (got %T)", pubKey)
	}

	return pubKey, account.GetAccountNumber(), account.GetSequence(), legacyMultisig.Threshold, nil
}

// queryAccountBasic queries just the account number and sequence (no pubkey required)
func queryAccountBasic(nodeURL, address string) (uint64, uint64, error) {
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read response: %w", err)
	}

	// Create codec for unmarshaling
	registry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	cryptocodec.RegisterInterfaces(registry)
	ethermint.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	var accountResponse struct {
		Account json.RawMessage `json:"account"`
	}
	if err := json.Unmarshal(body, &accountResponse); err != nil {
		return 0, 0, fmt.Errorf("failed to unmarshal account response: %w", err)
	}

	var account authtypes.AccountI
	if err := cdc.UnmarshalInterfaceJSON(accountResponse.Account, &account); err != nil {
		return 0, 0, fmt.Errorf("failed to unmarshal account: %w", err)
	}

	return account.GetAccountNumber(), account.GetSequence(), nil
}

// buildMultisigTx constructs a Cosmos SDK transaction with multisig signatures
func buildMultisigTx(
	baseTx SignedEIP712Tx,
	msgs []sdk.Msg,
	multisigPubKey cryptotypes.PubKey,
	signers []signerInfo,
	accNumber, sequence uint64,
) ([]byte, error) {
	// Create codec
	registry := codectypes.NewInterfaceRegistry()
	banktypes.RegisterInterfaces(registry)
	stakingtypes.RegisterInterfaces(registry)
	authtypes.RegisterInterfaces(registry)
	ethermint.RegisterInterfaces(registry)
	cryptocodec.RegisterInterfaces(registry)
	
	protoCodec := codec.NewProtoCodec(registry)

	// Create tx config
	txConfig := authtx.NewTxConfig(protoCodec, authtx.DefaultSignModes)

	// Create tx builder
	txBuilder := txConfig.NewTxBuilder()

	// Set messages
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return nil, fmt.Errorf("failed to set messages: %w", err)
	}

	// Set fee
	feeAmounts, gas := baseTx.GetFee()
	sdkFeeAmount := make([]sdk.Coin, 0)
	for _, fee := range feeAmounts {
		amount, ok := sdkmath.NewIntFromString(fee.Amount)
		if !ok {
			return nil, fmt.Errorf("invalid fee amount: %s", fee.Amount)
		}
		sdkFeeAmount = append(sdkFeeAmount, sdk.NewCoin(fee.Denom, amount))
	}
	txBuilder.SetFeeAmount(sdkFeeAmount)

	// Set gas limit
	var gasLimit uint64
	fmt.Sscanf(gas, "%d", &gasLimit)
	txBuilder.SetGasLimit(gasLimit)

	// Set memo
	txBuilder.SetMemo(baseTx.GetMemo())

	// Build multisig signature data
	legacyMultisig, _ := multisigPubKey.(*cryptomultisig.LegacyAminoPubKey)
	constituentKeys := legacyMultisig.GetPubKeys()
	multiSigData := multisig.NewMultisig(len(constituentKeys))

	// Add each signature
	for _, signer := range signers {
		// Remove recovery ID (V) from signature - use only [R || S]
		sigWithoutV := signer.sig[:64]
		
		sigV2 := signingtypes.SignatureV2{
			PubKey: signer.pubKey,
			Data: &signingtypes.SingleSignatureData{
				SignMode:  signingtypes.SignMode_SIGN_MODE_LEGACY_AMINO_JSON,
				Signature: sigWithoutV,
			},
			Sequence: sequence,
		}

		if err := multisig.AddSignatureV2(multiSigData, sigV2, constituentKeys); err != nil {
			return nil, fmt.Errorf("failed to add signature: %w", err)
		}
	}

	// Set the multisig signature
	multisigSig := signingtypes.SignatureV2{
		PubKey:   multisigPubKey,
		Data:     multiSigData,
		Sequence: sequence,
	}

	if err := txBuilder.SetSignatures(multisigSig); err != nil {
		return nil, fmt.Errorf("failed to set multisig signature: %w", err)
	}

	// Encode transaction
	txEncoder := txConfig.TxEncoder()
	txBytes, err := txEncoder(txBuilder.GetTx())
	if err != nil {
		return nil, fmt.Errorf("failed to encode transaction: %w", err)
	}

	return txBytes, nil
}

// ============================================================================
// UNUSED FUNCTIONS - Kept for reference (from single-signer version)
// ============================================================================

/*
// recoverPubKeyFromEIP712 recovers the public key from EIP-712 signature
// NOTE: This function is from the single-signer version and is not used in multisig
func recoverPubKeyFromEIP712(signedTx SignedEIP712Tx, sig []byte, evmChainID uint64) (cryptotypes.PubKey, error) {
	// Build messages from the signed transaction
	msgs, err := buildMessages(signedTx)
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
	fmt.Sscanf("0", "%d", &accNum) // account_number is "0" in the TypedData
	fmt.Sscanf(sequence, "%d", &seq)
	
	// Use legacytx.StdSignBytes to create proper Amino JSON
	aminoBytes := legacytx.StdSignBytes(chainID, accNum, seq, 0, 
		legacytx.NewStdFee(gasLimit, feeCoins), msgs, memo) //#nosec G115
	
	// Use the chain's EIP-712 function to build TypedData
	// Use evmCodec which was initialized with all necessary interfaces
	feeDelegation := &eip712.FeeDelegationOptions{
		FeePayer: sdk.MustAccAddressFromBech32(signedTx.MultisigDelegatorAddr),
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
	if recoveredAddr.String() != signedTx.Bech32Signer {
		return nil, fmt.Errorf("recovered address %s does not match expected %s",
			recoveredAddr.String(), signedTx.Bech32Signer)
	}

	return pubKey, nil
}

// queryAccount queries the account number and sequence from the chain
// NOTE: This function is from the single-signer version and is not used in multisig
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

// buildEIP712Tx constructs a Cosmos SDK transaction with EIP-712 extension (single signer)
// NOTE: This function is from the single-signer version and is not used in multisig
func buildEIP712Tx(
	signedTx SignedEIP712Tx,
	msgs []sdk.Msg,
	pubKey cryptotypes.PubKey,
	signature []byte,
	accNumber, sequence uint64,
) ([]byte, error) {
	// Create codec
	registry := codectypes.NewInterfaceRegistry()
	banktypes.RegisterInterfaces(registry)
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

// wrapPubKeyWithEthermintTypeURL creates a pubkey Any with the ethermint type URL
// This ensures compatibility with chains that expect /ethermint.crypto.v1.ethsecp256k1.PubKey
// NOTE: This function is from the single-signer version and is not used in multisig
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
*/

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

