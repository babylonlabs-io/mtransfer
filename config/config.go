package config

import (
	bbncfg "github.com/babylonlabs-io/babylon/client/config"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/pflag"
)

const defaultAppKeyName = "mtransfer"

func DefaultConfigWithHome(homePath string) bbncfg.BabylonConfig {
	bbnCfg := bbncfg.DefaultBabylonConfig()
	bbnCfg.Key = defaultAppKeyName
	bbnCfg.KeyDirectory = homePath
	bbnCfg.GasAdjustment = 1.1
	if err := bbnCfg.Validate(); err != nil {
		panic(err)
	}

	return bbnCfg
}

// ConfigWithFlags updates the default config values with the provided flags.
func ConfigWithFlags(ctx client.Context, homePath string, flagSet *pflag.FlagSet) (bbncfg.BabylonConfig, error) {
	cfg := DefaultConfigWithHome(homePath)
	if flagSet.Changed(flags.FlagFrom) {
		key, _ := flagSet.GetString(flags.FlagFrom)
		cfg.Key = key
	}
	if flagSet.Changed(flags.FlagChainID) {
		chainID, _ := flagSet.GetString(flags.FlagChainID)
		cfg.ChainID = chainID
	}
	if flagSet.Changed(flags.FlagNode) {
		nodeAddr, _ := flagSet.GetString(flags.FlagNode)
		cfg.RPCAddr = nodeAddr
	}
	if flagSet.Changed(flags.FlagKeyringBackend) {
		kb, _ := flagSet.GetString(flags.FlagKeyringBackend)
		_, err := client.NewKeyringFromBackend(ctx, kb)
		if err != nil {
			return cfg, err
		}
		cfg.KeyringBackend = kb
	}
	if flagSet.Changed(flags.FlagKeyringDir) {
		kd, _ := flagSet.GetString(flags.FlagKeyringDir)
		cfg.KeyDirectory = kd
	}
	if flagSet.Changed(flags.FlagOutput) {
		outFormat, _ := flagSet.GetString(flags.FlagOutput)
		cfg.OutputFormat = outFormat
	}
	if flagSet.Changed(flags.FlagSignMode) {
		signMode, _ := flagSet.GetString(flags.FlagSignMode)
		cfg.SignModeStr = signMode
	}

	if flagSet.Changed(flags.FlagGasAdjustment) {
		gasAdj, _ := flagSet.GetFloat64(flags.FlagGasAdjustment)
		cfg.GasAdjustment = gasAdj
	}

	return cfg, nil
}
