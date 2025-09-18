package config

const (
	// ShardeumBaseDenom is the base denomination of the Shardeum chain's coin (6 decimals for Cosmos).
	ShardeumBaseDenom = "shm"

	// ShardeumExtendedDenom is the extended denomination for EVM operations (18 decimals).
	ShardeumExtendedDenom = "ashm"

	// ShardeumDisplayDenom is the display denomination of the Shardeum chain's base coin.
	ShardeumDisplayDenom = "shm"

	// ShardeumChainID is the chain ID for Shardeum EVM chain.
	ShardeumChainID = 8119

	// DefaultChainID is the default Shardeum chain ID for testing
	DefaultChainID = "shardeum_8119-1"
)
