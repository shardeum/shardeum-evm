package config

import (
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
	// Empty for now, can be expanded with other fields as needed
}

// getNetworkGenesisPath resolves the genesis file path for a given network
func getNetworkGenesisPath(network string) (string, error) {
	var configDir string

	// 1) Check SHARDEUM_CONFIG_DIR first
	if envDir := os.Getenv("SHARDEUM_CONFIG_DIR"); envDir != "" {
		configDir = envDir
	} else {
		// 2) Use default path
		configDir = "config"
	}

	// Check for split genesis files first (pattern: {network}-genesis.genesis.json)
	splitGenesisPath := filepath.Join(configDir, "environments", fmt.Sprintf("%s-genesis.genesis.json", network))
	if _, err := os.Stat(splitGenesisPath); err == nil {
		fmt.Printf("DEBUG: Found split genesis file: %s\n", splitGenesisPath)
		return splitGenesisPath, nil
	}

	// Then check for monolithic genesis file (pattern: {network}-genesis.json)
	monolithicGenesisPath := filepath.Join(configDir, "environments", fmt.Sprintf("%s-genesis.json", network))
	if _, err := os.Stat(monolithicGenesisPath); err == nil {
		return monolithicGenesisPath, nil
	}

	return "", fmt.Errorf("genesis file not found: checked %s and %s", splitGenesisPath, monolithicGenesisPath)
}

// GetBlockGasLimit reads the genesis json file using AppGenesisFromFile
// to extract the consensus block gas limit before InitChain is called.
func GetBlockGasLimit(appOpts servertypes.AppOptions, logger log.Logger) uint64 {
	homeDir := cast.ToString(appOpts.Get(flags.FlagHome))
	if homeDir == "" {
		logger.Error("home directory not found in app options - this is required for proper configuration")
		panic("GetBlockGasLimit: home directory not found in app options")
	}

	// Try to get network-specific genesis path
	var genesisPath string
	if network := os.Getenv("SHARDEUM_NETWORK"); network != "" {
		if path, err := getNetworkGenesisPath(network); err == nil {
			genesisPath = path
			logger.Debug("using network-specific genesis file", "network", network, "path", genesisPath)
		} else {
			logger.Error("failed to get network-specific genesis path", "network", network, "error", err)
			panic(fmt.Sprintf("GetBlockGasLimit: failed to resolve genesis path for network '%s': %v", network, err))
		}
	} else {
		logger.Error("SHARDEUM_NETWORK environment variable not set - this is required for network-specific configuration")
		panic("GetBlockGasLimit: SHARDEUM_NETWORK environment variable not set")
	}

	appGenesis, err := genutiltypes.AppGenesisFromFile(genesisPath)
	if err != nil {
		logger.Error("failed to load genesis file", "path", genesisPath, "error", err)
		panic(fmt.Sprintf("GetBlockGasLimit: failed to load genesis file '%s': %v", genesisPath, err))
	}
	genDoc, err := appGenesis.ToGenesisDoc()
	if err != nil {
		logger.Error("failed to convert AppGenesis to GenesisDoc", "path", genesisPath, "error", err)
		panic(fmt.Sprintf("GetBlockGasLimit: failed to parse genesis file '%s': %v", genesisPath, err))
	}

	if genDoc.ConsensusParams == nil {
		logger.Error("consensus parameters not found in genesis", "path", genesisPath)
		panic(fmt.Sprintf("GetBlockGasLimit: consensus parameters missing in genesis file '%s'", genesisPath))
	}

	maxGas := genDoc.ConsensusParams.Block.MaxGas
	if maxGas == -1 {
		logger.Warn("genesis max_gas is unlimited (-1), using max uint64")
		return math.MaxUint64
	}
	if maxGas < -1 {
		logger.Error("invalid max_gas value in genesis", "max_gas", maxGas, "path", genesisPath)
		panic(fmt.Sprintf("GetBlockGasLimit: invalid max_gas value %d in genesis file '%s'", maxGas, genesisPath))
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
