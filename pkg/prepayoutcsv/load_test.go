package prepayoutcsv_test

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"storj.io/crypto-batch-payment/pkg/prepayoutcsv"
)

const (
	header = `address,amount,address-kind,mandatory,sanctioned,bonus`
)

var (
	goodCSV = []byte(header + `
0xDDc423E04E8A5E581F12453117159666E6EC143b,3.055353,eth,true,false,false
0xD7D8D54F10f2C70e7b0b1dC97B9F2f495D2cBc55,0.169281,zksync,false,true,false
0xA765936150751a8d54B40565cdca559de1416D16,2.356021,zksync-era,false,false,true
,0.000000,eth,true,false,false
`)
)

func TestParse(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		prepayouts, err := prepayoutcsv.Parse(goodCSV)
		require.NoError(t, err)
		require.Equal(t, []prepayoutcsv.Row{
			{
				Line:       2,
				Address:    common.HexToAddress("0xDDc423E04E8A5E581F12453117159666E6EC143b"),
				Kind:       "eth",
				Amount:     decimal.RequireFromString("3.055353"),
				Mandatory:  true,
				Sanctioned: false,
				Bonus:      false,
			},
			{
				Line:       3,
				Address:    common.HexToAddress("0xD7D8D54F10f2C70e7b0b1dC97B9F2f495D2cBc55"),
				Kind:       "zksync",
				Amount:     decimal.RequireFromString("0.169281"),
				Mandatory:  false,
				Sanctioned: true,
				Bonus:      false,
			},
			{
				Line:       4,
				Address:    common.HexToAddress("0xA765936150751a8d54B40565cdca559de1416D16"),
				Kind:       "zksync-era",
				Amount:     decimal.RequireFromString("2.356021"),
				Mandatory:  false,
				Sanctioned: false,
				Bonus:      true,
			},
			{
				Line:       5,
				Address:    common.HexToAddress("0x0000000000000000000000000000000000000000"),
				Kind:       "eth",
				Amount:     decimal.RequireFromString("0.000000"),
				Mandatory:  true,
				Sanctioned: false,
				Bonus:      false,
			},
		}, prepayouts)
	})

	t.Run("bad header", func(t *testing.T) {
		_, err := prepayoutcsv.Parse([]byte("bad,header\nsome,row"))
		require.EqualError(t, err, `record on line 1: invalid header "bad,header": expected "address,amount,address-kind,mandatory,sanctioned,bonus"`)
	})

	t.Run("invalid number of fields", func(t *testing.T) {
		_, err := prepayoutcsv.Parse([]byte(header + "\nsome,row"))
		require.EqualError(t, err, `record on line 2: expected 6 fields but got 2`)
	})

	t.Run("invalid field", func(t *testing.T) {
		testInvalidField := func(t *testing.T, n int, value string, equalErr string) {
			t.Helper()
			fields := []string{"0xDDc423E04E8A5E581F12453117159666E6EC143b", "3.055353", "eth", "true", "false", "false"}
			fields[n] = value
			_, err := prepayoutcsv.Parse([]byte(header + "\n" + strings.Join(fields, ",")))
			require.EqualError(t, err, equalErr)
		}

		t.Run("address", func(t *testing.T) {
			testInvalidField(t, 0, "not-an-address", `record on line 2: invalid ETH address "not-an-address"`)
		})

		t.Run("amount", func(t *testing.T) {
			testInvalidField(t, 1, "not-an-amount", `record on line 2: invalid amount "not-an-amount": can't convert not-an-amount to decimal`)
		})

		t.Run("kind", func(t *testing.T) {
			testInvalidField(t, 2, "", `record on line 2: invalid kind "": cannot be empty`)
		})

		t.Run("mandatory", func(t *testing.T) {
			testInvalidField(t, 3, "not-bool", `record on line 2: invalid boolean value "not-bool" for "mandatory" column`)
		})

		t.Run("sanctioned", func(t *testing.T) {
			testInvalidField(t, 4, "not-bool", `record on line 2: invalid boolean value "not-bool" for "sanctioned" column`)
		})

		t.Run("bonus", func(t *testing.T) {
			testInvalidField(t, 5, "not-bool", `record on line 2: invalid boolean value "not-bool" for "bonus" column`)
		})
	})
}
