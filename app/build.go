package app

import (
	"context"
	"strings"

	"go.uber.org/zap"

	appparams "github.com/babylonlabs-io/babylon/app/params"
	"github.com/babylonlabs-io/mtransfer/types"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func (a *App) BuildTxs(ctx types.Context, gasWanted uint64) ([]sdk.Tx, error) {
	fromAddr, err := a.getAddrStr(ctx.ClientCtx.FromName)
	if err != nil {
		return nil, err
	}
	logger := a.logger

	logger.Info("Loading and validating the transfer data...")
	// validate the data and build the tx outputs (address <> amount)
	totalAmount, entries, err := ctx.LoadTransferData()
	if err != nil {
		return nil, err
	}

	logger.Info("Transfer data stateless checks passed!",
		zap.String("Total Coins", totalAmount.String()),
		zap.Int("Entries count", len(entries)),
	)

	if ctx.ValidateOnly {
		return nil, nil
	}
	buildMsg := "Building unsigned txs"
	if ctx.ClientCtx.Offline {
		buildMsg += " in offline mode"
	}
	logger.Info(buildMsg)

	txs := make([]sdk.Tx, 0, len(entries))
	for i := ctx.StartIndex; i < len(entries); i += ctx.BatchSize {
		end := i + ctx.BatchSize
		if end > len(entries) {
			end = len(entries)
		}

		batch := entries[i:end]
		batchTotal := sdk.Coins{}
		for _, entry := range batch {
			batchTotal = batchTotal.Add(entry.Coins...)
		}

		input := banktypes.Input{Address: fromAddr, Coins: batchTotal}
		msg := banktypes.NewMsgMultiSend(input, batch)

		tx, err := a.buildTx(ctx, msg, gasWanted, end-i)
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}

	return txs, nil
}

func (a *App) buildTx(ctx types.Context, msg sdk.Msg, gasWanted uint64, batchSize int) (sdk.Tx, error) {
	var err error
	p := a.client.Provider()
	f := p.TxFactoryWithDefaults(tx.Factory{})
	gas := gasWanted
	if gas == 0 {
		gas, err = a.calculateGas(ctx, f, batchSize, msg)
		if err != nil {
			return nil, err
		}
	}
	f = f.WithGas(gas).WithGasPrices(a.cfg.GasPrices)

	txb, err := f.BuildUnsignedTx(msg)
	if err != nil {
		return nil, err
	}

	return txb.GetTx(), nil
}

// getAddrStr is a helper function to validate the provided address
// or get the corresponding address if a key name is provided
func (a *App) getAddrStr(str string) (string, error) {
	// check if from is an address or a key name
	var fromAddr string
	if strings.HasPrefix(str, appparams.Bech32PrefixAccAddr) {
		if _, err := sdk.AccAddressFromBech32(str); err != nil {
			return fromAddr, err
		}
		fromAddr = str
	} else {
		fromAddr = a.mustGetTxSigner(str)
	}
	return fromAddr, nil
}

func (a *App) calculateGas(ctx types.Context, f tx.Factory, batchSize int, msgs ...sdk.Msg) (uint64, error) {
	goCtx := context.Background()
	from := ctx.Sender
	// try to get the gas by simulation
	// Otherwise, estimate it based on empirical results
	if !ctx.ClientCtx.Offline {
		// Need sequence and acc num to simulate the tx to calculate the gas
		f = f.WithSequence(from.Sequence).WithAccountNumber(from.AccNumber)
		_, gas, err := a.client.Provider().CalculateGas(goCtx, f, from.KeyName, msgs...)
		if err != nil {
			return 0, err
		}
		return gas, nil
	}
	return estimateGas(batchSize), nil
}

// estimateGas calculates the estimated gas based on the number of recipients
// of a single MsgMultiSend. This straight line was calculated based on
// different runs with different batch sizes
func estimateGas(recipients int) uint64 {
	m := 22312.5
	b := 375000.0
	estimatedGas := (m*float64(recipients) + b) * 1.20 // Overestimate by 20% to be safe
	return uint64(estimatedGas)
}
