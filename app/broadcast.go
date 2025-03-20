package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/babylonlabs-io/babylon/client/babylonclient"
	"github.com/babylonlabs-io/mtransfer/types"
	"github.com/cometbft/cometbft/mempool"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/zap"
)

var retryableErrors = []string{
	babylonclient.ErrTimeoutAfterWaitingForTxBroadcast.Error(),
	"connection refused",
}

func (a *App) BroadcastTxs(ctx types.Context, txs []sdk.Tx) error {
	var (
		logger  = a.logger
		txCount = len(txs)
	)

	logger.Info("Broadcasting transactions...")
	for i := ctx.StartIndex; i < txCount; i++ {
		res, err := a.broadcastTx(txs[i], i)
		if err != nil {
			if strings.Contains(err.Error(), mempool.ErrTxInCache.Error()) {
				logger.Info("Transaction already in mempool", zap.Int("Index", i))
				continue
			}
			logger.Error("Transaction failed", zap.Int("Index", i), zap.String("Error", err.Error()))
			return err
		}

		logger.Info("Transaction sent to mempool", zap.String("TxHash", res.TxHash), zap.Uint32("Code", res.Code), zap.Int("Index", i))

		txIncluded, err := a.isTxIncluded(res.TxHash)
		if err != nil {
			return err
		}
		if !txIncluded {
			logger.Error("Transaction was not included", zap.Int("Index", i))
			return errors.New("transaction not included in block")
		}
	}

	return nil
}

func (a *App) broadcastTx(tx sdk.Tx, index int) (*babylonclient.RelayerTxResponse, error) {
	var (
		res        *babylonclient.RelayerTxResponse
		p          = a.client.Provider()
		timeout    = 35 * time.Second // tiemout to wait for tx to be included in block
		maxRetries = 5
		logger     = a.logger
		ctx        = context.Background()
		enc        = p.Cdc.TxConfig.TxEncoder()
	)

	txBz, err := enc(tx)
	if err != nil {
		return res, err
	}
	// if getting a retryable error, retry up to maxRetries
	for attempt := 1; attempt <= maxRetries; attempt++ {
		var (
			callbackErr error
			wg          sync.WaitGroup
		)

		wg.Add(1)
		callback := func(rtr *babylonclient.RelayerTxResponse, err error) {
			defer wg.Done()
			res = rtr
			callbackErr = err
		}

		if err := p.BroadcastTx(ctx, txBz, ctx, timeout, []func(*babylonclient.RelayerTxResponse, error){callback}); err != nil {
			if isRetryableError(err) && attempt < maxRetries {
				logger.Warn("Transaction error, retrying...",
					zap.Int("Index", index),
					zap.Int("Attempt", attempt),
					zap.String("Error", err.Error()),
				)
				time.Sleep(time.Duration(1<<attempt) * time.Second) // Exponential backoff
				continue
			}
			return res, err
		}
		wg.Wait()
		if callbackErr != nil {
			if isRetryableError(callbackErr) && attempt < maxRetries {
				logger.Warn("Transaction error, retrying...",
					zap.Int("Index", index),
					zap.Int("Attempt", attempt),
					zap.String("Error", callbackErr.Error()),
				)
				time.Sleep(time.Duration(1<<attempt) * time.Second)
				continue
			}
			return res, callbackErr
		}
		// all good, we can stop retry loop
		break
	}
	if res == nil {
		// this case could happen if the error within the retry is an expected error
		return res, errors.New("transaction response is empty")
	}
	if res.Code != 0 {
		return res, fmt.Errorf("transaction failed with code: %d", res.Code)
	}
	return res, nil
}

// isRetryableError checks if the error message matches any of the retryable errors.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	for _, retryErr := range retryableErrors {
		if strings.Contains(errMsg, retryErr) {
			return true
		}
	}
	return false
}
