package payer

import (
	"strings"

	"github.com/zeebo/errs"
)

// Type represents the payment method.
type Type string

const (
	Eth        Type = "eth"
	Sim        Type = "sim"
	ZkSync     Type = "zksync"
	ZkSyncEra  Type = "zksync-era"
	ZkWithdraw Type = "zkwithdraw"
	Polygon    Type = "polygon"
)

func (pt Type) String() string {
	return string(pt)
}

// TypeFromString parses string to a Type const.
func TypeFromString(t string) (Type, error) {
	switch strings.ToLower(t) {
	case "eth":
		return Eth, nil
	case "polygon":
		return Polygon, nil
	case "zksync":
		return ZkSync, nil
	case "zksync-era", "zksync2": // zksync2 for backcompat
		return ZkSyncEra, nil
	case "zkwithdraw":
		return ZkWithdraw, nil
	case "sim":
		return Sim, nil
	default:
		return "", errs.New("invalid payer type %q", t)
	}
}
