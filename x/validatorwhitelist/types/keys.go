package types

const (
	// ModuleName defines the module name
	ModuleName = "validatorwhitelist"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName
)

var (
	// ParamsKey is the key for the validator whitelist parameters
	ParamsKey = []byte{0x01}
)
