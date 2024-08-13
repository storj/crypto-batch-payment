package eth

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/zeebo/errs"
)

var (
	suffixRE = regexp.MustCompile(`([a-zA-Z]+)$`)
)

// ParseUnit returns WEI amount from string. Units accepted are:
// - ETH
// - GWEI
// - WEI
//
// If no unit is specified, WEI is assumed.
func ParseUnit(s string) (Unit, error) {
	if s == "" {
		return Unit{}, errs.New("invalid unit: empty")
	}

	var rawSuffix string
	if m := suffixRE.FindStringSubmatch(s); m != nil {
		rawSuffix = m[1]
	}

	s = s[:len(s)-len(rawSuffix)]

	suffix := strings.ToUpper(rawSuffix)
	if suffix == "" {
		suffix = "WEI"
	}

	var denom Denom
	switch suffix {
	case "ETH":
		denom = ETH
	case "GWEI":
		denom = GWEI
	case "WEI":
		denom = WEI
	default:
		return Unit{}, errs.New("unsupported suffix %q", rawSuffix)
	}

	raw, err := decimal.NewFromString(s)
	if err != nil {
		return Unit{}, errs.New("%s is not a valid %s unit: %v", s, suffix, err)
	}
	raw = raw.Shift(int32(denom))

	wei := raw.Truncate(0)
	if !wei.Equal(raw) {
		return Unit{}, errs.New("%s is not a valid %s unit: must be a whole number of WEI but got %s", s, suffix, raw)
	}
	return Unit{wei: wei, denom: denom}, nil
}

func RequireParseUnit(s string) Unit {
	u, err := ParseUnit(s)
	if err != nil {
		panic(err)
	}
	return u
}

type Unit struct {
	wei   decimal.Decimal
	denom Denom
}

func UnitFromInt(value int64, denom Denom) Unit {
	return UnitFromDecimal(decimal.NewFromInt(value), denom)
}

func UnitFromBigInt(value *big.Int, denom Denom) Unit {
	return UnitFromDecimal(decimal.NewFromBigInt(value, 0), denom)
}

func UnitFromDecimal(value decimal.Decimal, denom Denom) Unit {
	if denom.String() == "" {
		panic("denom is not one of WEI, GWEI, ETH")
	}
	wei := value.Shift(int32(denom))
	return Unit{wei: wei, denom: denom}
}

func (u Unit) Mul(x Unit) Unit {
	wei := u.wei.Mul(x.wei)
	return Unit{wei: wei, denom: u.denom}
}

func (u Unit) WEI() Unit {
	return Unit{wei: u.wei, denom: WEI}
}

func (u Unit) GWEI() Unit {
	return Unit{wei: u.wei, denom: GWEI}
}

func (u Unit) ETH() Unit {
	return Unit{wei: u.wei, denom: ETH}
}

func (u Unit) IsZero() bool {
	return u.wei.IsZero()
}

func (u Unit) String() string {
	return fmt.Sprintf("%s%s", u.wei.Shift(-int32(u.denom)), u.denom)
}

func (u Unit) Decimal(denom Denom) decimal.Decimal {
	return u.wei.Shift(-int32(denom))
}

func (u Unit) WEIInt() *big.Int {
	return u.wei.BigInt()
}

func DecimalToWEIInt(value decimal.Decimal, denom Denom) *big.Int {
	return UnitFromDecimal(value, denom).WEIInt()
}

type Denom int32

const (
	WEI  Denom = 0
	GWEI Denom = 9
	ETH  Denom = 18
)

func (d Denom) String() string {
	switch d {
	case WEI:
		return "wei"
	case GWEI:
		return "gwei"
	case ETH:
		return "eth"
	}
	return ""
}
