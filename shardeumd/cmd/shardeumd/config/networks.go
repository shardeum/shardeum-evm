package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	evmtypes "github.com/shardeum/shardeum-evm/x/vm/types"
)

// NetworkConfig defines the configuration for a specific network
type NetworkConfig struct {
	Name         string           `json:"name"`
	ChainID      string           `json:"chain_id"`
	EVMChainID   uint64           `json:"evm_chain_id"`
	BaseDenom    string           `json:"base_denom"`
	DisplayDenom string           `json:"display_denom"`
	Decimals     evmtypes.Decimals `json:"decimals"`
	Bech32Prefix string           `json:"bech32_prefix"`
	Ports        NetworkPorts     `json:"ports"`
}

// NetworkPorts defines the port configuration for a network
type NetworkPorts struct {
	RPC       string `json:"rpc"`
	REST      string `json:"rest"`
	JSONRPC   string `json:"json_rpc"`
	WebSocket string `json:"websocket"`
	GRPC      string `json:"grpc"`
}

// getBuiltinNetworks returns the list of built-in network names that have JSON config files
func getBuiltinNetworks() []string {
	return []string{"mainnet", "testnet", "devnet", "local"}
}

// GetNetworkConfig returns the network configuration for the given network name
func GetNetworkConfig(network string) (NetworkConfig, error) {
	// Check environment variable override
	if envNetwork := os.Getenv("SHARDEUM_NETWORK"); envNetwork != "" {
		network = envNetwork
	}

	// Try loading from config file
	configPath := getNetworkConfigPath(network)
	if _, err := os.Stat(configPath); err == nil {
		return loadNetworkConfigFromFile(configPath)
	}

	return NetworkConfig{}, fmt.Errorf("network '%s' not found in config files", network)
}

// applyEnvOverrides applies environment variable overrides to the network config
func applyEnvOverrides(config NetworkConfig) NetworkConfig {
	if chainID := os.Getenv("SHARDEUM_CHAIN_ID"); chainID != "" {
		config.ChainID = chainID
	}
	if evmChainIDStr := os.Getenv("SHARDEUM_EVM_CHAIN_ID"); evmChainIDStr != "" {
		if evmChainID, err := parseUint64(evmChainIDStr); err == nil {
			config.EVMChainID = evmChainID
		}
	}
	if baseDenom := os.Getenv("SHARDEUM_BASE_DENOM"); baseDenom != "" {
		config.BaseDenom = baseDenom
	}
	if displayDenom := os.Getenv("SHARDEUM_DISPLAY_DENOM"); displayDenom != "" {
		config.DisplayDenom = displayDenom
	}
	if rpcPort := os.Getenv("SHARDEUM_RPC_PORT"); rpcPort != "" {
		config.Ports.RPC = rpcPort
	}
	if restPort := os.Getenv("SHARDEUM_REST_PORT"); restPort != "" {
		config.Ports.REST = restPort
	}
	if jsonRPCPort := os.Getenv("SHARDEUM_JSON_RPC_PORT"); jsonRPCPort != "" {
		config.Ports.JSONRPC = jsonRPCPort
	}
	if wsPort := os.Getenv("SHARDEUM_WEBSOCKET_PORT"); wsPort != "" {
		config.Ports.WebSocket = wsPort
	}
	if grpcPort := os.Getenv("SHARDEUM_GRPC_PORT"); grpcPort != "" {
		config.Ports.GRPC = grpcPort
	}

	return config
}

// getNetworkConfigPath returns the path to the network configuration file
func getNetworkConfigPath(network string) string {
    	// 1) Explicit env dir
	if configDir := os.Getenv("SHARDEUM_CONFIG_DIR"); configDir != "" {
		p := filepath.Join(configDir, "environments", fmt.Sprintf("%s.json", network))
		if _, err := os.Stat(p); err == nil {
			return p
		}
		// Env set but file missing: fail with guidance
		panic(fmt.Sprintf("network config '%s.json' not found in SHARDEUM_CONFIG_DIR='%s'", network, configDir))
	}

	// 2) Next to the executable
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		p := filepath.Join(execDir, "environments", fmt.Sprintf("%s.json", network))
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// 3) Current working directory
	if cwd, err := os.Getwd(); err == nil {
		p := filepath.Join(cwd, "environments", fmt.Sprintf("%s.json", network))
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

    return filepath.Join("config/environments", fmt.Sprintf("%s.json", network))
}

// loadNetworkConfigFromFile loads network configuration from a JSON file
func loadNetworkConfigFromFile(filePath string) (NetworkConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return NetworkConfig{}, fmt.Errorf("failed to read network config file '%s': %w", filePath, err)
	}

	var config NetworkConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return NetworkConfig{}, fmt.Errorf("failed to parse network config file '%s': %w", filePath, err)
	}

	return applyEnvOverrides(config), nil
}

// GetAvailableNetworks returns a list of available network names
func GetAvailableNetworks() []string {
	networks := []string{}

	// Check for config files
	configsDir := "configs"
	if entries, err := os.ReadDir(configsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
				name := entry.Name()[:len(entry.Name())-5] // Remove .json extension
				networks = append(networks, name)
			}
		}
	}

	return networks
}

// parseUint64 parses a string to uint64
func parseUint64(s string) (uint64, error) {
	var result uint64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid uint64: %s", s)
		}
		result = result*10 + uint64(r-'0')
	}
	return result, nil
}

// UpdateChainsCoinInfo updates the global ChainsCoinInfo map with network configurations
func UpdateChainsCoinInfo() {
	builtinNetworks := getBuiltinNetworks()
	for _, networkName := range builtinNetworks {
		if config, err := GetNetworkConfig(networkName); err == nil {
			ChainsCoinInfo[config.EVMChainID] = evmtypes.EvmCoinInfo{
				Denom:         config.BaseDenom,
				ExtendedDenom: config.BaseDenom,
				DisplayDenom:  config.DisplayDenom,
				Decimals:      config.Decimals,
			}
		}
	}
}