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

// CommandSignTxs returns the sign-txs command of mtransferd daemon.
func CommandSignTxs() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "sign-txs",
		Short: "Sign the transactions provided.",
		Long: `Sign transactions located in the provided file and signer key.
The signed transactions are dumped to a 'signed_txs.json' file by default.

NOTE: Only offline signing is supported at the moment. Transactions are generated sequentially, starting with the specified sequence number. Each subsequent transaction will use the next sequence number to ensure validity.`,
		Example: `mtransferd sign-txs --file unsigned_txs.json --from my_key --chain-id bbn-test-1 --offline --account-number 1 --sequence 2`,
		Args:    cobra.NoArgs,
		RunE:    types.RunEWithCtx(runSignTxsCmd),
	}

	cmd.Flags().String(types.FileFlag, "", "Path to JSON file with unsigned transactions")
	cmd.Flags().Int(types.StartIndexFlag, 0, "Start index of the tx in the list to sign")
	cmd.Flags().String(types.OutputFileFlag, "signed_txs.json", "Name of the output file where the signed txs are dumped")
	cmd.MarkFlagRequired(types.FileFlag)
	cmd.MarkFlagRequired(flags.FlagFrom)
	cmd.MarkFlagRequired(flags.FlagOffline)
	cmd.MarkFlagRequired(flags.FlagAccountNumber)
	cmd.MarkFlagRequired(flags.FlagSequence)

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func runSignTxsCmd(ctx types.Context, cmd *cobra.Command, _ []string) error {
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
		signedTxs, err := a.SignTxs(ctx, txs)
		if err != nil {
			return
		}

		if err := ctx.WriteTxsJSONToOuput(signedTxs); err != nil {
			return
		}

		logger.Info("Done signing the transactions")
	}()

	select {
	case <-cmd.Context().Done():
		logger.Info("Shutdown signal received. Stopping signing tx process...")
		return nil
	case err := <-doneCh:
		return err
	}
}
