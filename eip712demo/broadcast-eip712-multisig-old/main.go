package main

import (
    "bytes"
    "context"
    "encoding/base64"
    "encoding/hex"
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "net/http"
    "os"
    "sort"
    "strings"

    sdkmath "cosmossdk.io/math"
    "github.com/cosmos/cosmos-sdk/client"
    "github.com/cosmos/cosmos-sdk/codec"
    codectypes "github.com/cosmos/cosmos-sdk/codec/types"
    cryptomultisig "github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
    "github.com/cosmos/cosmos-sdk/crypto/types/multisig"
    sdkcryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
    sdk "github.com/cosmos/cosmos-sdk/types"
    signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
    "github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
    authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
    authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
    banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
    stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

    "github.com/ethereum/go-ethereum/crypto"
    "github.com/ethereum/go-ethereum/signer/core/apitypes"

    cryptocodec "github.com/shardeum/shardeum-evm/crypto/codec"
    ethsecp "github.com/shardeum/shardeum-evm/crypto/ethsecp256k1"
    etherminttypes "github.com/shardeum/shardeum-evm/types"
)

func init() {
    // Set SDK config for Shardeum
    config := sdk.GetConfig()
    config.SetBech32PrefixForAccount("shardeum", "shardeum"+sdk.PrefixPublic)
    config.SetBech32PrefixForValidator("shardeum"+sdk.PrefixValidator+sdk.PrefixOperator, "shardeum"+sdk.PrefixValidator+sdk.PrefixOperator+sdk.PrefixPublic)
    config.SetBech32PrefixForConsensusNode("shardeum"+sdk.PrefixValidator+sdk.PrefixConsensus, "shardeum"+sdk.PrefixValidator+sdk.PrefixConsensus+sdk.PrefixPublic)
    config.Seal()
}

// SignedEIP712Tx is the JSON blob produced by the HTML page per signer.
type SignedEIP712Tx struct {
    TypedData       apitypes.TypedData `json:"typedData"`
    Signature       string             `json:"signature"`       // 0x-prefixed r|s|v
    EthAddress      string             `json:"ethAddress"`      // signer EOA
    Bech32Delegator string             `json:"bech32Delegator"` // multisig delegator address
    ValidatorAddr   string             `json:"validatorAddress"`
    Amount          string             `json:"amount"`
    Denom           string             `json:"denom"`
}

// --- helpers to interpret the typedData message (adapt as needed to match your schema) ---

func (tx *SignedEIP712Tx) getMsgs() []json.RawMessage {
    m := tx.TypedData.Message
    raw, ok := m["messages"]
    if !ok {
        return nil
    }
    bs, _ := json.Marshal(raw)
    var out []json.RawMessage
    _ = json.Unmarshal(bs, &out)
    return out
}

func (tx *SignedEIP712Tx) getFee() ([]sdk.Coin, string) {
    m := tx.TypedData.Message
    feeVal, ok := m["fee"]
    if !ok {
        return nil, "0"
    }
    bs, _ := json.Marshal(feeVal)
    var fee struct {
        Amount []struct {
            Denom  string `json:"denom"`
            Amount string `json:"amount"`
        } `json:"amount"`
        Gas string `json:"gas"`
    }
    _ = json.Unmarshal(bs, &fee)
    
    coins := make([]sdk.Coin, len(fee.Amount))
    for i, a := range fee.Amount {
        amt, ok := sdkmath.NewIntFromString(a.Amount)
        if !ok {
            continue
        }
        coins[i] = sdk.NewCoin(a.Denom, amt)
    }
    return coins, fee.Gas
}

func (tx *SignedEIP712Tx) getMemo() string {
    m := tx.TypedData.Message
    if memo, ok := m["memo"].(string); ok {
        return memo
    }
    return ""
}

// --- multisig helpers ---

// loadSignedFiles loads N JSON files and ensures the typed data hash matches.
func loadSignedFiles(paths []string) ([]SignedEIP712Tx, error) {
    out := make([]SignedEIP712Tx, 0, len(paths))
    var firstHash []byte
    for i, p := range paths {
        bz, err := os.ReadFile(p)
        if err != nil {
            return nil, fmt.Errorf("read %s: %w", p, err)
        }
        var s SignedEIP712Tx
        if err := json.Unmarshal(bz, &s); err != nil {
            return nil, fmt.Errorf("parse %s: %w", p, err)
        }
        hash, _, err := apitypes.TypedDataAndHash(s.TypedData)
        if err != nil {
            return nil, fmt.Errorf("typed-data hash %s: %w", p, err)
        }
        if i == 0 {
            firstHash = hash
        }
        if !bytes.Equal(firstHash, hash) {
            return nil, fmt.Errorf("typed data mismatch between %s and first file", p)
        }
        out = append(out, s)
    }
    return out, nil
}

// mustParseAndNormalizeSig converts 0x r|s|v to 65-byte r|s|v with v in {27,28}.
func mustParseAndNormalizeSig(sigHex string) []byte {
    sigHex = strings.TrimPrefix(sigHex, "0x")
    b, err := hex.DecodeString(sigHex)
    if err != nil || len(b) != 65 {
        panic(fmt.Errorf("bad sig length: %v", err))
    }
    if b[64] < 27 {
        b[64] += 27
    }
    return b
}

// recoverCosmosEthPubKey recovers the ECDSA pubkey from the EIP-712 digest.
func recoverCosmosEthPubKey(td apitypes.TypedData, sig65 []byte) (sdkcryptotypes.PubKey, string, error) {
    hash, _, err := apitypes.TypedDataAndHash(td)
    if err != nil {
        return nil, "", err
    }
    pub, err := crypto.SigToPub(hash, sig65)
    if err != nil {
        return nil, "", fmt.Errorf("SigToPub: %w", err)
    }
    comp := crypto.CompressPubkey(pub)
    pk := &ethsecp.PubKey{Key: comp}
    ethAddr := crypto.PubkeyToAddress(*pub).Hex()
    return pk, ethAddr, nil
}

// fetchMultisigPubKey queries the auth module for the multisig account.
func fetchMultisigPubKey(ctx context.Context, cdc codec.Codec, nodeREST, multisigAddr string) (sdkcryptotypes.PubKey, authtypes.AccountI, error) {
    url := fmt.Sprintf("%s/cosmos/auth/v1beta1/accounts/%s", strings.TrimRight(nodeREST, "/"), multisigAddr)
    resp, err := http.Get(url)
    if err != nil {
        return nil, nil, err
    }
    defer resp.Body.Close()
    body, _ := io.ReadAll(resp.Body)

    var q struct {
        Account json.RawMessage `json:"account"`
    }
    if err := json.Unmarshal(body, &q); err != nil {
        return nil, nil, fmt.Errorf("unmarshal account wrapper: %w", err)
    }

    var acc authtypes.AccountI
    if err := cdc.UnmarshalInterfaceJSON(q.Account, &acc); err != nil {
        return nil, nil, fmt.Errorf("unmarshal account: %w", err)
    }
    pk := acc.GetPubKey()
    return pk, acc, nil
}

// buildMsgsFromTyped builds Cosmos msgs from the typedData message.
// This example only supports MsgDelegate; extend it as needed.
func buildMsgsFromTyped(s SignedEIP712Tx) ([]sdk.Msg, error) {
    var msgs []sdk.Msg
    for _, raw := range s.getMsgs() {
        var m struct {
            Type string          `json:"type"`
            Val  json.RawMessage `json:"value"`
        }
        if err := json.Unmarshal(raw, &m); err != nil {
            return nil, err
        }
        switch m.Type {
        case "/cosmos.staking.v1beta1.MsgDelegate", "cosmos-sdk/MsgDelegate":
            var v struct {
                Delegator string `json:"delegator_address"`
                Validator string `json:"validator_address"`
                Amount    struct {
                    Denom  string `json:"denom"`
                    Amount string `json:"amount"`
                } `json:"amount"`
            }
            if err := json.Unmarshal(m.Val, &v); err != nil {
                return nil, err
            }
            amt, ok := sdkmath.NewIntFromString(v.Amount.Amount)
            if !ok {
                return nil, fmt.Errorf("invalid amount: %s", v.Amount.Amount)
            }
            msgs = append(msgs, &stakingtypes.MsgDelegate{
                DelegatorAddress: v.Delegator,
                ValidatorAddress: v.Validator,
                Amount:           sdk.NewCoin(v.Amount.Denom, amt),
            })
        default:
            return nil, fmt.Errorf("unsupported msg type: %s", m.Type)
        }
    }
    return msgs, nil
}

// broadcastEIP712Multisig assembles and broadcasts a multisig EIP-712 tx.
func broadcastEIP712Multisig(
    ctx context.Context,
    txCfg client.TxConfig,
    cdc codec.Codec,
    nodeREST string,
    evmChainID uint64,
    cosmosChainID string,
    multisigAddr string,
    signedFiles []string,
) (string, error) {
    // 1) load and validate signer blobs
    signeds, err := loadSignedFiles(signedFiles)
    if err != nil {
        return "", err
    }
    base := signeds[0]

    // 2) msgs, fee, gas, memo
    msgs, err := buildMsgsFromTyped(base)
    if err != nil {
        return "", err
    }
    feeCoins, gas := base.getFee()
    fee := sdk.Coins(feeCoins)
    var gasLimit uint64
    if _, err := fmt.Sscan(gas, &gasLimit); err != nil {
        return "", fmt.Errorf("bad gas: %w", err)
    }
    memo := base.getMemo()

    // 3) multisig account
    msPubKey, acc, err := fetchMultisigPubKey(ctx, cdc, nodeREST, multisigAddr)
    if err != nil {
        return "", fmt.Errorf("fetch multisig: %w", err)
    }
    lm, ok := msPubKey.(*cryptomultisig.LegacyAminoPubKey)
    if !ok {
        return "", fmt.Errorf("account pubkey is not multisig (got %T)", msPubKey)
    }
    msKeys := lm.GetPubKeys()
    accountNum := acc.GetAccountNumber()
    sequence := acc.GetSequence()

    // 4) set up TxBuilder
    tb := txCfg.NewTxBuilder()
    extBuilder, ok := tb.(authtx.ExtensionOptionsTxBuilder)
    if !ok {
        return "", fmt.Errorf("tx builder does not support extensions")
    }
    
    if err := extBuilder.SetMsgs(msgs...); err != nil {
        return "", err
    }
    extBuilder.SetMemo(memo)
    extBuilder.SetGasLimit(gasLimit)
    extBuilder.SetFeeAmount(fee)

    // mark as EIP-712 via ExtensionOptionsWeb3Tx
    web3ext := &etherminttypes.ExtensionOptionsWeb3Tx{
        TypedDataChainID: evmChainID,
    }
    optAny, err := codectypes.NewAnyWithValue(web3ext)
    if err != nil {
        return "", err
    }
    extBuilder.SetExtensionOptions(optAny)

    // 5) combine signatures into MultiSignatureData
    multi := multisig.NewMultisig(len(msKeys))

    for _, s := range signeds {
        sig := mustParseAndNormalizeSig(s.Signature)
        pk, ethAddr, err := recoverCosmosEthPubKey(s.TypedData, sig)
        if err != nil {
            return "", fmt.Errorf("recover pubkey: %w", err)
        }

        // find signer index
        idx := -1
        for i, k := range msKeys {
            if bytes.Equal(k.Address().Bytes(), pk.Address().Bytes()) {
                idx = i
                break
            }
        }
        if idx < 0 {
            return "", fmt.Errorf("recovered %s not in multisig pubkeys", ethAddr)
        }

        sigV2 := signing.SignatureV2{
            PubKey: pk,
            Data: &signing.SingleSignatureData{
                SignMode:  signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON,
                Signature: sig,
            },
            Sequence: sequence,
        }

        if err := multisig.AddSignatureV2(multi, sigV2, msKeys); err != nil {
            return "", fmt.Errorf("add signature: %w", err)
        }
    }

    // 6) attach aggregated multisig signature
    agg := signing.SignatureV2{
        PubKey:   msPubKey,
        Data:     multi,
        Sequence: sequence,
    }
    if err := extBuilder.SetSignatures(agg); err != nil {
        return "", err
    }

    // 7) encode and broadcast
    txBytes, err := txCfg.TxEncoder()(extBuilder.GetTx())
    if err != nil {
        return "", err
    }

    txHash, err := broadcastREST(nodeREST, txBytes)
    if err != nil {
        return "", err
    }

    _ = accountNum // currently unused but kept for reference
    return txHash, nil
}

// broadcastREST broadcasts using the gRPC-gateway /cosmos/tx/v1beta1/txs endpoint.
func broadcastREST(node string, txBytes []byte) (string, error) {
    req := struct {
        TxBytes string `json:"tx_bytes"`
        Mode    string `json:"mode"`
    }{
        TxBytes: base64.StdEncoding.EncodeToString(txBytes),
        Mode:    "BROADCAST_MODE_SYNC",
    }

    bz, _ := json.Marshal(req)
    resp, err := http.Post(strings.TrimRight(node, "/")+"/cosmos/tx/v1beta1/txs", "application/json", bytes.NewReader(bz))
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    out, _ := io.ReadAll(resp.Body)
    return string(out), nil
}

// --- wiring / main ---

func buildEncodingConfig() (codec.Codec, client.TxConfig) {
    // Minimal encoding setup; in your actual code, reuse the chain's encoding config.
    interfaceRegistry := codectypes.NewInterfaceRegistry()

    sdk.RegisterInterfaces(interfaceRegistry)
    authtypes.RegisterInterfaces(interfaceRegistry)
    banktypes.RegisterInterfaces(interfaceRegistry)
    stakingtypes.RegisterInterfaces(interfaceRegistry)
    cryptocodec.RegisterInterfaces(interfaceRegistry)
    etherminttypes.RegisterInterfaces(interfaceRegistry)

    marshaler := codec.NewProtoCodec(interfaceRegistry)
    
    // Set up amino codec for legacy signing
    aminoCodec := codec.NewLegacyAmino()
    banktypes.RegisterLegacyAminoCodec(aminoCodec)
    stakingtypes.RegisterLegacyAminoCodec(aminoCodec)
    cryptocodec.RegisterCrypto(aminoCodec)
    legacytx.RegressionTestingAminoCodec = aminoCodec
    
    txCfg := authtx.NewTxConfig(marshaler, authtx.DefaultSignModes)

    return marshaler, txCfg
}

func main() {
    nodeREST := flag.String("node", "http://localhost:1317", "REST endpoint")
    evmChainID := flag.Uint64("evm-chain-id", 1, "EVM/TypedData chain id")
    cosmosChainID := flag.String("cosmos-chain-id", "chain-1", "Cosmos chain id (informational)")
    multisigAddr := flag.String("multisig", "", "multisig bech32 address (delegator)")
    flag.Parse()

    if *multisigAddr == "" {
        fmt.Println("--multisig is required")
        os.Exit(1)
    }
    if flag.NArg() == 0 {
        fmt.Println("provide one or more signed EIP-712 JSON files")
        os.Exit(1)
    }

    cdc, txCfg := buildEncodingConfig()
    ctx := context.Background()

    // ensure deterministic order of signers (not strictly required but nice)
    files := flag.Args()
    sort.Strings(files)

    hash, err := broadcastEIP712Multisig(ctx, txCfg, cdc, *nodeREST, *evmChainID, *cosmosChainID, *multisigAddr, files)
    if err != nil {
        fmt.Println("error:", err)
        os.Exit(1)
    }

    fmt.Println("tx response:")
    fmt.Println(hash)
}