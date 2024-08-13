package receipts_test

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/receipts"
)

func TestBuffer(t *testing.T) {
	address1 := common.BytesToAddress(bytes.Repeat([]byte{1}, common.AddressLength))
	address2 := common.BytesToAddress(bytes.Repeat([]byte{2}, common.AddressLength))
	address3 := common.BytesToAddress(bytes.Repeat([]byte{3}, common.AddressLength))

	var b receipts.Buffer
	b.Emit(address1, decimal.NewFromInt(1), "hash1", payer.Eth)
	b.Emit(address2, decimal.NewFromInt(2), "hash2", payer.Eth)
	b.Emit(address3, decimal.NewFromInt(3), "hash3", payer.ZkSyncEra)
	receipts := b.Finalize()
	require.Equal(t, `wallet,amount,txhash,mechanism
0x0101010101010101010101010101010101010101,1,hash1,eth
0x0202020202020202020202020202020202020202,2,hash2,eth
0x0303030303030303030303030303030303030303,3,hash3,zksync-era
`, string(receipts))
}
