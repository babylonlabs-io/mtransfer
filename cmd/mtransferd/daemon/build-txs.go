package daemon

import (
	"fmt"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/babylonlabs-io/mtransfer/app"
	"github.com/babylonlabs-io/mtransfer/config"
	"github.com/babylonlabs-io/mtransfer/types"
)

// CommandBuildTxs returns the build-txs command of mtransferd daemon.
func CommandBuildTxs() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "build-txs",
		Short: "Build unsigned transactions with bank.MsgMultiSend.",
		Long: `Build unsigned transactions with bank.MsgMultiSend with the provided file, sender key or address, and batch sizes.
You can also run the transfer file validation to get 
the total coins to be transferred and the recipients count using the --validate-only flag.
The unsigned transactions are dumped to a 'unsigned_txs.json' file by default.

NOTE: You can build these transactions in offline mode using the '--offline' flag.
In this mode, gas estimation is based on empirical results from test runs with different batch sizes.
Also, the sequence and account-number flags are not needed in offline mode.`,
		Example: `mtransferd build-txs --file transfer.json --from my_key_or_addr --batch-size 10000 --sequence 1 --account-number 1`,
		Args:    cobra.NoArgs,
		RunE:    types.RunEWithCtx(runBuildTxCmd),
	}

	cmd.Flags().String(types.FileFlag, "", "Path to JSON file with recipients")
	cmd.Flags().Int(types.BatchSizeFlag, 10000, "Batch size for MultiSend messages")
	cmd.Flags().Int(types.StartIndexFlag, 0, "Start index of the transfer recipient list")
	cmd.Flags().Bool(types.ValidateFlag, false, "Run only transfer file validation and get the total coins to be transferred and recipient count")
	cmd.Flags().String(types.OutputFileFlag, "unsigned_txs.json", "Name of the output file where the txs are dumped")
	cmd.MarkFlagRequired(types.FileFlag)
	cmd.MarkFlagRequired(flags.FlagFrom)

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func runBuildTxCmd(ctx types.Context, cmd *cobra.Command, _ []string) error {
	homePath, err := filepath.Abs(ctx.ClientCtx.HomeDir)
	if err != nil {
		return err
	}

	gasStr, _ := cmd.Flags().GetString(flags.FlagGas)
	gasSetting, _ := flags.ParseGasSetting(gasStr)

	// ignore the default gas (when not specified in the flag)
	if gasSetting.Gas == flags.DefaultGasLimit {
		gasSetting.Gas = 0
	}

	logger := zap.NewExample()
	defer logger.Sync()

	cfg, err := config.ConfigWithFlags(ctx.ClientCtx, homePath, cmd.Flags())
	if err != nil {
		return fmt.Errorf("failed to get app config: %w", err)
	}
	a, err := app.NewApp(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to load app: %w", err)
	}

	doneCh := make(chan error, 1)
	go func() {
		txs, err := a.BuildTxs(ctx, gasSetting.Gas)
		defer func() { doneCh <- err }()
		if err != nil {
			return
		}
		if ctx.ValidateOnly {
			return
		}

		if err := ctx.WriteTxsJSONToOuput(txs); err != nil {
			return
		}

		logger.Info("Done! Unsigned transactions generated")
	}()

	select {
	case <-cmd.Context().Done():
		logger.Info("Shutdown signal received. Stopping build tx process...")
		return nil
	case err := <-doneCh:
		return err
	}
}
