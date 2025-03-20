package types

import (
	"encoding/json"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Sender struct {
	KeyName   string
	Sequence  uint64
	AccNumber uint64
}

func NewSender(keyName string, seq, accNum uint64) Sender {
	return Sender{
		KeyName:   keyName,
		Sequence:  seq,
		AccNumber: accNum,
	}
}

type TxsJSON struct {
	Txs []any `json:"txs"`
}

func NewTxsJSON(txs []sdk.Tx, ec sdk.TxEncoder) (*TxsJSON, error) {
	if ec == nil {
		return nil, errors.New("cannot print unsigned tx: tx json encoder is nil")
	}

	// Collect transactions as JSON objects
	var txList []any
	for _, tx := range txs {
		jsonBytes, err := ec(tx)
		if err != nil {
			return nil, err
		}

		var txData any
		if err := json.Unmarshal(jsonBytes, &txData); err != nil {
			return nil, fmt.Errorf("failed to decode transaction JSON: %w", err)
		}

		txList = append(txList, txData)
	}

	return &TxsJSON{Txs: txList}, nil
}
