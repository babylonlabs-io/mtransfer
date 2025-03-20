package app

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	bbnclient "github.com/babylonlabs-io/babylon/client/client"
	bbncfg "github.com/babylonlabs-io/babylon/client/config"

	akr "github.com/babylonlabs-io/mtransfer/keyring"
	"github.com/babylonlabs-io/mtransfer/types"
)

type App struct {
	client *bbnclient.Client
	cfg    bbncfg.BabylonConfig
	kr     keyring.Keyring
	logger *zap.Logger
}

func NewApp(
	cfg bbncfg.BabylonConfig,
	logger *zap.Logger,
) (*App, error) {

	bc, err := bbnclient.New(
		&cfg,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Babylon client: %w", err)
	}

	kr, err := akr.CreateKeyring(
		cfg.KeyDirectory,
		cfg.ChainID,
		cfg.KeyringBackend,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create keyring: %w", err)
	}

	return &App{
		client: bc,
		cfg:    cfg,
		kr:     kr,
		logger: logger,
	}, nil
}

func (a *App) Start(ctx types.Context) error {
	fromAddr := a.mustGetTxSigner(ctx.ClientCtx.FromName)
	logger := a.logger

	logger.Info("Loading and validating the transfer data...")
	// validate the data and build the tx outputs (address <> amount)
	totalAmount, entries, err := ctx.LoadTransferData()
	if err != nil {
		return err
	}

	logger.Info("Transfer data stateless checks passed!",
		zap.String("Total Coins", totalAmount.String()),
		zap.Int("Entries count", len(entries)),
	)

	if ctx.ValidateOnly {
		return nil
	}

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

		var txIncluded bool
		for !txIncluded {
			input := banktypes.Input{Address: fromAddr, Coins: batchTotal}
			msg := banktypes.NewMsgMultiSend(input, batch)

			logger.Info("Sending MultiSend tx",
				zap.Int("Starting index", i),
				zap.String("Starting address", entries[i].Address),
				zap.Int("Ending index", end-1),
				zap.String("Ending address", entries[end-1].Address),
				zap.String("Batch total coins", batchTotal.String()),
			)

			res, err := a.client.ReliablySendMsg(context.Background(), msg, nil, nil)
			if err != nil {
				logger.Error("Transaction failed", zap.Int("Start index", i), zap.Int("End index", end-1))
				return err
			}
			logger.Info("Transaction sent to mempool", zap.String("TxHash", res.TxHash), zap.Uint32("Code", res.Code))

			// check tx was included in a block before sending the next one
			txIncluded, err = a.isTxIncluded(res.TxHash)
			if err != nil {
				return err
			}
			if !txIncluded {
				logger.Error("Transaction was not included. Resending...")
			}
		}
	}

	logger.Info("Transfer completed successfully")
	return nil
}

func (a *App) mustGetTxSigner(key string) string {
	if key == "" {
		key = a.cfg.Key
	}
	signerKey, err := a.client.GetKeyring().Key(key)
	if err != nil {
		panic(fmt.Sprintf("Failed to get signer key: %s", err))
	}

	addr, err := signerKey.GetAddress()
	if err != nil {
		panic(fmt.Sprintf("Failed to get signer key address: %s", err))
	}
	prefix := a.cfg.AccountPrefix

	return sdk.MustBech32ifyAddressBytes(prefix, addr)
}

// isTxIncluded retries querying the latest block until it succeeds or context is canceled.
func (a *App) isTxIncluded(txHash string) (bool, error) {
	logger := a.logger
	waitTime := 35 * time.Second
	hashBz, err := hex.DecodeString(txHash)
	if err != nil {
		logger.Error("invalid tx hash", zap.String("Hash", txHash))
		return false, err
	}

	res, err := a.client.Provider().WaitForBlockInclusion(context.Background(), hashBz, waitTime)
	if err != nil {
		return false, err
	}

	// successful code is == 0
	if res.Code != 0 {
		logger.Warn("Error in tx execution", zap.String("TxHash", txHash), zap.Uint32("Code", res.Code))
		return false, nil
	}
	logger.Info("Tx included in block", zap.Int64("Height", res.Height), zap.Int64("Gas Used", res.GasUsed))
	return true, nil
}
