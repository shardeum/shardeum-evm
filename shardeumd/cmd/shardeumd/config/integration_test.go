package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
	suite.Suite
	tempDir      string
	configsDir   string
	configDir    string
	repoRoot     string
	originalWd   string
	mockBinary   string
	originalEnv  map[string]string
	startScript  string
	addNodeScript string
	setNetScript string
}

func TestIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(IntegrationTestSuite))
}

func (suite *IntegrationTestSuite) SetupSuite() {
	// Get repo root
	wd, err := os.Getwd()
	suite.Require().NoError(err)
	suite.originalWd = wd

	// Find repo root
	repoRoot := wd
	for {
		scriptsDir := filepath.Join(repoRoot, "scripts")
		if _, err := os.Stat(scriptsDir); err == nil {
			break
		}
		parent := filepath.Dir(repoRoot)
		if parent == repoRoot {
			suite.T().Fatal("Could not find repo root")
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

func (suite *IntegrationTestSuite) SetupTest() {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "integration-test")
	suite.Require().NoError(err)
	suite.tempDir = tempDir

	suite.configsDir = filepath.Join(tempDir, "configs")
	suite.configDir = filepath.Join(tempDir, "config")
	err = os.MkdirAll(suite.configsDir, 0755)
	suite.Require().NoError(err)
	err = os.MkdirAll(suite.configDir, 0755)
	suite.Require().NoError(err)

	// Create mock binary
	suite.createMockBinary()

	// Save environment
	suite.originalEnv = make(map[string]string)
	envVars := []string{
		"SHARDEUM_NETWORK", "SHARDEUM_CHAIN_ID", "SHARDEUM_EVM_CHAIN_ID",
		"SHARDEUM_BASE_DENOM", "SHARDEUM_DISPLAY_DENOM",
		"BINARY", "SKIP_BUILD",
	}
	for _, envVar := range envVars {
		suite.originalEnv[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}

	// Change to temp directory
	err = os.Chdir(tempDir)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TearDownTest() {
	// Restore working directory
	os.Chdir(suite.originalWd)

	// Restore environment
	for envVar, value := range suite.originalEnv {
		if value == "" {
			os.Unsetenv(envVar)
		} else {
			os.Setenv(envVar, value)
		}
	}

	// Clean up
	os.RemoveAll(suite.tempDir)
}

func (suite *IntegrationTestSuite) createMockBinary() {
	suite.mockBinary = filepath.Join(suite.tempDir, "test-shardeumd")
	mockScript := `#!/bin/bash
case "$1" in
    "init")
        echo "Initializing node $2 with chain-id $4 at home $6"
        mkdir -p "$6/config" "$6/data"
        cat > "$6/config/genesis.json" << EOF
{
  "chain_id": "$4",
  "app_state": {
    "bank": {
      "denom_metadata": [{"base": "ashm", "display": "shm"}]
    }
  }
}
EOF
        cat > "$6/config/config.toml" << EOF
[rpc]
laddr = "tcp://0.0.0.0:26657"
[api]  
address = "tcp://0.0.0.0:1317"
EOF
        ;;
    "keys")
        case "$2" in
            "add")
                echo '{"name":"'$3'","address":"cosmos1test","mnemonic":"test mnemonic"}'
                ;;
            "show")
                echo '{"name":"'$3'","address":"cosmos1test"}'
                ;;
        esac
        ;;
    "add-genesis-account")
        echo "Adding genesis account: $2 $3"
        ;;
    "gentx")
        echo "Generating genesis transaction"
        mkdir -p "$(dirname "$5")"
        echo '{"body":{"messages":[]}}' > "$5"
        ;;
    "collect-gentxs")
        echo "Collecting genesis transactions"
        ;;
    "start")
        echo "Starting node with chain-id from config..."
        sleep 2
        echo "Node started successfully"
        ;;
    "status")
        echo '{"NodeInfo":{"network":"test-chain"},"SyncInfo":{"latest_block_height":"100"}}'
        ;;
    *)
        echo "Mock shardeumd: $@"
        ;;
esac`

	err := os.WriteFile(suite.mockBinary, []byte(mockScript), 0755)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) createNetworkConfigs() {
	networks := map[string]struct {
		chainID    string
		evmChainID int
	}{
		"mainnet": {"shardeum-1", 8119},
		"testnet": {"shardeum-testnet", 8119},
		"devnet":  {"shardeum-devnet", 8119},
		"local":   {"shardeum-local", 8119},
		"custom":  {"custom-chain", 9999},
	}

	for network, config := range networks {
		suite.createTestConfig(network, config.chainID, config.evmChainID)
		suite.createTestGenesis(network, config.chainID)
	}

	// Create default fallback genesis
	err := os.WriteFile(filepath.Join(suite.configDir, "genesis.json"), 
		[]byte(`{"chain_id":"shardeum-testnet"}`), 0644)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) createTestConfig(network, chainID string, evmChainID int) {
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
	err := os.WriteFile(configPath, []byte(config), 0644)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) createTestGenesis(network, chainID string) {
	genesis := fmt.Sprintf(`{
  "chain_id": "%s",
  "app_state": {
    "bank": {
      "denom_metadata": [{"base": "ashm", "display": "shm"}]
    },
    "evm": {
      "params": {"evm_denom": "ashm"}
    }
  }
}`, chainID)

	genesisPath := filepath.Join(suite.configDir, network+"-genesis.json")
	err := os.WriteFile(genesisPath, []byte(genesis), 0644)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = suite.tempDir
	cmd.Env = append(os.Environ(),
		"BINARY="+suite.mockBinary,
		"SKIP_BUILD=1",
	)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// Integration Tests

func (suite *IntegrationTestSuite) TestCompleteNetworkDeploymentWorkflow() {
	suite.T().Log("Testing complete network deployment workflow")
	
	// Setup
	suite.createNetworkConfigs()

	// Step 1: Set network environment using set_network.sh
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

	// Step 2: Simulate node initialization (what start_network.sh does)
	nodeDir := filepath.Join(suite.tempDir, ".testnet", "node0")
	err = os.MkdirAll(nodeDir, 0755)
	suite.Require().NoError(err)

	// Initialize node
	initOutput, err := suite.runCommand(suite.mockBinary, "init", "node0", 
		"--chain-id", "shardeum-testnet", "--home", nodeDir)
	suite.Require().NoError(err)
	suite.Require().Contains(initOutput, "Initializing node")

	// Step 3: Verify genesis file was created correctly
	genesisPath := filepath.Join(nodeDir, "config", "genesis.json")
	suite.Require().FileExists(genesisPath)

	// Step 4: Copy network-specific genesis
	networkGenesis := filepath.Join(suite.configDir, "testnet-genesis.json")
	if _, err := os.Stat(networkGenesis); err == nil {
		genesisContent, err := os.ReadFile(networkGenesis)
		suite.Require().NoError(err)
		err = os.WriteFile(genesisPath, genesisContent, 0644)
		suite.Require().NoError(err)
	}

	// Step 5: Verify final setup
	suite.Require().DirExists(filepath.Join(nodeDir, "config"))
	suite.Require().DirExists(filepath.Join(nodeDir, "data"))
	suite.Require().FileExists(genesisPath)

	suite.T().Log("Complete workflow test passed")
}

func (suite *IntegrationTestSuite) TestNetworkSwitchingIntegration() {
	suite.T().Log("Testing network switching integration")
	
	suite.createNetworkConfigs()

	networks := []struct {
		name     string
		chainID  string
		evmChainID string
	}{
		{"mainnet", "shardeum-1", "8119"},
		{"testnet", "shardeum-testnet", "8119"},
		{"devnet", "shardeum-devnet", "8119"},
		{"local", "shardeum-local", "8119"},
	}

	for _, network := range networks {
		suite.T().Logf("Testing network: %s", network.name)
		
		// Set network environment
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
		suite.Require().Contains(outputStr, "EVM_CHAIN_ID="+network.evmChainID)

		// Verify start_network.sh can read the configuration
		output2, err2 := suite.runCommand("bash", suite.startScript, "--help")
		suite.Require().NoError(err2)
		suite.Require().Contains(output2, "Usage:")

		// Verify add_node.sh can read the configuration
		output3, err3 := suite.runCommand("bash", suite.addNodeScript, "node1", "--help")
		suite.Require().NoError(err3)
		suite.Require().Contains(output3, "Usage:")
	}

	suite.T().Log("Network switching integration test passed")
}

func (suite *IntegrationTestSuite) TestEnvironmentVariablePrecedenceIntegration() {
	suite.T().Log("Testing environment variable precedence integration")
	
	suite.createNetworkConfigs()

	// Test that set_network.sh overrides pre-existing environment variables
	cmd := exec.Command("bash", "-c", fmt.Sprintf(`
		cd %s
		export SHARDEUM_NETWORK=mainnet
		export SHARDEUM_CHAIN_ID=custom-override-chain
		export SHARDEUM_EVM_CHAIN_ID=7777
		source %s testnet 2>/dev/null || true
		echo "NETWORK=$SHARDEUM_NETWORK"
		echo "CHAIN_ID=$SHARDEUM_CHAIN_ID"
		echo "EVM_CHAIN_ID=$SHARDEUM_EVM_CHAIN_ID"
	`, suite.tempDir, suite.setNetScript))
	
	output, err := cmd.CombinedOutput()
	suite.Require().NoError(err)
	
	outputStr := string(output)
	// set_network.sh testnet should override to testnet
	suite.Require().Contains(outputStr, "NETWORK=testnet")
	// Should use testnet chain ID from config
	suite.Require().Contains(outputStr, "CHAIN_ID=shardeum-testnet")
	suite.Require().Contains(outputStr, "EVM_CHAIN_ID=8119")

	suite.T().Log("Environment precedence integration test passed")
}

func (suite *IntegrationTestSuite) TestErrorHandlingIntegration() {
	suite.T().Log("Testing error handling integration")
	
	// Test 1: Invalid network configuration
	suite.createTestConfig("testnet", "test-chain", 8119)
	
	// Create invalid JSON
	invalidPath := filepath.Join(suite.configsDir, "invalid.json")
	err := os.WriteFile(invalidPath, []byte("invalid json"), 0644)
	suite.Require().NoError(err)

	output, err := suite.runCommand("bash", suite.startScript, "1", "--network", "invalid")
	suite.Require().Error(err)
	suite.Require().Contains(output, "Error")

	// Test 2: Missing network configuration
	output, err = suite.runCommand("bash", suite.startScript, "1", "--network", "nonexistent")
	suite.Require().Error(err)
	suite.Require().Contains(output, "Network configuration file not found")
	suite.Require().Contains(output, "Available networks:")
	suite.Require().Contains(output, "testnet")

	// Test 3: Missing node ID for add_node.sh
	output, err = suite.runCommand("bash", suite.addNodeScript)
	suite.Require().Error(err)
	suite.Require().Contains(output, "node_id is required")

	suite.T().Log("Error handling integration test passed")
}

func (suite *IntegrationTestSuite) TestGenesisFileSelectionIntegration() {
	suite.T().Log("Testing genesis file selection integration")
	
	suite.createNetworkConfigs()

	// Test that network-specific genesis files are used
	expectedChainIDs := map[string]string{
		"mainnet": "shardeum-1",
		"testnet": "shardeum-testnet", 
		"devnet":  "shardeum-devnet",
	}
	
	for network, expectedChainID := range expectedChainIDs {
		// Verify genesis file exists
		genesisFile := fmt.Sprintf("%s-genesis.json", network)
		genesisPath := filepath.Join(suite.configDir, genesisFile)
		suite.Require().FileExists(genesisPath)

		// Verify genesis has correct chain ID
		genesisContent, err := os.ReadFile(genesisPath)
		suite.Require().NoError(err)
		suite.Require().Contains(string(genesisContent), expectedChainID)
	}

	suite.T().Log("Genesis file selection integration test passed")
}

func (suite *IntegrationTestSuite) TestPortConfigurationIntegration() {
	suite.T().Log("Testing port configuration integration")
	
	suite.createNetworkConfigs()

	// Test that set_network.sh sets port environment variables using predefined testnet
	cmd := exec.Command("bash", "-c", fmt.Sprintf(`
		cd %s
		source %s testnet 2>/dev/null || true
		echo "RPC_PORT=$SHARDEUM_RPC_PORT"
		echo "REST_PORT=$SHARDEUM_REST_PORT"
		echo "JSON_RPC_PORT=$SHARDEUM_JSON_RPC_PORT"
		echo "WEBSOCKET_PORT=$SHARDEUM_WEBSOCKET_PORT"
		echo "GRPC_PORT=$SHARDEUM_GRPC_PORT"
	`, suite.tempDir, suite.setNetScript))
	
	output, err := cmd.CombinedOutput()
	suite.Require().NoError(err)
	
	outputStr := string(output)
	suite.Require().Contains(outputStr, "RPC_PORT=26657")
	suite.Require().Contains(outputStr, "REST_PORT=1317")
	suite.Require().Contains(outputStr, "JSON_RPC_PORT=8545")
	suite.Require().Contains(outputStr, "WEBSOCKET_PORT=8546")
	suite.Require().Contains(outputStr, "GRPC_PORT=9090")

	suite.T().Log("Port configuration integration test passed")
}

func (suite *IntegrationTestSuite) TestMultiNodeSetupIntegration() {
	suite.T().Log("Testing multi-node setup integration")
	
	suite.createNetworkConfigs()

	// Simulate setting up multiple nodes
	baseDir := filepath.Join(suite.tempDir, ".testnet")
	err := os.MkdirAll(baseDir, 0755)
	suite.Require().NoError(err)

	// Initialize 3 nodes
	for i := 0; i < 3; i++ {
		nodeID := fmt.Sprintf("node%d", i)
		nodeDir := filepath.Join(baseDir, nodeID)
		
		// Initialize node
		output, err := suite.runCommand(suite.mockBinary, "init", nodeID,
			"--chain-id", "shardeum-testnet", "--home", nodeDir)
		suite.Require().NoError(err)
		suite.Require().Contains(output, "Initializing node")

		// Verify node directory structure
		suite.Require().DirExists(filepath.Join(nodeDir, "config"))
		suite.Require().DirExists(filepath.Join(nodeDir, "data"))
		suite.Require().FileExists(filepath.Join(nodeDir, "config", "genesis.json"))
		suite.Require().FileExists(filepath.Join(nodeDir, "config", "config.toml"))
	}

	suite.T().Log("Multi-node setup integration test passed")
}

func (suite *IntegrationTestSuite) TestPerformanceIntegration() {
	suite.T().Log("Testing performance integration")
	
	suite.createNetworkConfigs()

	start := time.Now()
	
	// Test that operations complete in reasonable time
	operations := []func(){
		func() {
			suite.runCommand("bash", suite.startScript, "--help")
		},
		func() {
			suite.runCommand("bash", suite.addNodeScript, "node1", "--help")
		},
		func() {
			cmd := exec.Command("bash", "-c", fmt.Sprintf(`
				cd %s
				source %s testnet 2>/dev/null || true
			`, suite.tempDir, suite.setNetScript))
			cmd.CombinedOutput()
		},
	}

	for i, op := range operations {
		opStart := time.Now()
		op()
		opDuration := time.Since(opStart)
		suite.Require().Less(opDuration, 5*time.Second, "Operation %d took too long: %v", i, opDuration)
	}

	totalDuration := time.Since(start)
	suite.Require().Less(totalDuration, 10*time.Second, "Total performance test took too long: %v", totalDuration)

	suite.T().Logf("Performance integration test passed in %v", totalDuration)
}