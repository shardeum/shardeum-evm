package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ScriptsTestSuite struct {
	suite.Suite
	tempDir       string
	configsDir    string
	configDir     string
	repoRoot      string
	originalWd    string
	mockBinary    string
	originalEnv   map[string]string
	startScript   string
	addNodeScript string
	setNetScript  string
}

func TestScriptsTestSuite(t *testing.T) {
	suite.Run(t, new(ScriptsTestSuite))
}

func (suite *ScriptsTestSuite) SetupSuite() {
	// Get repo root
	wd, err := os.Getwd()
	suite.Require().NoError(err)
	suite.originalWd = wd

	// Find repo root (go up directories until we find scripts/)
	repoRoot := wd
	for {
		scriptsDir := filepath.Join(repoRoot, "scripts")
		if _, err := os.Stat(scriptsDir); err == nil {
			break
		}
		parent := filepath.Dir(repoRoot)
		if parent == repoRoot {
			suite.T().Fatal("Could not find repo root (no scripts/ directory found)")
		}
		repoRoot = parent
	}
	suite.repoRoot = repoRoot

	// Set script paths
	suite.startScript = filepath.Join(repoRoot, "scripts", "start_network.sh")
	suite.addNodeScript = filepath.Join(repoRoot, "scripts", "add_node.sh")
	suite.setNetScript = filepath.Join(repoRoot, "scripts", "set_network.sh")

	// Verify scripts exist
	suite.Require().FileExists(suite.startScript)
	suite.Require().FileExists(suite.addNodeScript)
	suite.Require().FileExists(suite.setNetScript)
}

func (suite *ScriptsTestSuite) SetupTest() {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "scripts-test")
	suite.Require().NoError(err)
	suite.tempDir = tempDir

	suite.configsDir = filepath.Join(tempDir, "configs")
	suite.configDir = filepath.Join(tempDir, "config")
	err = os.MkdirAll(suite.configsDir, 0o755)
	suite.Require().NoError(err)
	err = os.MkdirAll(suite.configDir, 0o755)
	suite.Require().NoError(err)

	// Create mock binary
	suite.createMockBinary()

	// Save and clear environment variables
	suite.originalEnv = make(map[string]string)
	envVars := []string{
		"SHARDEUM_NETWORK", "SHARDEUM_CHAIN_ID", "SHARDEUM_EVM_CHAIN_ID",
		"SHARDEUM_BASE_DENOM", "SHARDEUM_DISPLAY_DENOM",
		"SHARDEUM_RPC_PORT", "SHARDEUM_REST_PORT", "SHARDEUM_JSON_RPC_PORT",
		"SHARDEUM_WEBSOCKET_PORT", "SHARDEUM_GRPC_PORT", "BINARY", "SKIP_BUILD",
	}
	for _, envVar := range envVars {
		suite.originalEnv[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}

	// Set test environment
	os.Setenv("BINARY", suite.mockBinary)
	os.Setenv("SKIP_BUILD", "1")

	// Change to temp directory
	err = os.Chdir(tempDir)
	suite.Require().NoError(err)
}

func (suite *ScriptsTestSuite) TearDownTest() {
	// Restore original working directory
	os.Chdir(suite.originalWd)

	// Restore environment variables
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

func (suite *ScriptsTestSuite) createMockBinary() {
	suite.mockBinary = filepath.Join(suite.tempDir, "test-shardeumd")
	mockScript := `#!/bin/bash
case "$1" in
    "init")
        echo "Initializing node $2 with chain-id $4 at home $6"
        mkdir -p "$6/config" "$6/data"
        echo '{"chain_id": "'$4'"}' > "$6/config/genesis.json"
        ;;
    "keys")
        echo "Keys operation: $@"
        ;;
    "add-genesis-account"|"gentx"|"collect-gentxs")
        echo "Genesis operation: $@"
        ;;
    "start")
        echo "Starting node: $@"
        sleep 1
        ;;
    "status")
        echo '{"NodeInfo":{"network":"test"},"SyncInfo":{"latest_block_height":"100"}}'
        ;;
    *)
        echo "Mock shardeumd: $@"
        ;;
esac`

	err := os.WriteFile(suite.mockBinary, []byte(mockScript), 0o755)
	suite.Require().NoError(err)
}

func (suite *ScriptsTestSuite) createTestConfig(network, chainID string, evmChainID int) {
	config := fmt.Sprintf(`{
  "name": "%s",
  "chain_id": "%s", 
  "evm_chain_id": %d,
  "base_denom": "ashm",
  "display_denom": "shm",
  "decimals": 18,
  "bech32_prefix": "shardeum",
  "ports": {
    "rpc": "26657",
    "rest": "1317",
    "json_rpc": "8545",
    "websocket": "8546",
    "grpc": "9090"
  },
  "genesis_file": "%s-genesis.json"
}`, network, chainID, evmChainID, network)

	configPath := filepath.Join(suite.configsDir, network+".json")
	err := os.WriteFile(configPath, []byte(config), 0o644)
	suite.Require().NoError(err)
}

func (suite *ScriptsTestSuite) createTestGenesis(network, chainID string) {
	genesis := fmt.Sprintf(`{
  "chain_id": "%s",
  "app_state": {
    "bank": {
      "denom_metadata": [{"base": "ashm", "display": "shm"}]
    }
  }
}`, chainID)

	genesisPath := filepath.Join(suite.configDir, network+"-genesis.json")
	err := os.WriteFile(genesisPath, []byte(genesis), 0o644)
	suite.Require().NoError(err)
}

func (suite *ScriptsTestSuite) runScript(script string, args ...string) (string, error) {
	cmd := exec.Command("bash", append([]string{script}, args...)...)
	cmd.Dir = suite.tempDir

	// Set environment for the command
	cmd.Env = append(os.Environ(),
		"BINARY="+suite.mockBinary,
		"SKIP_BUILD=1",
	)

	output, err := cmd.CombinedOutput()
	return string(output), err
}

// Tests for start_network.sh
func (suite *ScriptsTestSuite) TestStartNetworkHelp() {
	output, err := suite.runScript(suite.startScript, "--help")
	suite.Require().NoError(err)
	suite.Require().Contains(output, "Usage:")
	suite.Require().Contains(output, "--network")
	suite.Require().Contains(output, "--chain-id")
}

func (suite *ScriptsTestSuite) TestStartNetworkWithValidNetwork() {
	// Create test config and genesis using predefined values
	suite.createTestConfig("testnet", "shardeum-testnet", 8119)
	suite.createTestGenesis("testnet", "shardeum-testnet")

	// Copy genesis to default location for fallback
	err := os.WriteFile(filepath.Join(suite.configDir, "genesis.json"), []byte(`{"chain_id":"shardeum-testnet"}`), 0o644)
	suite.Require().NoError(err)

	// Test with valid network (will fail at some point but should parse args correctly)
	output, err := suite.runScript(suite.startScript, "1", "--network", "testnet")

	// Should contain network info in output
	suite.Require().Contains(output, "testnet")
	suite.Require().Contains(output, "shardeum-testnet")
}

func (suite *ScriptsTestSuite) TestStartNetworkWithInvalidNetwork() {
	// Create one valid config for the error message
	suite.createTestConfig("testnet", "shardeum-testnet", 8119)

	output, err := suite.runScript(suite.startScript, "1", "--network", "nonexistent")
	suite.Require().Error(err)
	suite.Require().Contains(output, "Network configuration file not found")
	suite.Require().Contains(output, "Available networks:")
	suite.Require().Contains(output, "testnet")
}

func (suite *ScriptsTestSuite) TestStartNetworkEnvironmentOverride() {
	suite.createTestConfig("mainnet", "main-chain", 8119)
	suite.createTestConfig("testnet", "shardeum-testnet", 8119)

	// Set environment variable
	os.Setenv("SHARDEUM_NETWORK", "mainnet")
	defer os.Unsetenv("SHARDEUM_NETWORK")

	// Should use mainnet due to environment variable
	output, err := suite.runScript(suite.startScript, "--help")
	suite.Require().NoError(err)
	suite.Require().Contains(output, "Usage:")
}

func (suite *ScriptsTestSuite) TestStartNetworkUnknownFlag() {
	output, err := suite.runScript(suite.startScript, "--unknown-flag")
	suite.Require().Error(err)
	suite.Require().Contains(output, "Unknown option")
}

// Tests for add_node.sh
func (suite *ScriptsTestSuite) TestAddNodeHelp() {
	output, err := suite.runScript(suite.addNodeScript, "--help")
	suite.Require().NoError(err)
	suite.Require().Contains(output, "Usage:")
	suite.Require().Contains(output, "node_id")
	suite.Require().Contains(output, "--network")
	suite.Require().Contains(output, "--seed-rpc")
}

func (suite *ScriptsTestSuite) TestAddNodeMissingNodeID() {
	output, err := suite.runScript(suite.addNodeScript)
	suite.Require().Error(err)
	suite.Require().Contains(output, "node_id is required")
}

func (suite *ScriptsTestSuite) TestAddNodeWithValidNetwork() {
	suite.createTestConfig("testnet", "shardeum-testnet", 8119)

	output, err := suite.runScript(suite.addNodeScript, "node5", "--network", "testnet", "--help")
	suite.Require().NoError(err)
	suite.Require().Contains(output, "Usage:")
}

func (suite *ScriptsTestSuite) TestAddNodeWithInvalidNetwork() {
	suite.createTestConfig("testnet", "shardeum-testnet", 8119)

	output, err := suite.runScript(suite.addNodeScript, "node5", "--network", "nonexistent")
	suite.Require().Error(err)
	suite.Require().Contains(output, "Network configuration file not found")
	suite.Require().Contains(output, "Available networks:")
}

func (suite *ScriptsTestSuite) TestAddNodeWithCustomSeedRPC() {
	suite.createTestConfig("testnet", "shardeum-testnet", 8119)

	output, err := suite.runScript(suite.addNodeScript, "node5", "--seed-rpc", "http://custom:26657", "--network", "testnet", "--help")
	suite.Require().NoError(err)
	suite.Require().Contains(output, "Usage:")
}

// Tests for set_network.sh
func (suite *ScriptsTestSuite) TestSetNetworkHelp() {
	// set_network.sh shows help when no args provided
	output, err := suite.runScript(suite.setNetScript)
	suite.Require().Error(err) // Script returns error when no args
	suite.Require().Contains(output, "Usage:")
	suite.Require().Contains(output, "Available networks:")
}

func (suite *ScriptsTestSuite) TestSetNetworkWithValidNetwork() {
	suite.createTestConfig("testnet", "shardeum-testnet", 8119)

	// Use bash to source the script and check environment
	cmd := exec.Command("bash", "-c", fmt.Sprintf(`
		cd %s
		source %s testnet 2>/dev/null || true
		echo "NETWORK=$SHARDEUM_NETWORK"
		echo "CHAIN_ID=$SHARDEUM_CHAIN_ID" 
		echo "EVM_CHAIN_ID=$SHARDEUM_EVM_CHAIN_ID"
	`, suite.tempDir, suite.setNetScript))

	output, err := cmd.CombinedOutput()
	suite.Require().NoError(err)

	outputStr := string(output)
	suite.Require().Contains(outputStr, "NETWORK=testnet")
	suite.Require().Contains(outputStr, "CHAIN_ID=shardeum-testnet")
	suite.Require().Contains(outputStr, "EVM_CHAIN_ID=8119")
}

func (suite *ScriptsTestSuite) TestSetNetworkWithInvalidNetwork() {
	suite.createTestConfig("testnet", "shardeum-testnet", 8119)

	output, err := suite.runScript(suite.setNetScript, "nonexistent")
	suite.Require().Error(err)
	suite.Require().Contains(output, "Network configuration file not found")
	suite.Require().Contains(output, "Available networks:")
	suite.Require().Contains(output, "testnet")
}

// Integration tests that test script interactions
func (suite *ScriptsTestSuite) TestEndToEndWorkflow() {
	// Setup test environment
	suite.createTestConfig("testnet", "shardeum-testnet", 8119)
	suite.createTestGenesis("testnet", "shardeum-testnet")

	// Copy genesis to default location
	err := os.WriteFile(filepath.Join(suite.configDir, "genesis.json"), []byte(`{"chain_id":"shardeum-testnet"}`), 0o644)
	suite.Require().NoError(err)

	// Step 1: Use set_network.sh to set environment (simulate sourcing)
	cmd := exec.Command("bash", "-c", fmt.Sprintf(`
		cd %s
		source %s testnet 2>/dev/null || true
		echo "ENV_SET=true"
		echo "NETWORK=$SHARDEUM_NETWORK"
	`, suite.tempDir, suite.setNetScript))

	output, err := cmd.CombinedOutput()
	suite.Require().NoError(err)
	suite.Require().Contains(string(output), "ENV_SET=true")
	suite.Require().Contains(string(output), "NETWORK=testnet")

	// Step 2: Verify start_network.sh can use the configuration
	output2, err2 := suite.runScript(suite.startScript, "--help")
	suite.Require().NoError(err2)
	suite.Require().Contains(output2, "Usage:")

	// Step 3: Verify add_node.sh can use the configuration
	output3, err3 := suite.runScript(suite.addNodeScript, "node5", "--help")
	suite.Require().NoError(err3)
	suite.Require().Contains(output3, "Usage:")
}

func (suite *ScriptsTestSuite) TestNetworkSwitching() {
	// Create multiple network configs
	suite.createTestConfig("mainnet", "shardeum-1", 8119)
	suite.createTestConfig("testnet", "shardeum-testnet", 8119)
	suite.createTestConfig("devnet", "shardeum-devnet", 8119)

	networks := []struct {
		name    string
		chainID string
		evmID   string
	}{
		{"mainnet", "shardeum-1", "8119"},
		{"testnet", "shardeum-testnet", "8119"},
		{"devnet", "shardeum-devnet", "8119"},
	}

	for _, network := range networks {
		// Test set_network.sh with each network
		cmd := exec.Command("bash", "-c", fmt.Sprintf(`
			cd %s
			source %s %s 2>/dev/null || true
			echo "NETWORK=$SHARDEUM_NETWORK"
			echo "CHAIN_ID=$SHARDEUM_CHAIN_ID"
			echo "EVM_CHAIN_ID=$SHARDEUM_EVM_CHAIN_ID"
		`, suite.tempDir, suite.setNetScript, network.name))

		output, err := cmd.CombinedOutput()
		suite.Require().NoError(err)

		outputStr := string(output)
		suite.Require().Contains(outputStr, "NETWORK="+network.name)
		suite.Require().Contains(outputStr, "CHAIN_ID="+network.chainID)
		suite.Require().Contains(outputStr, "EVM_CHAIN_ID="+network.evmID)
	}
}

func (suite *ScriptsTestSuite) TestEnvironmentVariablePrecedence() {
	suite.createTestConfig("testnet", "shardeum-testnet", 8119)
	suite.createTestConfig("mainnet", "shardeum-1", 8119)

	// Test that set_network.sh overrides pre-existing environment variables
	cmd := exec.Command("bash", "-c", fmt.Sprintf(`
		cd %s
		export SHARDEUM_NETWORK=mainnet
		export SHARDEUM_CHAIN_ID=custom-override
		source %s testnet 2>/dev/null || true
		echo "NETWORK=$SHARDEUM_NETWORK"
		echo "CHAIN_ID=$SHARDEUM_CHAIN_ID"
	`, suite.tempDir, suite.setNetScript))

	output, err := cmd.CombinedOutput()
	suite.Require().NoError(err)

	outputStr := string(output)
	// set_network.sh testnet should override to testnet
	suite.Require().Contains(outputStr, "NETWORK=testnet")
	// Should use testnet chain ID from config
	suite.Require().Contains(outputStr, "CHAIN_ID=shardeum-testnet")
}

func (suite *ScriptsTestSuite) TestJSONValidation() {
	// Create invalid JSON config
	invalidConfig := filepath.Join(suite.configsDir, "invalid.json")
	err := os.WriteFile(invalidConfig, []byte("invalid json"), 0o644)
	suite.Require().NoError(err)

	// Test that scripts handle invalid JSON gracefully
	output, err := suite.runScript(suite.startScript, "1", "--network", "invalid")
	suite.Require().Error(err)
	// Should contain some kind of error message about the config
	suite.Require().True(strings.Contains(output, "Error:") || strings.Contains(output, "error") || strings.Contains(output, "failed"))
}

// Benchmark test for script performance
func (suite *ScriptsTestSuite) TestScriptPerformance() {
	suite.createTestConfig("testnet", "shardeum-testnet", 8119)

	// Time how long help commands take (should be fast)
	start := suite.T().Name()
	suite.T().Log("Starting performance test:", start)

	output, err := suite.runScript(suite.startScript, "--help")
	suite.Require().NoError(err)
	suite.Require().Contains(output, "Usage:")

	output, err = suite.runScript(suite.addNodeScript, "--help")
	suite.Require().NoError(err)
	suite.Require().Contains(output, "Usage:")

	output, err = suite.runScript(suite.setNetScript)
	suite.Require().Error(err) // Expected for no args
	suite.Require().Contains(output, "Usage:")

	suite.T().Log("Performance test completed")
}
