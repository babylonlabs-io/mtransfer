package app

import (
	"errors"

	"github.com/babylonlabs-io/mtransfer/types"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authclient "github.com/cosmos/cosmos-sdk/x/auth/client"
)

func (a *App) SignTxs(ctx types.Context, txs []sdk.Tx) ([]sdk.Tx, error) {
	logger := a.logger
	p := a.client.Provider()
	txCfg := p.Cdc.TxConfig

	if !ctx.ClientCtx.Offline {
		return nil, errors.New("signing transactions is only supported in offline mode. Make sure to use the '--offline', '--sequence' and '--account-number' flags")
	}
	logger.Info("Signing transactions in offline mode...")

	signer := ctx.Sender
	startIndex := ctx.StartIndex
	f := p.TxFactoryWithDefaults(tx.Factory{}).WithAccountNumber(signer.AccNumber)
	seq := signer.Sequence
	txCount := len(txs)
	signedTxs := make([]sdk.Tx, 0, txCount-startIndex)
	for i := startIndex; i < txCount; i++ {
		txBuilder, err := txCfg.WrapTxBuilder(txs[i])
		if err != nil {
			return nil, err
		}

		f = f.WithSequence(seq)
		err = authclient.SignTx(f, ctx.ClientCtx, ctx.ClientCtx.GetFromName(), txBuilder, ctx.ClientCtx.Offline, ctx.OverwriteSig)
		if err != nil {
			return nil, err
		}
		signedTx := txBuilder.GetTx()
		signedTxs = append(signedTxs, signedTx)
		// increase sequence so subsequent transaction
		// use the next sequence number to ensure validity
		seq++
	}

	return signedTxs, nil
}
