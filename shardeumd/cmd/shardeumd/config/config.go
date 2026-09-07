package config

import (
	types "github.com/shardeum/shardeum-evm/crypto/hd"
	evmtypes "github.com/shardeum/shardeum-evm/x/vm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ChainsCoinInfo is a map of the chain id and its corresponding EvmCoinInfo
// that allows initializing the app with different coin info based on the
// chain id
var ChainsCoinInfo = map[uint64]evmtypes.EvmCoinInfo{}

// InitializeChainsCoinInfo initializes the ChainsCoinInfo map with network configurations
func InitializeChainsCoinInfo() {
	// Initialize static test chain IDs first with safe fallbacks
	ChainsCoinInfo[EighteenDecimalsChainID] = evmtypes.EvmCoinInfo{
		Denom:         GetSafeDenom(),
		ExtendedDenom: GetSafeDenom(),
		DisplayDenom:  GetSafeDisplayDenom(),
		Decimals:      evmtypes.EighteenDecimals.Uint32(),
	}

	ChainsCoinInfo[SixDecimalsChainID] = evmtypes.EvmCoinInfo{
		Denom:         "utest",
		ExtendedDenom: "atest",
		DisplayDenom:  "test",
		Decimals:      evmtypes.SixDecimals.Uint32(),
	}

	ChainsCoinInfo[TwelveDecimalsChainID] = evmtypes.EvmCoinInfo{
		Denom:         GetSafeDenom(),
		ExtendedDenom: GetSafeDenom(),
		DisplayDenom:  GetSafeDisplayDenom(),
		Decimals:      evmtypes.TwelveDecimals.Uint32(),
	}

	ChainsCoinInfo[TwoDecimalsChainID] = evmtypes.EvmCoinInfo{
		Denom:         GetSafeDenom(),
		ExtendedDenom: GetSafeDenom(),
		DisplayDenom:  GetSafeDisplayDenom(),
		Decimals:      evmtypes.TwoDecimals.Uint32(),
	}

	ChainsCoinInfo[TestChainID1] = evmtypes.EvmCoinInfo{
		Denom:         GetSafeDenom(),
		ExtendedDenom: GetSafeDenom(),
		DisplayDenom:  GetSafeDisplayDenom(),
		Decimals:      evmtypes.EighteenDecimals.Uint32(),
	}

	ChainsCoinInfo[TestChainID2] = evmtypes.EvmCoinInfo{
		Denom:         GetSafeDenom(),
		ExtendedDenom: GetSafeDenom(),
		DisplayDenom:  GetSafeDisplayDenom(),
		Decimals:      evmtypes.EighteenDecimals.Uint32(),
	}

	// Update with network configurations from JSON files
	UpdateChainsCoinInfo()
}

// GetSafeDenom returns a safe denomination that won't panic during initialization
func GetSafeDenom() string {
	config, err := GetNetworkConfig("local")
	if err != nil {
		// Fallback to default if config files aren't available
		return "ashm"
	}
	return config.BaseDenom
}

// GetSafeDisplayDenom returns a safe display denomination that won't panic during initialization
func GetSafeDisplayDenom() string {
	config, err := GetNetworkConfig("local")
	if err != nil {
		// Fallback to default if config files aren't available
		return "shm"
	}
	return config.DisplayDenom
}

// GetBech32Prefix returns the Bech32 prefix for the current network
func GetBech32Prefix() string {
	return GetDefaultNetworkConfig().Bech32Prefix
}

// GetBech32PrefixAccAddr returns the Bech32 prefix of an account's address for the current network
func GetBech32PrefixAccAddr() string {
	return GetBech32Prefix()
}

// GetBech32PrefixAccPub returns the Bech32 prefix of an account's public key for the current network
func GetBech32PrefixAccPub() string {
	return GetBech32Prefix() + sdk.PrefixPublic
}

// GetBech32PrefixValAddr returns the Bech32 prefix of a validator's operator address for the current network
func GetBech32PrefixValAddr() string {
	return GetBech32Prefix() + sdk.PrefixValidator + sdk.PrefixOperator
}

// GetBech32PrefixValPub returns the Bech32 prefix of a validator's operator public key for the current network
func GetBech32PrefixValPub() string {
	return GetBech32Prefix() + sdk.PrefixValidator + sdk.PrefixOperator + sdk.PrefixPublic
}

// GetBech32PrefixConsAddr returns the Bech32 prefix of a consensus node address for the current network
func GetBech32PrefixConsAddr() string {
	return GetBech32Prefix() + sdk.PrefixValidator + sdk.PrefixConsensus
}

// GetBech32PrefixConsPub returns the Bech32 prefix of a consensus node public key for the current network
func GetBech32PrefixConsPub() string {
	return GetBech32Prefix() + sdk.PrefixValidator + sdk.PrefixConsensus + sdk.PrefixPublic
}

// GetDisplayDenom returns the denomination displayed to users in client applications for the current network
func GetDisplayDenom() string {
	return GetDefaultNetworkConfig().DisplayDenom
}

// GetBaseDenom returns the default denomination used in the chain for the current network
func GetBaseDenom() string {
	return GetDefaultNetworkConfig().BaseDenom
}

// GetBaseDenomUnit returns the precision of the base denomination for the current network
func GetBaseDenomUnit() int {
	return int(GetDefaultNetworkConfig().Decimals)
}

// GetEVMChainID returns the EIP-155 replay-protection chain id for the current network
func GetEVMChainID() uint64 {
	return GetDefaultNetworkConfig().EVMChainID
}

// Legacy constants for backward compatibility
const (
	// BaseDenomUnit defines the precision of the base denomination.
	BaseDenomUnit = 18
)

// SetBech32Prefixes sets the global prefixes to be used when serializing addresses and public keys to Bech32 strings.
func SetBech32Prefixes(config *sdk.Config) {
	config.SetBech32PrefixForAccount(GetBech32PrefixAccAddr(), GetBech32PrefixAccPub())
	config.SetBech32PrefixForValidator(GetBech32PrefixValAddr(), GetBech32PrefixValPub())
	config.SetBech32PrefixForConsensusNode(GetBech32PrefixConsAddr(), GetBech32PrefixConsPub())
}

// SetBip44CoinType sets the global coin type to be used in hierarchical deterministic wallets.
func SetBip44CoinType(config *sdk.Config) {
	config.SetCoinType(types.Bip44CoinType)
	config.SetPurpose(sdk.Purpose)                  // Shared
	config.SetFullFundraiserPath(types.BIP44HDPath) //nolint: staticcheck
}
