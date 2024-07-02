package payouts

import (
	"storj.io/crypto-batch-payment/pkg/csv"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
)

func FromCSV(rows []csv.Row) []*pipelinedb.Payout {
	payouts := make([]*pipelinedb.Payout, 0, len(rows))
	for _, row := range rows {
		payouts = append(payouts, &pipelinedb.Payout{
			CSVLine: row.Line,
			Payee:   row.Address,
			USD:     row.USD,
		})
	}
	return payouts
}
