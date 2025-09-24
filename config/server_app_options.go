package config

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/holiman/uint256"
	"github.com/spf13/cast"

	srvflags "github.com/shardeum/shardeum-evm/server/flags"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/client/flags"
	sdkserver "github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
)

// NetworkConfig represents the minimal network configuration needed for genesis resolution
type NetworkConfig struct {
	GenesisFile string `json:"genesis_file,omitempty"`
}

// getNetworkGenesisPath resolves the genesis file path for a given network
func getNetworkGenesisPath(network string) (string, error) {
	// Try to load network config to get genesis file name
	var networkConfigPath string
	
	// 1) Check SHARDEUM_CONFIG_DIR first
	if configDir := os.Getenv("SHARDEUM_CONFIG_DIR"); configDir != "" {
		networkConfigPath = filepath.Join(configDir, "environments", fmt.Sprintf("%s.json", network))
	} else {
		// 2) Use default path
		networkConfigPath = filepath.Join("config", "environments", fmt.Sprintf("%s.json", network))
	}

	// Load network config to get genesis file name
	data, err := os.ReadFile(networkConfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to read network config: %w", err)
	}

	var config NetworkConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("failed to parse network config: %w", err)
	}

	if config.GenesisFile == "" {
		return "", fmt.Errorf("no genesis file specified in network config")
	}

	// Resolve genesis file path using same logic as network config
	var genesisPath string
	if configDir := os.Getenv("SHARDEUM_CONFIG_DIR"); configDir != "" {
		genesisPath = filepath.Join(configDir, "environments", config.GenesisFile)
	} else {
		genesisPath = filepath.Join("config", "environments", config.GenesisFile)
	}

	if _, err := os.Stat(genesisPath); err != nil {
		return "", fmt.Errorf("genesis file not found: %s", genesisPath)
	}

	return genesisPath, nil
}

// GetBlockGasLimit reads the genesis json file using AppGenesisFromFile
// to extract the consensus block gas limit before InitChain is called.
func GetBlockGasLimit(appOpts servertypes.AppOptions, logger log.Logger) uint64 {
	homeDir := cast.ToString(appOpts.Get(flags.FlagHome))
	if homeDir == "" {
		logger.Warn("home directory not found in app options, falling back to max uint64 block gas limit")
		return math.MaxUint64
	}

	// Try to get network-specific genesis path
	var genesisPath string
	if network := os.Getenv("SHARDEUM_NETWORK"); network != "" {
		if path, err := getNetworkGenesisPath(network); err == nil {
			genesisPath = path
			logger.Debug("using network-specific genesis file", "network", network, "path", genesisPath)
		} else {
			logger.Warn("failed to get network-specific genesis path, falling back to max uint64 block gas limit", "network", network, "error", err)
			return math.MaxUint64
		}
	} else {
		logger.Warn("SHARDEUM_NETWORK not set, falling back to max uint64 block gas limit")
		return math.MaxUint64
	}

	appGenesis, err := genutiltypes.AppGenesisFromFile(genesisPath)
	if err != nil {
		logger.Warn("failed to load genesis file, falling back to max uint64 block gas limit", "path", genesisPath, "error", err)
		return math.MaxUint64
	}
	genDoc, err := appGenesis.ToGenesisDoc()
	if err != nil {
		logger.Warn("failed to convert AppGenesis to GenesisDoc, falling back to max uint64 block gas limit", "path", genesisPath, "error", err)
		return math.MaxUint64
	}

	if genDoc.ConsensusParams == nil {
		logger.Warn("consensus parameters not found in genesis (nil), falling back to max uint64 block gas limit")
		return math.MaxUint64
	}

	maxGas := genDoc.ConsensusParams.Block.MaxGas
	if maxGas == -1 {
		logger.Warn("genesis max_gas is unlimited (-1), using max uint64")
		return math.MaxUint64
	}
	if maxGas < -1 {
		logger.Warn("invalid max_gas value in genesis, falling back to max uint64 block gas limit", "max_gas", maxGas)
		return math.MaxUint64
	}
	blockGasLimit := uint64(maxGas) // #nosec G115 -- maxGas >= 0 checked above

	logger.Debug(
		"extracted block gas limit from genesis using SDK AppGenesisFromFile",
		"genesis_path", genesisPath,
		"max_gas", maxGas,
		"block_gas_limit", blockGasLimit,
	)

	return blockGasLimit
}

// GetMinGasPrices reads the min gas prices from the app options, set from app.toml
// This is currently not used, but is kept in case this is useful for the mempool,
// in addition to the min tip flag
func GetMinGasPrices(appOpts servertypes.AppOptions, logger log.Logger) sdk.DecCoins {
	minGasPricesStr := cast.ToString(appOpts.Get(sdkserver.FlagMinGasPrices))
	minGasPrices, err := sdk.ParseDecCoins(minGasPricesStr)
	if err != nil {
		logger.With("error", err).Info("failed to parse min gas prices, using empty DecCoins")
		minGasPrices = sdk.DecCoins{}
	}

	return minGasPrices
}

// GetMinTip reads the min tip from the app options, set from app.toml
// This field is also known as the minimum priority fee
func GetMinTip(appOpts servertypes.AppOptions, logger log.Logger) *uint256.Int {
	minTipUint64 := cast.ToUint64(appOpts.Get(srvflags.EVMMinTip))
	minTip := uint256.NewInt(minTipUint64)

	if minTip.Cmp(uint256.NewInt(0)) >= 0 { // zero or positive
		return minTip
	}

	logger.Error("invalid min tip value in app.toml or flag, falling back to nil", "min_tip", minTipUint64)
	return nil
}
