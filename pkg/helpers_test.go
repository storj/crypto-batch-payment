package batchpayment

import (
	"math/big"
	"testing"
)

func TestPrettyETH(t *testing.T) {
	tests := []struct {
		wei  *big.Int
		want string
	}{
		{
			wei:  big.NewInt(123_000_000_000),
			want: "123 GWei",
		},
		{
			wei:  big.NewInt(123_000_000),
			want: "0.123 GWei",
		},
		{
			wei:  big.NewInt(123_000_000_000_000),
			want: "123000 GWei",
		},
		{
			wei:  big.NewInt(123_000_000_000_000_0),
			want: "0.00123 ETH",
		},
		{
			wei:  big.NewInt(8),
			want: "8 Wei",
		},
	}

	for _, tt := range tests {
		t.Run(tt.wei.String(), func(t *testing.T) {
			if got := PrettyETH(tt.wei); got != tt.want {
				t.Errorf("PrettyETH() = %v, want %v", got, tt.want)
			}
		})
	}
}
