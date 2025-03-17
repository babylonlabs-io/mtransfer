package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	appparams "github.com/babylonlabs-io/babylon/app/params"
	bbncfg "github.com/babylonlabs-io/babylon/client/config"

	"github.com/babylonlabs-io/mtransfer/cmd/mtransferd/daemon"
	cfg "github.com/babylonlabs-io/mtransfer/config"
)

// NewRootCmd creates a new root command for mtransferd. It is called once in the main function.
func NewRootCmd() *cobra.Command {
	defaultDir := btcutil.AppDataDir("mtransferd", false)
	rootCmd := &cobra.Command{
		Use:               "mtransferd",
		Short:             "mtransferd - Multi Transfer Daemon (mtransferd).",
		Long:              `mtransferd is the daemon to send many bank.MsgMultiSend txs.`,
		SilenceErrors:     false,
		PersistentPreRunE: persistClientCtx(client.Context{}),
	}
	rootCmd.PersistentFlags().String(flags.FlagHome, defaultDir, "The application home directory")

	return rootCmd
}

func main() {
	cmd := NewRootCmd()
	cmd.AddCommand(
		daemon.CommandInit(),
		daemon.CommandStart(),
		daemon.CommandKeys(),
		version.NewVersionCommand(),
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := cmd.ExecuteContext(ctx); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Whoops. There was an error while executing your mtransferd CLI '%s'", err)
		os.Exit(1) //nolint:gocritic
	}
}

// persistClientCtx persist some vars from the cmd or config to the client context.
// It gives preferences to flags over the values in the config. If the flag is not set
// and exists a value in the config that could be used, it will be set in the ctx.
func persistClientCtx(ctx client.Context) func(cmd *cobra.Command, _ []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		encCfg := appparams.DefaultEncodingConfig()
		std.RegisterInterfaces(encCfg.InterfaceRegistry)

		ctx = ctx.
			WithCodec(encCfg.Codec).
			WithInterfaceRegistry(encCfg.InterfaceRegistry).
			WithTxConfig(encCfg.TxConfig).
			WithLegacyAmino(encCfg.Amino).
			WithInput(os.Stdin).
			WithAccountRetriever(types.AccountRetriever{})

		// set the default command outputs
		cmd.SetOut(cmd.OutOrStdout())
		cmd.SetErr(cmd.ErrOrStderr())

		if err := client.SetCmdClientContextHandler(ctx, cmd); err != nil {
			return err
		}

		ctx = client.GetClientContextFromCmd(cmd)
		// get the default config
		cfg := cfg.DefaultConfigWithHome(ctx.HomeDir)

		// load the defaults if not set by flag
		// flags have preference over config.
		ctx, err := fillContextFromBabylonConfig(ctx, cmd.Flags(), cfg)
		if err != nil {
			return err
		}

		// updates the ctx in the cmd in case something was modified bt the config
		return client.SetCmdClientContext(cmd, ctx)
	}
}

// fillContextFromBabylonConfig loads the bbn config to the context if values were not set by flag.
// Preference is FlagSet values over the config.
func fillContextFromBabylonConfig(ctx client.Context, flagSet *pflag.FlagSet, bbnConf bbncfg.BabylonConfig) (client.Context, error) {
	if !flagSet.Changed(flags.FlagFrom) {
		ctx = ctx.WithFrom(bbnConf.Key)
	}
	if !flagSet.Changed(flags.FlagChainID) {
		ctx = ctx.WithChainID(bbnConf.ChainID)
	}
	if !flagSet.Changed(flags.FlagKeyringBackend) {
		kr, err := client.NewKeyringFromBackend(ctx, bbnConf.KeyringBackend)
		if err != nil {
			return ctx, err
		}
		ctx = ctx.WithKeyring(kr)
	}
	if !flagSet.Changed(flags.FlagKeyringDir) {
		ctx = ctx.WithKeyringDir(bbnConf.KeyDirectory)
	}
	if !flagSet.Changed(flags.FlagOutput) {
		ctx = ctx.WithOutputFormat(bbnConf.OutputFormat)
	}
	if !flagSet.Changed(flags.FlagSignMode) {
		ctx = ctx.WithSignModeStr(bbnConf.SignModeStr)
	}

	return ctx, nil
}
