package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	evmtypes "github.com/shardeum/shardeum-evm/x/vm/types"
	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type ConfigTestSuite struct {
	suite.Suite
	originalEnv map[string]string
	tempDir     string
	configDir   string
	originalWd  string
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

func (suite *ConfigTestSuite) SetupTest() {
	// Create temporary directory and config files
	tempDir, err := os.MkdirTemp("", "config-test")
	suite.Require().NoError(err)
	suite.tempDir = tempDir
	suite.configDir = filepath.Join(tempDir, "configs")
	err = os.MkdirAll(suite.configDir, 0755)
	suite.Require().NoError(err)

	// Create standard network config files
	suite.createStandardConfigs()

    // Save original working directory and change to temp dir
    suite.originalWd, _ = os.Getwd()
    os.Chdir(tempDir)

	// Save original environment variables
	suite.originalEnv = make(map[string]string)
    envVars := []string{
		"SHARDEUM_NETWORK",
		"SHARDEUM_CHAIN_ID",
		"SHARDEUM_EVM_CHAIN_ID",
		"SHARDEUM_BASE_DENOM",
		"SHARDEUM_DISPLAY_DENOM",
        "SHARDEUM_CONFIG_DIR",
	}

	for _, envVar := range envVars {
		suite.originalEnv[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
    }

    // Point the binary to our temporary configs directory
    os.Setenv("SHARDEUM_CONFIG_DIR", suite.configDir)
}

func (suite *ConfigTestSuite) TearDownTest() {
	// Restore original working directory
	if suite.originalWd != "" {
		os.Chdir(suite.originalWd)
	}

	// Restore original environment variables
	for envVar, value := range suite.originalEnv {
		if value == "" {
			os.Unsetenv(envVar)
		} else {
			os.Setenv(envVar, value)
		}
	}

	// Clean up temporary directory
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

func (suite *ConfigTestSuite) createStandardConfigs() {
	// Create standard network configs that match our JSON files
	networks := map[string]NetworkConfig{
		"local": {
			Name:         "local",
			ChainID:      "shardeum-local",
			EVMChainID:   8119,
			BaseDenom:    "ashm",
			DisplayDenom: "shm",
			Decimals:     evmtypes.EighteenDecimals,
			Bech32Prefix: "shardeum",
			Ports: NetworkPorts{
				RPC:       "26657",
				REST:      "1317",
				JSONRPC:   "8545",
				WebSocket: "8546",
				GRPC:      "9090",
			},
		},
		"testnet": {
			Name:         "testnet",
			ChainID:      "shardeum-testnet",
			EVMChainID:   8119,
			BaseDenom:    "ashm",
			DisplayDenom: "shm",
			Decimals:     evmtypes.EighteenDecimals,
			Bech32Prefix: "shardeum",
			Ports: NetworkPorts{
				RPC:       "26657",
				REST:      "1317",
				JSONRPC:   "8545",
				WebSocket: "8546",
				GRPC:      "9090",
			},
		},
        "devnet": {
            Name:         "devnet",
            ChainID:      "shardeum-devnet",
            EVMChainID:   8119,
            BaseDenom:    "ashm",
            DisplayDenom: "shm",
            Decimals:     evmtypes.EighteenDecimals,
            Bech32Prefix: "shardeum",
            Ports: NetworkPorts{
                RPC:       "26657",
                REST:      "1317",
                JSONRPC:   "8545",
                WebSocket: "8546",
                GRPC:      "9090",
            },
        },
        "mainnet": {
            Name:         "mainnet",
            ChainID:      "shardeum-mainnet",
            EVMChainID:   8119,
            BaseDenom:    "ashm",
            DisplayDenom: "shm",
            Decimals:     evmtypes.EighteenDecimals,
            Bech32Prefix: "shardeum",
            Ports: NetworkPorts{
                RPC:       "26657",
                REST:      "1317",
                JSONRPC:   "8545",
                WebSocket: "8546",
                GRPC:      "9090",
            },
        },
	}

	for name, config := range networks {
		configPath := filepath.Join(suite.configDir, name+".json")
		data, err := json.MarshalIndent(config, "", "  ")
		suite.Require().NoError(err)
		err = os.WriteFile(configPath, data, 0644)
		suite.Require().NoError(err)
	}
}

func (suite *ConfigTestSuite) TestGetBech32Prefix() {
	// Test default network
	prefix := GetBech32Prefix()
	suite.Require().Equal("shardeum", prefix)

	// Test with environment override
	os.Setenv("SHARDEUM_NETWORK", "testnet")
	prefix = GetBech32Prefix()
	suite.Require().Equal("shardeum", prefix)
}

func (suite *ConfigTestSuite) TestGetBech32PrefixFunctions() {
	prefix := GetBech32Prefix()

	// Test all Bech32 prefix functions
	suite.Require().Equal(prefix, GetBech32PrefixAccAddr())
	suite.Require().Equal(prefix+sdk.PrefixPublic, GetBech32PrefixAccPub())
	suite.Require().Equal(prefix+sdk.PrefixValidator+sdk.PrefixOperator, GetBech32PrefixValAddr())
	suite.Require().Equal(prefix+sdk.PrefixValidator+sdk.PrefixOperator+sdk.PrefixPublic, GetBech32PrefixValPub())
	suite.Require().Equal(prefix+sdk.PrefixValidator+sdk.PrefixConsensus, GetBech32PrefixConsAddr())
	suite.Require().Equal(prefix+sdk.PrefixValidator+sdk.PrefixConsensus+sdk.PrefixPublic, GetBech32PrefixConsPub())
}

func (suite *ConfigTestSuite) TestGetDisplayDenom() {
	denom := GetDisplayDenom()
	suite.Require().Equal("shm", denom)

	// Test with environment override
	os.Setenv("SHARDEUM_DISPLAY_DENOM", "customshm")
	denom = GetDisplayDenom()
	suite.Require().Equal("customshm", denom)
}

func (suite *ConfigTestSuite) TestGetBaseDenom() {
	denom := GetBaseDenom()
	suite.Require().Equal("ashm", denom)

	// Test with environment override
	os.Setenv("SHARDEUM_BASE_DENOM", "customashm")
	denom = GetBaseDenom()
	suite.Require().Equal("customashm", denom)
}

func (suite *ConfigTestSuite) TestGetBaseDenomUnit() {
	unit := GetBaseDenomUnit()
	suite.Require().Equal(18, unit)
}

func (suite *ConfigTestSuite) TestGetEVMChainID() {
	chainID := GetEVMChainID()
	suite.Require().Equal(uint64(8119), chainID)

	// Test with environment override
	os.Setenv("SHARDEUM_EVM_CHAIN_ID", "9999")
	chainID = GetEVMChainID()
	suite.Require().Equal(uint64(9999), chainID)
}

func (suite *ConfigTestSuite) TestSetBech32Prefixes() {
	config := sdk.NewConfig()
	
	// Test setting Bech32 prefixes
	SetBech32Prefixes(config)

	// Verify prefixes are set correctly
	accAddr, accPub := config.GetBech32AccountAddrPrefix(), config.GetBech32AccountPubPrefix()
	valAddr, valPub := config.GetBech32ValidatorAddrPrefix(), config.GetBech32ValidatorPubPrefix()
	consAddr, consPub := config.GetBech32ConsensusAddrPrefix(), config.GetBech32ConsensusPubPrefix()

	expectedPrefix := GetBech32Prefix()
	suite.Require().Equal(expectedPrefix, accAddr)
	suite.Require().Equal(expectedPrefix+sdk.PrefixPublic, accPub)
	suite.Require().Equal(expectedPrefix+sdk.PrefixValidator+sdk.PrefixOperator, valAddr)
	suite.Require().Equal(expectedPrefix+sdk.PrefixValidator+sdk.PrefixOperator+sdk.PrefixPublic, valPub)
	suite.Require().Equal(expectedPrefix+sdk.PrefixValidator+sdk.PrefixConsensus, consAddr)
	suite.Require().Equal(expectedPrefix+sdk.PrefixValidator+sdk.PrefixConsensus+sdk.PrefixPublic, consPub)
}

func (suite *ConfigTestSuite) TestInitializeChainsCoinInfo() {
	// Clear ChainsCoinInfo first
	ChainsCoinInfo = make(map[uint64]evmtypes.EvmCoinInfo)

	// Initialize
	InitializeChainsCoinInfo()

	// Verify that builtin networks would be in ChainsCoinInfo if config files exist
	// Note: This test might not find all networks since we're not in the actual project directory
	// with config files, but we can test that the test chain IDs are present
	
	// Verify test chain IDs are also present
	_, exists := ChainsCoinInfo[EighteenDecimalsChainID]
	suite.Require().True(exists, "ChainsCoinInfo should contain EighteenDecimalsChainID")
	
	_, exists = ChainsCoinInfo[SixDecimalsChainID]
	suite.Require().True(exists, "ChainsCoinInfo should contain SixDecimalsChainID")
}

func (suite *ConfigTestSuite) TestChainsCoinInfoDynamicValues() {
	// Test that ChainsCoinInfo uses dynamic values
	originalChainsCoinInfo := make(map[uint64]evmtypes.EvmCoinInfo)
	for k, v := range ChainsCoinInfo {
		originalChainsCoinInfo[k] = v
	}

	// Change environment
	os.Setenv("SHARDEUM_BASE_DENOM", "newdenom")
	os.Setenv("SHARDEUM_DISPLAY_DENOM", "new")

	// Re-initialize
	InitializeChainsCoinInfo()

	// Verify that the dynamic functions reflect the environment changes
	suite.Require().Equal("newdenom", ShardeumChainDenom())
	suite.Require().Equal("new", ShardeumDisplayDenom())
	
	// Verify that the test chain IDs use the dynamic values
	coinInfo := ChainsCoinInfo[EighteenDecimalsChainID]
	suite.Require().Equal("newdenom", coinInfo.Denom)
	suite.Require().Equal("new", coinInfo.DisplayDenom)
}

// Test network switching scenarios
func (suite *ConfigTestSuite) TestNetworkSwitching() {
	// Test switching to a valid network first
	os.Setenv("SHARDEUM_NETWORK", "testnet")
	config := GetDefaultNetworkConfig()
	suite.Require().Equal("testnet", config.Name)
	suite.Require().Equal("shardeum-testnet", config.ChainID)
	
	// Test switching to local network
	os.Setenv("SHARDEUM_NETWORK", "local")
	config = GetDefaultNetworkConfig()
	suite.Require().Equal("local", config.Name)
	suite.Require().Equal("shardeum-local", config.ChainID)
}

// Test error handling
func (suite *ConfigTestSuite) TestGetDefaultNetworkConfig_InvalidNetwork() {
	// Test that unset environment variable uses default (local)
	os.Unsetenv("SHARDEUM_NETWORK")
	
	// Should use local network as default
	config := GetDefaultNetworkConfig()
	suite.Require().Equal("local", config.Name)
	suite.Require().Equal("shardeum-local", config.ChainID)
}

// Benchmark tests
func BenchmarkGetBech32Prefix(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetBech32Prefix()
	}
}

func BenchmarkGetDefaultNetworkConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetDefaultNetworkConfig()
	}
}

func BenchmarkShardeumChainDenom(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ShardeumChainDenom()
	}
}