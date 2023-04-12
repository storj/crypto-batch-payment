package payer

import (
	"strings"

	"github.com/zeebo/errs"
)

// PayerType represents the payment method.
type PayerType string

const (
	Eth     PayerType = "eth"
	Sim     PayerType = "sim"
	ZkSync  PayerType = "zksync"
	ZkSync2 PayerType = "zksync2"
	// specific zksync payment which pays to eth from zksync
	ZkWithdraw PayerType = "zkwithdraw"
	Polygon    PayerType = "polygon"
)

// PayerTypeFromString parses string to a PayerType const.
func PayerTypeFromString(t string) (PayerType, error) {
	switch strings.ToLower(t) {
	case "eth":
		return Eth, nil
	case "polygon":
		return Polygon, nil
	case "zksync":
		return ZkSync, nil
	case "zksync2":
		return ZkSync2, nil
	case "zkwithdraw":
		return ZkWithdraw, nil
	case "sim":
		return Sim, nil
	default:
		return "", errs.New("invalid payer type %q", t)
	}
}
