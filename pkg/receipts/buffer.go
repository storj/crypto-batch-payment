package receipts

import (
	"bytes"
	"encoding/csv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"storj.io/crypto-batch-payment/pkg/payer"
)

type Buffer struct {
	buf bytes.Buffer
	csv *csv.Writer
}

func (b *Buffer) Emit(wallet common.Address, amount decimal.Decimal, txHash string, mechanism payer.Type) {
	b.init()
	b.write(wallet.String(), amount.String(), txHash, mechanism.String())
}

func (b *Buffer) Finalize() []byte {
	b.csv.Flush()
	return b.buf.Bytes()
}

func (b *Buffer) init() {
	if b.csv == nil {
		b.csv = csv.NewWriter(&b.buf)
		b.write("wallet", "amount", "txhash", "mechanism")
	}
}

func (b *Buffer) write(c1, c2, c3, c4 string) {
	_ = b.csv.Write([]string{c1, c2, c3, c4})
}
