package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	evmtypes "github.com/shardeum/shardeum-evm/x/vm/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type NetworkConfigTestSuite struct {
	suite.Suite
	tempDir    string
	configDir  string
	originalEnv map[string]string
}

func TestNetworkConfigTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkConfigTestSuite))
}

func (suite *NetworkConfigTestSuite) SetupTest() {
	// Create temporary directory for test configs
	tempDir, err := os.MkdirTemp("", "network-config-test")
	suite.Require().NoError(err)
	suite.tempDir = tempDir
	suite.configDir = filepath.Join(tempDir, "configs")
	err = os.MkdirAll(suite.configDir, 0755)
	suite.Require().NoError(err)

	// Create standard network config files for testing
	suite.createStandardNetworkConfigs()

	// Save original environment variables
	suite.originalEnv = make(map[string]string)
	envVars := []string{
		"SHARDEUM_NETWORK",
		"SHARDEUM_CHAIN_ID",
		"SHARDEUM_EVM_CHAIN_ID",
		"SHARDEUM_BASE_DENOM",
		"SHARDEUM_DISPLAY_DENOM",
		"SHARDEUM_RPC_PORT",
		"SHARDEUM_REST_PORT",
		"SHARDEUM_JSON_RPC_PORT",
		"SHARDEUM_WEBSOCKET_PORT",
		"SHARDEUM_GRPC_PORT",
	}

	for _, envVar := range envVars {
		suite.originalEnv[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}

	// Change working directory to temp dir for tests
	originalWd, _ := os.Getwd()
	os.Chdir(tempDir)
	suite.T().Cleanup(func() {
		os.Chdir(originalWd)
	})
}

func (suite *NetworkConfigTestSuite) TearDownTest() {
	// Restore original environment variables
	for envVar, value := range suite.originalEnv {
		if value == "" {
			os.Unsetenv(envVar)
		} else {
			os.Setenv(envVar, value)
		}
	}

	// Clean up temporary directory
	os.RemoveAll(suite.tempDir)
}

func (suite *NetworkConfigTestSuite) createTestConfig(name string, config NetworkConfig) {
	configPath := filepath.Join(suite.configDir, name+".json")
	data, err := json.MarshalIndent(config, "", "  ")
	suite.Require().NoError(err)
	err = os.WriteFile(configPath, data, 0644)
	suite.Require().NoError(err)
}

func (suite *NetworkConfigTestSuite) createStandardNetworkConfigs() {
	// Create standard network config files that match the JSON files
	networks := map[string]NetworkConfig{
		"mainnet": {
			Name:         "mainnet",
			ChainID:      "shardeum-1",
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
			GenesisFile: "mainnet-genesis.json",
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
			GenesisFile: "testnet-genesis.json",
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
			GenesisFile: "devnet-genesis.json",
		},
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
			GenesisFile: "local-genesis.json",
		},
	}

	for name, config := range networks {
		suite.createTestConfig(name, config)
	}
}

func (suite *NetworkConfigTestSuite) TestBuiltinNetworks() {
	// Test that all builtin networks can be loaded from JSON files
	expectedNetworks := []string{"mainnet", "testnet", "devnet", "local"}
	
	for _, network := range expectedNetworks {
		config, err := GetNetworkConfig(network)
		suite.Require().NoError(err, "Builtin network %s should be loadable", network)
		suite.Require().Equal(network, config.Name)
		suite.Require().NotEmpty(config.ChainID)
		suite.Require().NotZero(config.EVMChainID)
		suite.Require().NotEmpty(config.BaseDenom)
		suite.Require().NotEmpty(config.DisplayDenom)
		suite.Require().NotEmpty(config.Bech32Prefix)
		suite.Require().NotEmpty(config.Ports.RPC)
		suite.Require().NotEmpty(config.Ports.REST)
		suite.Require().NotEmpty(config.Ports.JSONRPC)
		suite.Require().NotEmpty(config.Ports.WebSocket)
		suite.Require().NotEmpty(config.Ports.GRPC)
	}
}

func (suite *NetworkConfigTestSuite) TestGetNetworkConfig_Builtin() {
	// Test getting builtin network
	config, err := GetNetworkConfig("testnet")
	suite.Require().NoError(err)
	suite.Require().Equal("testnet", config.Name)
	suite.Require().Equal("shardeum-testnet", config.ChainID)
	suite.Require().Equal(uint64(8119), config.EVMChainID)
	suite.Require().Equal("ashm", config.BaseDenom)
	suite.Require().Equal("shm", config.DisplayDenom)
	suite.Require().Equal("shardeum", config.Bech32Prefix)
}

func (suite *NetworkConfigTestSuite) TestGetNetworkConfig_CustomFile() {
	// Create custom network config
	customConfig := NetworkConfig{
		Name:         "custom",
		ChainID:      "custom-chain",
		EVMChainID:   9999,
		BaseDenom:    "custom",
		DisplayDenom: "cust",
		Decimals:     evmtypes.EighteenDecimals,
		Bech32Prefix: "custom",
		Ports: NetworkPorts{
			RPC:       "26657",
			REST:      "1317",
			JSONRPC:   "8545",
			WebSocket: "8546",
			GRPC:      "9090",
		},
		GenesisFile: "custom-genesis.json",
	}
	suite.createTestConfig("custom", customConfig)

	// Test getting custom network
	config, err := GetNetworkConfig("custom")
	suite.Require().NoError(err)
	suite.Require().Equal("custom", config.Name)
	suite.Require().Equal("custom-chain", config.ChainID)
	suite.Require().Equal(uint64(9999), config.EVMChainID)
	suite.Require().Equal("custom", config.BaseDenom)
	suite.Require().Equal("cust", config.DisplayDenom)
	suite.Require().Equal("custom", config.Bech32Prefix)
}

func (suite *NetworkConfigTestSuite) TestGetNetworkConfig_NotFound() {
	// Test getting non-existent network
	_, err := GetNetworkConfig("nonexistent")
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "network 'nonexistent' not found")
}

func (suite *NetworkConfigTestSuite) TestGetNetworkConfig_EnvironmentOverride() {
	// Set environment variable to override network
	os.Setenv("SHARDEUM_NETWORK", "mainnet")

	// Request testnet but should get mainnet due to env override
	config, err := GetNetworkConfig("testnet")
	suite.Require().NoError(err)
	suite.Require().Equal("mainnet", config.Name)
	suite.Require().Equal("shardeum-1", config.ChainID)
}

func (suite *NetworkConfigTestSuite) TestApplyEnvOverrides() {
	// Get testnet config from JSON file
	config, err := GetNetworkConfig("testnet")
	suite.Require().NoError(err)

	// Set environment overrides
	os.Setenv("SHARDEUM_CHAIN_ID", "custom-chain-id")
	os.Setenv("SHARDEUM_EVM_CHAIN_ID", "7777")
	os.Setenv("SHARDEUM_BASE_DENOM", "customdenom")
	os.Setenv("SHARDEUM_DISPLAY_DENOM", "custom")
	os.Setenv("SHARDEUM_RPC_PORT", "27657")
	os.Setenv("SHARDEUM_REST_PORT", "2317")
	os.Setenv("SHARDEUM_JSON_RPC_PORT", "9545")
	os.Setenv("SHARDEUM_WEBSOCKET_PORT", "9546")
	os.Setenv("SHARDEUM_GRPC_PORT", "10090")

	overriddenConfig := applyEnvOverrides(config)

	suite.Require().Equal("custom-chain-id", overriddenConfig.ChainID)
	suite.Require().Equal(uint64(7777), overriddenConfig.EVMChainID)
	suite.Require().Equal("customdenom", overriddenConfig.BaseDenom)
	suite.Require().Equal("custom", overriddenConfig.DisplayDenom)
	suite.Require().Equal("27657", overriddenConfig.Ports.RPC)
	suite.Require().Equal("2317", overriddenConfig.Ports.REST)
	suite.Require().Equal("9545", overriddenConfig.Ports.JSONRPC)
	suite.Require().Equal("9546", overriddenConfig.Ports.WebSocket)
	suite.Require().Equal("10090", overriddenConfig.Ports.GRPC)
}

func (suite *NetworkConfigTestSuite) TestApplyEnvOverrides_InvalidValues() {
	// Get testnet config from JSON file
	config, err := GetNetworkConfig("testnet")
	suite.Require().NoError(err)

	// Set invalid EVM chain ID (should be ignored)
	os.Setenv("SHARDEUM_EVM_CHAIN_ID", "invalid")

	overriddenConfig := applyEnvOverrides(config)

	// Should keep original value since invalid
	suite.Require().Equal(config.EVMChainID, overriddenConfig.EVMChainID)
}

func (suite *NetworkConfigTestSuite) TestGetAvailableNetworks() {
	// Create some custom configs
	customConfigs := []string{"custom1", "custom2", "custom3"}
	for _, name := range customConfigs {
		suite.createTestConfig(name, NetworkConfig{
			Name:    name,
			ChainID: name + "-chain",
		})
	}

	networks := GetAvailableNetworks()

	// Should include builtin networks (from JSON files)
	suite.Require().Contains(networks, "mainnet")
	suite.Require().Contains(networks, "testnet")
	suite.Require().Contains(networks, "devnet")
	suite.Require().Contains(networks, "local")

	// Should include custom networks
	for _, custom := range customConfigs {
		suite.Require().Contains(networks, custom)
	}
}

func (suite *NetworkConfigTestSuite) TestParseUint64() {
	testCases := []struct {
		input    string
		expected uint64
		hasError bool
	}{
		{"123", 123, false},
		{"0", 0, false},
		{"9999", 9999, false},
		{"invalid", 0, true},
		{"123abc", 0, true},
		{"", 0, false},
		{"-123", 0, true},
	}

	for _, tc := range testCases {
		result, err := parseUint64(tc.input)
		if tc.hasError {
			suite.Require().Error(err, "Expected error for input: %s", tc.input)
		} else {
			suite.Require().NoError(err, "Expected no error for input: %s", tc.input)
			suite.Require().Equal(tc.expected, result, "Expected %d for input: %s", tc.expected, tc.input)
		}
	}
}

func (suite *NetworkConfigTestSuite) TestUpdateChainsCoinInfo() {
	// Call UpdateChainsCoinInfo
	UpdateChainsCoinInfo()

	// Verify that ChainsCoinInfo has been updated with builtin networks
	builtinNetworks := getBuiltinNetworks()
	for _, networkName := range builtinNetworks {
		config, err := GetNetworkConfig(networkName)
		suite.Require().NoError(err)
		
		coinInfo, exists := ChainsCoinInfo[config.EVMChainID]
		suite.Require().True(exists, "ChainsCoinInfo should contain EVM chain ID %d", config.EVMChainID)
		suite.Require().Equal(config.BaseDenom, coinInfo.Denom)
		suite.Require().Equal(config.BaseDenom, coinInfo.ExtendedDenom)
		suite.Require().Equal(config.DisplayDenom, coinInfo.DisplayDenom)
		suite.Require().Equal(config.Decimals, coinInfo.Decimals)
	}
}

func (suite *NetworkConfigTestSuite) TestLoadNetworkConfigFromFile_InvalidJSON() {
	// Create invalid JSON file
	invalidConfigPath := filepath.Join(suite.configDir, "invalid.json")
	err := os.WriteFile(invalidConfigPath, []byte("invalid json"), 0644)
	suite.Require().NoError(err)

	// Should return error for invalid JSON
	_, err = loadNetworkConfigFromFile(invalidConfigPath)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "failed to parse network config file")
}

func (suite *NetworkConfigTestSuite) TestLoadNetworkConfigFromFile_NonExistent() {
	// Should return error for non-existent file
	_, err := loadNetworkConfigFromFile("/nonexistent/path.json")
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "failed to read network config file")
}

// Test the constants and helper functions
func TestNetworkConstants(t *testing.T) {
	// Create temporary directory and config files for this test
	tempDir, err := os.MkdirTemp("", "network-constants-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	configDir := filepath.Join(tempDir, "configs")
	err = os.MkdirAll(configDir, 0755)
	require.NoError(t, err)
	
	// Create local.json config file
	localConfig := NetworkConfig{
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
	}
	configPath := filepath.Join(configDir, "local.json")
	data, err := json.MarshalIndent(localConfig, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)
	
	// Change to temp directory
	originalWd, _ := os.Getwd()
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	// Clean environment for isolated test
	originalEnv := os.Getenv("SHARDEUM_NETWORK")
	os.Unsetenv("SHARDEUM_NETWORK")
	defer func() {
		if originalEnv != "" {
			os.Setenv("SHARDEUM_NETWORK", originalEnv)
		}
	}()

	// Test default network config
	config := GetDefaultNetworkConfig()
	require.NotEmpty(t, config.ChainID)
	require.NotZero(t, config.EVMChainID)
	require.NotEmpty(t, config.BaseDenom)
	require.NotEmpty(t, config.DisplayDenom)

	// Test helper functions
	require.Equal(t, config.BaseDenom, ShardeumChainDenom())
	require.Equal(t, config.DisplayDenom, ShardeumDisplayDenom())
	require.Equal(t, config.EVMChainID, ShardeumChainID())
	require.Equal(t, config.ChainID, DefaultChainID())
}

func TestNetworkConstants_EnvironmentOverride(t *testing.T) {
	// Create temporary directory and config files for this test
	tempDir, err := os.MkdirTemp("", "network-override-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	configDir := filepath.Join(tempDir, "configs")
	err = os.MkdirAll(configDir, 0755)
	require.NoError(t, err)
	
	// Create testnet.json config file
	testnetConfig := NetworkConfig{
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
	}
	configPath := filepath.Join(configDir, "testnet.json")
	data, err := json.MarshalIndent(testnetConfig, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)
	
	// Change to temp directory
	originalWd, _ := os.Getwd()
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	// Set environment to testnet
	os.Setenv("SHARDEUM_NETWORK", "testnet")
	defer os.Unsetenv("SHARDEUM_NETWORK")

	// Test that constants use testnet config
	require.Equal(t, "shardeum-testnet", DefaultChainID())
	require.Equal(t, "ashm", ShardeumChainDenom())
	require.Equal(t, "shm", ShardeumDisplayDenom())
	require.Equal(t, uint64(8119), ShardeumChainID())
}

// Benchmark tests for performance
func BenchmarkGetNetworkConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GetNetworkConfig("testnet")
	}
}

func BenchmarkApplyEnvOverrides(b *testing.B) {
	// Create temp config for benchmark
	tempDir, _ := os.MkdirTemp("", "benchmark-test")
	defer os.RemoveAll(tempDir)
	configDir := filepath.Join(tempDir, "configs")
	os.MkdirAll(configDir, 0755)
	
	testConfig := NetworkConfig{
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
	}
	configPath := filepath.Join(configDir, "testnet.json")
	data, _ := json.MarshalIndent(testConfig, "", "  ")
	os.WriteFile(configPath, data, 0644)
	
	originalWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalWd)
	
	for i := 0; i < b.N; i++ {
		_ = applyEnvOverrides(testConfig)
	}
}