package constants

import (
	erc20types "github.com/shardeum/shardeum-evm/x/erc20/types"
	evmtypes "github.com/shardeum/shardeum-evm/x/vm/types"

	"cosmossdk.io/math"
)

const (
	// DefaultGasPrice is used in testing as the default to use for transactions
	DefaultGasPrice = 20

	// ShardeumAttoDenom provides the Shardeum atto denom for use in tests
	ShardeumAttoDenom = "ashm"

	// ShardeumMicroDenom provides the Shardeum micro denom for use in tests
	ShardeumMicroDenom = "ushm"

	// ShardeumDisplayDenom provides the Shardeum display denom for use in tests
	ShardeumDisplayDenom = "shm"

	// ExampleBech32Prefix provides an example Bech32 prefix for use in tests
	ExampleBech32Prefix = "cosmos"

	// ExampleEIP155ChainID provides an example EIP-155 chain ID for use in tests
	ExampleEIP155ChainID = 9001

	// ShardeumEIP155ChainID provides the Shardeum EIP-155 chain ID for use in tests
	ShardeumEIP155ChainID = 8119

	// ShardeumMainChainID is the Shardeum main chain ID
	ShardeumMainChainID = 8119
	// ExampleEvmAddress1 is the example EVM address
	ExampleEvmAddressAlice = "0x1e0DE5DB1a39F99cBc67B00fA3415181b3509e42"
	// ExampleEvmAddress2 is the example EVM address
	ExampleEvmAddressBob = "0x0AFc8e15F0A74E98d0AEC6C67389D2231384D4B2"
)

type ChainID struct {
	ChainID    string `json:"chain_id"`
	EVMChainID uint64 `json:"evm_chain_id"`
}

var (
	// ExampleChainIDPrefix provides a chain ID prefix for EIP-155 that can be used in tests
	ExampleChainIDPrefix = "cosmos"

	// ExampleChainID provides a chain ID that can be used in tests
	ExampleChainID = ChainID{
		ChainID:    ExampleChainIDPrefix + "-1",
		EVMChainID: 9001,
	}

	// ShardeumChainID provides the Shardeum chain ID for use in tests
	ShardeumChainID = ChainID{
		ChainID:    "shardeum_8119-1",
		EVMChainID: 8119,
	}

	// SixDecimalsChainID provides a chain ID which is being set up with 6 decimals
	SixDecimalsChainID = ChainID{
		ChainID:    "ossix-2",
		EVMChainID: 9002,
	}

	// TwelveDecimalsChainID provides a chain ID which is being set up with 12 decimals
	TwelveDecimalsChainID = ChainID{
		ChainID:    "ostwelve-3",
		EVMChainID: 9003,
	}

	// TwoDecimalsChainID provides a chain ID which is being set up with 2 decimals
	TwoDecimalsChainID = ChainID{
		ChainID:    "ostwo-4",
		EVMChainID: 9004,
	}

	// ExampleChainCoinInfo provides the coin info for the example chain
	//
	// It is a map of the chain id and its corresponding EvmCoinInfo
	// that allows initializing the app with different coin info based on the
	// chain id
	ExampleChainCoinInfo = map[ChainID]evmtypes.EvmCoinInfo{
		ExampleChainID: {
			Denom:         ShardeumAttoDenom,
			ExtendedDenom: ShardeumAttoDenom,
			DisplayDenom:  ShardeumDisplayDenom,
			Decimals:      evmtypes.EighteenDecimals.Uint32(),
		},
		SixDecimalsChainID: {
			Denom:         "utest",
			ExtendedDenom: "atest",
			DisplayDenom:  "test",
			Decimals:      evmtypes.SixDecimals.Uint32(),
		},
		TwelveDecimalsChainID: {
			Denom:         "ptest2",
			ExtendedDenom: "atest2",
			DisplayDenom:  "test2",
			Decimals:      evmtypes.TwelveDecimals.Uint32(),
		},
		TwoDecimalsChainID: {
			Denom:         "ctest3",
			ExtendedDenom: "atest3",
			DisplayDenom:  "test3",
			Decimals:      evmtypes.TwoDecimals.Uint32(),
		},
	}

	// OtherCoinDenoms provides a list of other coin denoms that can be used in tests
	OtherCoinDenoms = []string{
		"foo",
		"bar",
	}

	// ExampleTokenPairs creates a slice of token pairs, that contains a pair for the native denom of the example chain
	// implementation.
	ExampleTokenPairs = []erc20types.TokenPair{
		{
			Erc20Address:  "0x0000000000000000000000000000000000000000",
			Denom:         ShardeumAttoDenom,
			Enabled:       true,
			ContractOwner: erc20types.OWNER_MODULE,
		},
	}

	// ExampleAllowances creates a slice of allowances, that contains an allowance for the native denom of the example chain
	// implementation.
	ExampleAllowances = []erc20types.Allowance{
		{
			Erc20Address: "0x0000000000000000000000000000000000000000",
			Owner:        ExampleEvmAddressAlice,
			Spender:      ExampleEvmAddressBob,
			Value:        math.NewInt(100),
		},
	}
)
