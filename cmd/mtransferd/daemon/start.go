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

// CommandStart returns the start command of mtransferd daemon.
func CommandStart() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "start",
		Short: "Start the transfer process.",
		Long: `Start the transfer process with the provided file, signer key and batch sizes.
This process will automatically generate the transactions, sign and broadcast them.
If want a more step-by-step approach, checkout the other commands ('build-txs', 'sign-txs' and 'broadcast-txs')

You can also run the transfer file validation to get 
the total coins to be transferred and the recipients count using the --validate-only flag`,
		Example: `mtransferd start --file transfer.json --from my_key --batch-size 10000`,
		Args:    cobra.NoArgs,
		RunE:    types.RunEWithCtx(runStartCmd),
	}

	cmd.Flags().String(types.FileFlag, "", "Path to JSON file with recipients")
	cmd.Flags().Int(types.BatchSizeFlag, 10000, "Batch size for MultiSend messages")
	cmd.Flags().Int(types.StartIndexFlag, 0, "Start index of the transfer recipient list")
	cmd.Flags().Bool(types.ValidateFlag, false, "Run only transfer file validation and get the total coins to be transferred and recipient count")
	cmd.MarkFlagRequired(types.FileFlag)

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func runStartCmd(ctx types.Context, cmd *cobra.Command, _ []string) error {
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

	// Run Start in a separate goroutine to monitor context cancellation
	doneCh := make(chan error, 1)
	go func() {
		doneCh <- a.Start(ctx)
	}()

	select {
	case <-cmd.Context().Done():
		logger.Info("Shutdown signal received. Stopping transfer process...")
		return nil
	case err := <-doneCh:
		return err
	}
}
