package batchpayment

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/zeebo/errs"
)

const (
	Decimals = 18
)

func HashFromString(s string) (common.Hash, error) {
	h := common.HexToHash(s)
	if h.Hex() != s {
		return common.Hash{}, errs.New("%q is not valid hash", s)
	}
	return h, nil
}

func AddressFromString(s string) (common.Address, error) {
	if !common.IsHexAddress(s) {
		return common.Address{}, errs.New("%q is not a hex address", s)
	}
	return common.HexToAddress(s), nil
}

func PrettyETH(wei *big.Int) string {
	switch {
	case wei.Cmp(big.NewInt(1_000_000_000_000_000)) > 0:
		return fmt.Sprintf("%s ETH", decimal.NewFromBigInt(wei, -18))
	case wei.Cmp(big.NewInt(1_000_000_0)) > 0:
		return fmt.Sprintf("%s GWei", decimal.NewFromBigInt(wei, -9))
	default:
		return fmt.Sprintf("%s Wei", wei)
	}

}
