package cosmos

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	anteinterfaces "github.com/shardeum/shardeum-evm/ante/interfaces"
	"github.com/shardeum/shardeum-evm/crypto/ethsecp256k1"
	"github.com/shardeum/shardeum-evm/ethereum/eip712"
	"github.com/shardeum/shardeum-evm/types"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

var evmCodec codec.ProtoCodecMarshaler

func init() {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	evmCodec = codec.NewProtoCodec(registry)
}

// Deprecated: LegacyEip712SigVerificationDecorator Verify all signatures for a tx and return an error if any are invalid. Note,
// the LegacyEip712SigVerificationDecorator decorator will not get executed on ReCheck.
// NOTE: As of v10, EIP-712 signature verification is handled by the ethsecp256k1 public key (see ethsecp256k1.go)
//
// CONTRACT: Pubkeys are set in context for all signers before this decorator runs
// CONTRACT: Tx must implement SigVerifiableTx interface
type LegacyEip712SigVerificationDecorator struct {
	ak anteinterfaces.AccountKeeper
}

// Deprecated: NewLegacyEip712SigVerificationDecorator creates a new LegacyEip712SigVerificationDecorator
func NewLegacyEip712SigVerificationDecorator(
	ak anteinterfaces.AccountKeeper,
) LegacyEip712SigVerificationDecorator {
	return LegacyEip712SigVerificationDecorator{
		ak: ak,
	}
}

// AnteHandle handles validation of EIP712 signed cosmos txs.
// it is not run on RecheckTx
func (svd LegacyEip712SigVerificationDecorator) AnteHandle(ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	// no need to verify signatures on recheck tx
	if ctx.IsReCheckTx() {
		return next(ctx, tx, simulate)
	}

	// Check if this transaction has Web3Tx extension - if not, skip EIP-712 verification
	txWithExtensions, ok := tx.(authante.HasExtensionOptionsTx)
	if !ok {
		return next(ctx, tx, simulate)
	}
	
	opts := txWithExtensions.GetExtensionOptions()
	if len(opts) == 0 {
		return next(ctx, tx, simulate)
	}
	
	// Only use EIP-712 verification if the transaction has Web3Tx extension
	hasWeb3Tx := false
	for _, opt := range opts {
		if opt.GetTypeUrl() == "/cosmos.evm.types.v1.ExtensionOptionsWeb3Tx" {
			hasWeb3Tx = true
			break
		}
	}
	
	if !hasWeb3Tx {
		return next(ctx, tx, simulate)
	}

	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return ctx, errorsmod.Wrapf(errortypes.ErrInvalidType, "tx %T doesn't implement authsigning.SigVerifiableTx", tx)
	}

	authSignTx, ok := tx.(authsigning.Tx)
	if !ok {
		return ctx, errorsmod.Wrapf(errortypes.ErrInvalidType, "tx %T doesn't implement the authsigning.Tx interface", tx)
	}

	// stdSigs contains the sequence number, account number, and signatures.
	// When simulating, this would just be a 0-length slice.
	sigs, err := sigTx.GetSignaturesV2()
	if err != nil {
		return ctx, err
	}

	signerAddrs, err := sigTx.GetSigners()
	if err != nil {
		return ctx, err
	}

	// EIP712 allows just one signature
	if len(sigs) != 1 {
		return ctx, errorsmod.Wrapf(
			errortypes.ErrTooManySignatures,
			"invalid number of signers (%d);  EIP712 signatures allows just one signature",
			len(sigs),
		)
	}

	// check that signer length and signature length are the same
	if len(sigs) != len(signerAddrs) {
		return ctx, errorsmod.Wrapf(errortypes.ErrorInvalidSigner, "invalid number of signers;  expected: %d, got %d", len(signerAddrs), len(sigs))
	}

	// EIP712 has just one signature, avoid looping here and only read index 0
	i := 0
	sig := sigs[i]

	acc, err := authante.GetSignerAcc(ctx, svd.ak, signerAddrs[i])
	if err != nil {
		return ctx, err
	}

	// retrieve pubkey
	pubKey := acc.GetPubKey()
	if !simulate && pubKey == nil {
		return ctx, errorsmod.Wrap(errortypes.ErrInvalidPubKey, "pubkey on account is not set")
	}

	// Check account sequence number.
	if sig.Sequence != acc.GetSequence() {
		return ctx, errorsmod.Wrapf(
			errortypes.ErrWrongSequence,
			"account sequence mismatch, expected %d, got %d", acc.GetSequence(), sig.Sequence,
		)
	}

	// retrieve signer data
	genesis := ctx.BlockHeight() == 0
	chainID := ctx.ChainID()

	var accNum uint64
	if !genesis {
		accNum = acc.GetAccountNumber()
	}

	signerData := authsigning.SignerData{
		ChainID:       chainID,
		AccountNumber: accNum,
		Sequence:      acc.GetSequence(),
	}

	if simulate {
		return next(ctx, tx, simulate)
	}

	if err := VerifySignature(pubKey, signerData, sig.Data, authSignTx); err != nil {
		errMsg := fmt.Errorf("signature verification failed; please verify account number (%d) and chain-id (%s): %w", accNum, chainID, err)
		return ctx, errorsmod.Wrap(errortypes.ErrUnauthorized, errMsg.Error())
	}

	// Mark context to skip standard Cosmos signature verification
	// since EIP-712 signature was already verified
	ctx = ctx.WithValue("eip712-verified", true)

	return next(ctx, tx, simulate)
}

// VerifySignature verifies a transaction signature contained in SignatureData abstracting over different signing modes
// and single vs multi-signatures.
func VerifySignature(
	pubKey cryptotypes.PubKey,
	signerData authsigning.SignerData,
	sigData signing.SignatureData,
	tx authsigning.Tx,
) error {
	switch data := sigData.(type) {
	case *signing.SingleSignatureData:
		if data.SignMode != signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON {
			return errorsmod.Wrapf(errortypes.ErrNotSupported, "unexpected SignatureData %T: wrong SignMode", sigData)
		}

		// Note: this prevents the user from sending trash data in the signature field
		if len(data.Signature) != 0 {
			return errorsmod.Wrap(errortypes.ErrTooManySignatures, "invalid signature value; EIP712 must have the cosmos transaction signature empty")
		}

		// @contract: this code is reached only when Msg has Web3Tx extension (so this custom Ante handler flow),
		// and the signature is SIGN_MODE_LEGACY_AMINO_JSON which is supported for EIP712 for now

		msgs := tx.GetMsgs()
		if len(msgs) == 0 {
			return errorsmod.Wrap(errortypes.ErrNoSignatures, "tx doesn't contain any msgs to verify signature")
		}

		txBytes := legacytx.StdSignBytes(
			signerData.ChainID,
			signerData.AccountNumber,
			signerData.Sequence,
			tx.GetTimeoutHeight(),
			legacytx.StdFee{
				Amount: tx.GetFee(),
				Gas:    tx.GetGas(),
			},
			msgs, tx.GetMemo(),
		)

		// DEBUG: Log the StdSignBytes output
		fmt.Printf("\n������🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢\n")
		fmt.Printf("🟢 CHAIN STD SIGN BYTES\n")
		fmt.Printf("🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢\n")
		fmt.Printf("%s\n", string(txBytes))
		fmt.Printf("🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢\n\n")

		// Extract EVM chain ID from Cosmos chain ID (e.g., "shardeum_8117-1" -> 8117)
		// Format is: <name>_<evm-chain-id>-<version>
		var signerChainID uint64
		parts := strings.Split(signerData.ChainID, "_")
		if len(parts) == 2 {
			// Split on hyphen to get the EVM chain ID part
			evmParts := strings.Split(parts[1], "-")
			if len(evmParts) >= 1 {
				parsed, err := strconv.ParseUint(evmParts[0], 10, 64)
				if err == nil {
					signerChainID = parsed
				}
			}
		}
		
		if signerChainID == 0 {
			return errorsmod.Wrapf(errortypes.ErrInvalidChainID, "failed to extract EVM chain ID from: %s", signerData.ChainID)
		}

		txWithExtensions, ok := tx.(authante.HasExtensionOptionsTx)
		if !ok {
			return errorsmod.Wrap(errortypes.ErrUnknownExtensionOptions, "tx doesn't contain any extensions")
		}
		opts := txWithExtensions.GetExtensionOptions()
		if len(opts) != 1 {
			return errorsmod.Wrap(errortypes.ErrUnknownExtensionOptions, "tx doesn't contain expected amount of extension options")
		}

		extOpt, ok := opts[0].GetCachedValue().(*types.ExtensionOptionsWeb3Tx)
		if !ok {
			return errorsmod.Wrap(errortypes.ErrUnknownExtensionOptions, "unknown extension option")
		}

		if extOpt.TypedDataChainID != signerChainID {
			return errorsmod.Wrap(errortypes.ErrInvalidChainID, "invalid chain-id")
		}

		if len(extOpt.FeePayer) == 0 {
			return errorsmod.Wrap(errortypes.ErrUnknownExtensionOptions, "no feePayer on ExtensionOptionsWeb3Tx")
		}
		feePayer, err := sdk.AccAddressFromBech32(extOpt.FeePayer)
		if err != nil {
			return errorsmod.Wrap(err, "failed to parse feePayer from ExtensionOptionsWeb3Tx")
		}

		feeDelegation := &eip712.FeeDelegationOptions{
			FeePayer: feePayer,
		}

		typedData, err := eip712.LegacyWrapTxToTypedData(evmCodec, extOpt.TypedDataChainID, msgs[0], txBytes, feeDelegation)
		if err != nil {
			return errorsmod.Wrap(err, "failed to create EIP-712 typed data from tx")
		}

		// Use custom hash function that preserves decimal chainId format (matches MetaMask)
		sigHash, typedDataJSONStr, err := eip712.TypedDataAndHashWithDecimalChainID(typedData)
		if err != nil {
			return errorsmod.Wrap(err, "failed to calculate EIP-712 hash")
		}

		// DEBUG: Log the reconstructed typed data
		fmt.Printf("\n🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡\n")
		fmt.Printf("🟡 CHAIN RECONSTRUCTED TYPED DATA (with decimal chainId)\n")
		fmt.Printf("🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡\n")
		fmt.Printf("%s\n", typedDataJSONStr)
		fmt.Printf("🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡🟡\n\n")
		
		// Save to file for comparison
		_ = os.WriteFile("/tmp/chain-typed-data.json", []byte(typedDataJSONStr), 0644)


		// DEBUG: Log the calculated hash
		fmt.Printf("\n🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴\n")
		fmt.Printf("🔴 CHAIN CALCULATED HASH\n")
		fmt.Printf("🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴\n")
		fmt.Printf("%x\n", sigHash)
		fmt.Printf("🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴\n\n")
		
		// DEBUG: Log feePayer signature
		fmt.Printf("\n🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣\n")
		fmt.Printf("🟣 FEE PAYER SIGNATURE (from ExtensionOptionsWeb3Tx)\n")
		fmt.Printf("🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣\n")
		fmt.Printf("%x\n", extOpt.FeePayerSig)
		fmt.Printf("🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣🟣\n\n")
		
		fmt.Printf("🔍🔍🔍 EIP712 DEBUG END 🔍🔍🔍\n\n")

		feePayerSig := extOpt.FeePayerSig
		if len(feePayerSig) != ethcrypto.SignatureLength {
			return errorsmod.Wrap(errortypes.ErrorInvalidSigner, "signature length doesn't match typical [R||S||V] signature 65 bytes")
		}

		// Remove the recovery offset if needed (ie. Metamask eip712 signature)
		if feePayerSig[ethcrypto.RecoveryIDOffset] == 27 || feePayerSig[ethcrypto.RecoveryIDOffset] == 28 {
			feePayerSig[ethcrypto.RecoveryIDOffset] -= 27
		}

		feePayerPubkey, err := secp256k1.RecoverPubkey(sigHash, feePayerSig)
		if err != nil {
			return errorsmod.Wrap(err, "failed to recover delegated fee payer from sig")
		}

		ecPubKey, err := ethcrypto.UnmarshalPubkey(feePayerPubkey)
		if err != nil {
			return errorsmod.Wrap(err, "failed to unmarshal recovered fee payer pubkey")
		}

		pk := &ethsecp256k1.PubKey{
			Key: ethcrypto.CompressPubkey(ecPubKey),
		}

		if !pubKey.Equals(pk) {
			return errorsmod.Wrapf(errortypes.ErrInvalidPubKey, "feePayer pubkey %s is different from transaction pubkey %s", pubKey, pk)
		}

		recoveredFeePayerAcc := sdk.AccAddress(pk.Address().Bytes())

		if !recoveredFeePayerAcc.Equals(feePayer) {
			return errorsmod.Wrapf(errortypes.ErrorInvalidSigner, "failed to verify delegated fee payer %s signature", recoveredFeePayerAcc)
		}

	// DEBUG: Log verification inputs
	fmt.Printf("\n🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢\n")
	fmt.Printf("🟢 ABOUT TO CALL secp256k1.VerifySignature\n")
	fmt.Printf("🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢\n")
	fmt.Printf("pubKey.Bytes() (compressed): %x\n", pubKey.Bytes())
	fmt.Printf("feePayerPubkey (uncompressed): %x\n", feePayerPubkey)
	fmt.Printf("sigHash: %x\n", sigHash)
	fmt.Printf("feePayerSig[0:64]: %x\n", feePayerSig[:len(feePayerSig)-1])
	fmt.Printf("🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢🟢\n\n")

	// Try both compressed and uncompressed public key formats for verification
	// go-ethereum's VerifySignature accepts both formats
	var verified bool
	useCompressed := true // Toggle this to switch between formats
	
	if useCompressed {
		// Use compressed public key (33 bytes)
		fmt.Printf("🔧 Using COMPRESSED pubkey for verification\n")
		verified = secp256k1.VerifySignature(pubKey.Bytes(), sigHash, feePayerSig[:len(feePayerSig)-1])
	} else {
		// Use uncompressed public key (64 bytes without 0x04 prefix)
		// feePayerPubkey from RecoverPubkey is uncompressed (65 bytes with 0x04 prefix)
		// We need to strip the 0x04 prefix to get the 64-byte uncompressed key
		fmt.Printf("🔧 Using UNCOMPRESSED pubkey for verification\n")
		verified = secp256k1.VerifySignature(feePayerPubkey[1:], sigHash, feePayerSig[:len(feePayerSig)-1])
	}
	
	if !verified {
		fmt.Printf("\n❌❌❌ SIGNATURE VERIFICATION FAILED ❌❌❌\n\n")
		return errorsmod.Wrap(errortypes.ErrorInvalidSigner, "unable to verify signer signature of EIP712 typed data")
	}
	
	fmt.Printf("\n✅✅✅ EIP-712 SIGNATURE VERIFICATION SUCCEEDED! ✅✅✅\n\n")
	return nil
	default:
		return errorsmod.Wrapf(errortypes.ErrTooManySignatures, "unexpected SignatureData %T", sigData)
	}
}
