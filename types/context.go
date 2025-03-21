package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"

	appparams "github.com/babylonlabs-io/babylon/app/params"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Context struct {
	ClientCtx      client.Context
	InputFilePath  string
	OutputFilePath string
	Sender         Sender
	StartIndex     int
	BatchSize      int
	OverwriteSig   bool
	ValidateOnly   bool
}

// RunEWithCtx runs cmd with context and returns an error.
func RunEWithCtx(
	fRunWithCtx func(ctx Context, cmd *cobra.Command, args []string) error,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx, err := getContext(cmd)
		if err != nil {
			return err
		}

		return fRunWithCtx(ctx, cmd, args)
	}
}

func newContext(clientCtx client.Context) Context {
	return Context{ClientCtx: clientCtx}
}

func getContext(cmd *cobra.Command) (Context, error) {
	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		return Context{}, err
	}

	ctx := newContext(clientCtx)
	return readFlags(ctx, cmd.Flags())
}

// readFlags populates the context fields based on flag values
func readFlags(ctx Context, flagSet *pflag.FlagSet) (Context, error) {
	filePath, _ := flagSet.GetString(FileFlag)
	if filePath != "" {
		ctx.InputFilePath = filePath
	}

	batchSize, _ := flagSet.GetInt(BatchSizeFlag)
	ctx.BatchSize = batchSize

	outputFile, _ := flagSet.GetString(OutputFileFlag)
	if outputFile != "" {
		ctx.OutputFilePath = outputFile
	}
	startIdx, _ := flagSet.GetInt(StartIndexFlag)
	ctx.StartIndex = startIdx

	validateOnly, _ := flagSet.GetBool(ValidateFlag)
	ctx.ValidateOnly = validateOnly

	overwrite, _ := flagSet.GetBool(OverwriteFlag)
	ctx.OverwriteSig = overwrite

	accNum, _ := flagSet.GetUint64(flags.FlagAccountNumber)
	seq, _ := flagSet.GetUint64(flags.FlagSequence)
	ctx.Sender = NewSender(ctx.ClientCtx.GetFromName(), seq, accNum)

	return ctx, nil
}

// GetTxsFromInput reads a JSON file and decodes it into []sdk.Tx
func (ctx Context) GetTxsFromInput() ([]sdk.Tx, error) {
	dc := ctx.ClientCtx.TxConfig.TxJSONDecoder()
	if dc == nil {
		return nil, fmt.Errorf("tx decoder is nil")
	}
	if ctx.InputFilePath == "" {
		return nil, errors.New("could not find input file")
	}
	// Read file content
	data, err := os.ReadFile(ctx.InputFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal into TxsJSON
	var txsJSON TxsJSON
	if err := json.Unmarshal(data, &txsJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Decode transactions
	var txs []sdk.Tx
	for _, txData := range txsJSON.Txs {
		txBytes, err := json.Marshal(txData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal txData: %w", err)
		}

		tx, err := dc(txBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to decode transaction: %w", err)
		}

		txs = append(txs, tx)
	}

	return txs, nil
}

func (ctx Context) LoadTransferData() (sdk.Coins, []banktypes.Output, error) {
	if ctx.InputFilePath == "" {
		return nil, nil, errors.New("could not find input file")
	}

	data, err := os.ReadFile(ctx.InputFilePath)
	if err != nil {
		return nil, nil, err
	}

	var rawData map[string]struct {
		Aggregates struct {
			TotalBaby float64 `json:"total_baby"`
		} `json:"aggregates"`
	}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, nil, err
	}

	// Convert map to slice
	totalAmount := sdk.Coins{}
	entries := make([]banktypes.Output, 0, len(rawData))
	for address, data := range rawData {
		// if amount is zero skip it
		if data.Aggregates.TotalBaby == 0 {
			continue
		}
		coins, err := sdk.ParseCoinsNormalized(fmt.Sprintf("%f%s", data.Aggregates.TotalBaby, appparams.HumanCoinUnit))
		if err != nil {
			return nil, nil, fmt.Errorf("invalid amount for address %s: %w", address, err)
		}
		totalAmount = totalAmount.Add(coins...)
		accAddr, err := sdk.AccAddressFromBech32(address)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid address %s: %w", address, err)
		}
		entries = append(entries, banktypes.NewOutput(accAddr, coins))
	}

	// Sort by address so the order is deterministic
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Address < entries[j].Address
	})

	return totalAmount, entries, nil
}

func (ctx Context) WriteTxsJSONToOuput(txs []sdk.Tx) error {
	ec := ctx.ClientCtx.TxConfig.TxJSONEncoder()
	txsJSON, err := NewTxsJSON(txs, ec)
	if err != nil {
		return err
	}
	return ctx.WriteJSONToOutput(txsJSON)
}

func (ctx Context) WriteJSONToOutput(data any) error {
	outputData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal transactions: %w", err)
	}

	// Write buffer content to output file if specified, otherwise print to stdout
	if ctx.OutputFilePath == "" {
		fmt.Println(string(outputData))
		return nil
	}

	if err := os.WriteFile(ctx.OutputFilePath, outputData, 0644); err != nil {
		return fmt.Errorf("failed to write to output file: %w", err)
	}

	return nil
}
