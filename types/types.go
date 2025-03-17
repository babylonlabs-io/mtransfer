package types

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	appparams "github.com/babylonlabs-io/babylon/app/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func LoadTransferData(filename string) (sdk.Coins, []banktypes.Output, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, err
	}

	var rawData map[string]struct {
		Aggregates struct {
			TotalBaby float64 `json:"total_baby"`
		} `json:"aggregates"`
	}
	if err := json.Unmarshal(file, &rawData); err != nil {
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
