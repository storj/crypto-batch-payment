package csv

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseData(t *testing.T) {
	type dataRow struct {
		address string
		usd     string
	}

	testCases := []struct {
		name string
		csv  string
		rows []dataRow
		err  string
	}{
		{
			name: "bad header",
			csv:  `bad,stuff`,
			err:  `record on line 1: invalid header "bad,stuff"; expected "addr,usdAmnt"`,
		},
		{
			name: "not enough fields",
			csv: `addr,usdAmnt
				0x00112233445566778899aabbccddeeff00112233
			`,
			err: `record on line 2: wrong number of fields`,
		},
		{
			name: "too many enough fields",
			csv: `addr,usdAmnt
				0x00112233445566778899aabbccddeeff00112233,a,b
			`,
			err: `record on line 2: wrong number of fields`,
		},
		{
			name: "bad address is not hex",
			csv: `addr,usdAmnt
				0xgggggg33445566778899aabbccddeeff00112233,600
			`,
			err: `record on line 2: invalid ETH address "0xgggggg33445566778899aabbccddeeff00112233"`,
		},
		{
			name: "bad address is too short",
			csv: `addr,usdAmnt
				0x0011223344,600
			`,
			err: `record on line 2: invalid ETH address "0x0011223344"`,
		},
		{
			name: "bad address is too long",
			csv: `addr,usdAmnt
				0x00112233445566778899aabbccddeeff0011223344,600
			`,
			err: `record on line 2: invalid ETH address "0x00112233445566778899aabbccddeeff0011223344"`,
		},
		{
			name: "bad amount",
			csv: `addr,usdAmnt
				0x00112233445566778899aabbccddeeff00112233,A
			`,
			err: `record on line 2: invalid amount "A": can't convert A to decimal`,
		},
		{
			name: "zero amount",
			csv: `addr,usdAmnt
				0x00112233445566778899aabbccddeeff00112233,0
			`,
			err: `record on line 2: invalid amount "0": must be a positive value`,
		},
		{
			name: "negative amount",
			csv: `addr,usdAmnt
				0x00112233445566778899aabbccddeeff00112233,-2
			`,
			err: `record on line 2: invalid amount "-2": must be a positive value`,
		},
		{
			name: "success",
			csv: `addr,usdAmnt
			# normal address and amount
			0x00112233445566778899aabbccddeeff00112233,1234.56

			# normal address and amount
			0xffeeddccbbaa99887766554433221100ffeeddcc,300

			# address without hex prefix and amount in scientific notation
			1234123412341234123412341234123412341234,1e5

			# address without hex prefix and sub-dollar amount in scientific notation
			1234123412341234123412341234123412341234,5e-05
			`,
			rows: []dataRow{
				{
					address: "00112233445566778899aabbccddeeff00112233",
					usd:     "1234.56",
				},
				{
					address: "ffeeddccbbaa99887766554433221100ffeeddcc",
					usd:     "300",
				},
				{
					address: "1234123412341234123412341234123412341234",
					usd:     "100000",
				},
				{
					address: "1234123412341234123412341234123412341234",
					usd:     "0.00005",
				},
			},
		},
		{
			name: "success with alternate usdAmnt header",
			csv: `addr,amnt
			0x00112233445566778899aabbccddeeff00112233,1234.56
			`,
			rows: []dataRow{
				{
					address: "00112233445566778899aabbccddeeff00112233",
					usd:     "1234.56",
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rows, err := Parse([]byte(testCase.csv))
			if testCase.err != "" {
				require.EqualError(t, err, testCase.err)
				return
			}
			require.NoError(t, err)

			var actualRows []dataRow
			for _, row := range rows {
				actualRows = append(actualRows, dataRow{
					address: fmt.Sprintf("%040x", row.Address),
					usd:     row.USD.String(),
				})
			}
			require.Equal(t, testCase.rows, actualRows)
		})
	}
}
