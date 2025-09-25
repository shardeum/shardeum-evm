package main

import (
	"fmt"
	"os"

	"github.com/shardeum/shardeum-evm/shardeumd/cmd/shardeumd/cmd"
	shardeumdconfig "github.com/shardeum/shardeum-evm/shardeumd/cmd/shardeumd/config"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func main() {
	setupSDKConfig()

	rootCmd := cmd.NewRootCmd()
	if err := svrcmd.Execute(rootCmd, "shardeumd", shardeumdconfig.MustGetDefaultNodeHome()); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}

func setupSDKConfig() {
	// Initialize chains coin info with network configurations
	shardeumdconfig.InitializeChainsCoinInfo()

	config := sdk.GetConfig()
	shardeumdconfig.SetBech32Prefixes(config)
	shardeumdconfig.SetBip44CoinType(config)
	config.Seal()
}
