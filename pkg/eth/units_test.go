package eth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"storj.io/crypto-batch-payment/pkg/eth"
)

func TestParseUnits(t *testing.T) {
	for _, tc := range []struct {
		in  string
		out string
		err string
	}{
		{
			in:  "1.24ETH",
			out: "1.24eth",
		},
		{
			in:  "1.24eth",
			out: "1.24eth",
		},
		{
			in:  "1.24GWEI",
			out: "1.24gwei",
		},
		{
			in:  "1.24gwei",
			out: "1.24gwei",
		},
		{
			in:  "124WEI",
			out: "124wei",
		},
		{
			in:  "1.24e18",
			out: "1240000000000000000wei",
		},
		{
			in:  "",
			err: "invalid unit: empty",
		},
		{
			in:  "foo",
			err: `unsupported suffix "foo"`,
		},
		{
			in:  "1f1eth",
			err: "1f1 is not a valid ETH unit: can't convert 1f1 to decimal",
		},
		{
			in:  "1.24",
			err: "1.24 is not a valid WEI unit: must be a whole number of WEI but got 1.24",
		},
	} {
		t.Run(tc.in, func(t *testing.T) {
			out, err := eth.ParseUnit(tc.in)
			if tc.err != "" {
				assert.EqualError(t, err, tc.err)
				assert.Zero(t, out)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.out, out.String())
		})
	}
}

func TestUnitsExchange(t *testing.T) {
	var (
		weiUnit  = eth.RequireParseUnit("1wei")
		gweiUnit = eth.RequireParseUnit("1gwei")
		ethUnit  = eth.RequireParseUnit("1eth")
	)

	assert.Equal(t, "1wei", weiUnit.WEI().String())
	assert.Equal(t, "0.000000001gwei", weiUnit.GWEI().String())
	assert.Equal(t, "0.000000000000000001eth", weiUnit.ETH().String())
	assert.Equal(t, "1", weiUnit.WEIInt().String())
	assert.Equal(t, "1", weiUnit.Decimal(eth.WEI).String())
	assert.Equal(t, "0.000000001", weiUnit.Decimal(eth.GWEI).String())
	assert.Equal(t, "0.000000000000000001", weiUnit.Decimal(eth.ETH).String())

	assert.Equal(t, "1000000000wei", gweiUnit.WEI().String())
	assert.Equal(t, "1gwei", gweiUnit.GWEI().String())
	assert.Equal(t, "0.000000001eth", gweiUnit.ETH().String())
	assert.Equal(t, "1000000000", gweiUnit.WEIInt().String())
	assert.Equal(t, "1", gweiUnit.Decimal(eth.GWEI).String())
	assert.Equal(t, "0.000000001", gweiUnit.Decimal(eth.ETH).String())

	assert.Equal(t, "1000000000000000000wei", ethUnit.WEI().String())
	assert.Equal(t, "1000000000gwei", ethUnit.GWEI().String())
	assert.Equal(t, "1eth", ethUnit.ETH().String())
	assert.Equal(t, "1000000000000000000", ethUnit.WEIInt().String())
	assert.Equal(t, "1000000000000000000", ethUnit.WEIInt().String())
	assert.Equal(t, "1000000000", ethUnit.Decimal(eth.GWEI).String())
	assert.Equal(t, "1", ethUnit.Decimal(eth.ETH).String())
}
