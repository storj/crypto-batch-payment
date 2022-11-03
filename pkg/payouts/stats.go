package payouts

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"

	"storj.io/crypto-batch-payment/pkg/csv"
)

type RowStats struct {
	// Addresses is the number of addresses
	Addresses int

	// Payouts is the number of payouts to addresses
	Payouts int

	// USD is the total USD of all payouts
	USD decimal.Decimal
}

func GetRowStats(rows []csv.Row) RowStats {
	addresses := make(map[common.Address]struct{})
	usd := decimal.New(0, 0)

	for _, row := range rows {
		addresses[row.Address] = struct{}{}
		usd.Add(row.USD)
	}

	return RowStats{
		Addresses: len(addresses),
		Payouts:   len(rows),
		USD:       usd,
	}
}
