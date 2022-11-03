package pipeline

import (
	"fmt"
	"strings"

	"github.com/zeebo/errs"
)

type NodeType string

const (
	Geth   NodeType = "geth"
	Parity NodeType = "parity"
)

func (t NodeType) String() string {
	return string(t)
}

func NodeTypeFromString(s string) (NodeType, error) {
	switch strings.ToLower(s) {
	case "geth":
		return Geth, nil
	case "parity":
		return Parity, nil
	default:
		return "", errs.New("invalid node type %q", s)
	}
}

// PriceBumpRatio is used to calculate low gas price. If a transaction gas
// price, multiplied by the PriceBumpRatio, is less than the current gas price
// then it is in danger of being dropped from the geth or parity txpool.
func (t NodeType) PriceBumpRatio() float64 {
	switch t {
	case Geth:
		return 1.1
	case Parity:
		return 1.125
	default:
		panic(fmt.Errorf("unhandled price bump ratio for %q node", t))
	}
}
