package config

import (
	"os"
)

const (
	// EighteenDecimalsChainID is the chain ID for the 18 decimals chain.
	EighteenDecimalsChainID = 9001

	// SixDecimalsChainID is the chain ID for the 6 decimals chain.
	SixDecimalsChainID = 9002

	// TwelveDecimalsChainID is the chain ID for the 12 decimals chain.
	TwelveDecimalsChainID = 9003

	// TwoDecimalsChainID is the chain ID for the 2 decimals chain.
	TwoDecimalsChainID = 9004

	CosmosChainID = 262144

	// TestChainID1 is test chain IDs for IBC E2E test
	TestChainID1 = 9005
	// TestChainID2 is test chain IDs for IBC E2E test
	TestChainID2 = 9006

	// DefaultNetworkName is the default network to use when none is specified
	DefaultNetworkName = "local"
)

// GetDefaultNetworkConfig returns the configuration for the default network
func GetDefaultNetworkConfig() NetworkConfig {
	networkName := DefaultNetworkName
	if envNetwork := os.Getenv("SHARDEUM_NETWORK"); envNetwork != "" {
		networkName = envNetwork
	}
	
	config, err := GetNetworkConfig(networkName)
	if err != nil {
		// Fallback to local configuration if default fails
		config, err = GetNetworkConfig("local")
		if err != nil {
			// Panic as this indicates missing config files
			panic("No network configuration found - ensure configs/local.json exists")
		}
	}
	return config
}

// ShardeumChainDenom returns the denomination of the Shardeum chain's base coin for the current network
func ShardeumChainDenom() string {
	return GetDefaultNetworkConfig().BaseDenom
}

// ShardeumDisplayDenom returns the display denomination of the Shardeum chain's base coin for the current network
func ShardeumDisplayDenom() string {
	return GetDefaultNetworkConfig().DisplayDenom
}

// ShardeumChainID returns the chain ID for Shardeum EVM chain for the current network
func ShardeumChainID() uint64 {
	return GetDefaultNetworkConfig().EVMChainID
}

// DefaultChainID returns the default Shardeum chain ID for the current network
func DefaultChainID() string {
	return GetDefaultNetworkConfig().ChainID
}
