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

// CommandBroadcastTxs broadcasts the signed txs.
func CommandBroadcastTxs() *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "broadcast-txs",
		Short:   "Broadcast the signed transactions provided.",
		Long:    `Broadcast the transactions located in the provided file.`,
		Example: `mtransferd broadcast-txs --file signed_txs.json`,
		Args:    cobra.NoArgs,
		RunE:    types.RunEWithCtx(runBroadcastTxsCmd),
	}

	cmd.Flags().String(types.FileFlag, "", "Path to JSON file with unsigned transactions")
	cmd.Flags().Int(types.StartIndexFlag, 0, "Start index of the tx in the list to sign")
	cmd.MarkFlagRequired(types.FileFlag)

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func runBroadcastTxsCmd(ctx types.Context, cmd *cobra.Command, _ []string) error {
	homePath, err := filepath.Abs(ctx.ClientCtx.HomeDir)
	if err != nil {
		return err
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
		logger.Info("Getting transactions from file...")
		txs, err := ctx.GetTxsFromInput()
		defer func() { doneCh <- err }()
		if err != nil {
			return
		}
		err = a.BroadcastTxs(ctx, txs)
		if err != nil {
			return
		}

	}()

	select {
	case <-cmd.Context().Done():
		logger.Info("Shutdown signal received. Stopping broadcasting process...")
		return nil
	case err := <-doneCh:
		return err
	}
}
