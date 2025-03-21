package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/babylonlabs-io/babylon/testutil/datagen"
	"github.com/babylonlabs-io/mtransfer/types"
)

// CmdRandomTransfers returns the command to generate random input.
func CmdRandomTransfers() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "generate-random-txs",
		Short: "generates random transfers input file with baby addr and smal values",
		Args:  cobra.ExactArgs(2), // how many addr, float amount of baby
		RunE:  types.RunEWithCtx(runGenerateRandomTxs),
	}

	now := time.Now().Format("2006-01-02_15-04-05")
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	outputPath := filepath.Join(pwd, fmt.Sprintf("transfer-%s.json", now))
	cmd.Flags().String(types.OutputFileFlag, outputPath, "Name of the output file where the signed txs are dumped")

	return cmd
}

type InputTransfers struct {
	Aggregates InputTransfersAggregate `json:"aggregates"`
}

type InputTransfersAggregate struct {
	TotalBaby float64 `json:"total_baby"`
}

func runGenerateRandomTxs(ctx types.Context, cmd *cobra.Command, args []string) error {
	qntAddrs, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return err
	}

	babyAmount, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return err
	}

	rawData := make(map[string]InputTransfers, qntAddrs)
	for i := 0; i < int(qntAddrs); i++ {
		addr := datagen.GenRandomAddress()

		rawData[addr.String()] = InputTransfers{
			Aggregates: InputTransfersAggregate{
				TotalBaby: babyAmount,
			},
		}
	}

	return ctx.WriteJSONToOutput(rawData)
}
