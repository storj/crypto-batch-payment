package zksync

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_packBigInt(t *testing.T) {
	require.Equal(t, "2000", hex.EncodeToString(packBigInt(big.NewInt(1), uint8(5), uint8(11))))
	require.Equal(t, "4001", hex.EncodeToString(packBigInt(big.NewInt(10), uint8(5), uint8(11))))
	require.Equal(t, "800c", hex.EncodeToString(packBigInt(big.NewInt(100), uint8(5), uint8(11))))
	require.Equal(t, "459a", hex.EncodeToString(packBigInt(big.NewInt(123456789), uint8(5), uint8(11))))
	require.Equal(t, "a34d", hex.EncodeToString(packBigInt(big.NewInt(621000), uint8(5), uint8(11))))
	require.Equal(t, "00ee0d1300", hex.EncodeToString(packBigInt(big.NewInt(9990000), uint8(5), uint8(35))))
}

func Test_unpack(t *testing.T) {
	tests := []struct {
		input    []byte
		split    uint8
		expected *big.Int
	}{
		{
			input:    []byte{0x00, 0xee, 0x0d, 0x13, 0x00},
			split:    5,
			expected: big.NewInt(9990000),
		},
		{
			input:    []byte{0x20, 00},
			split:    5,
			expected: big.NewInt(1),
		},
	}

	for _, testCase := range tests {
		t.Run(fmt.Sprintf("Unpack %s", testCase.expected), func(t *testing.T) {
			res := unpackBigInt(testCase.input, testCase.split)
			require.Equal(t, testCase.expected, res)
		})
	}
}

func Test_packUnpack(t *testing.T) {
	mBits := uint8(11)
	expBits := uint8(5)
	for i := 0; i < 20; i++ {
		for _, k := range []int{1, 2, 16, 100, 1000, 10000} {
			original := big.NewInt(int64(k * i))
			t.Run(fmt.Sprintf("Pack and unpack %s", original.String()), func(t *testing.T) {
				packed := packBigInt(original, expBits, mBits)
				unpacked := unpackBigInt(packed, expBits)
				require.Equal(t, original, unpacked)
			})
		}
	}
}

func Test_splitByBits(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		split uint8
		left  []byte
		right []byte
	}{
		{
			name:  "Split two bytes by 5",
			input: []byte{0b001_10011, 0b11001100},
			split: 5,
			left:  []byte{0b00000110, 0b01100001},
			right: []byte{0b00010011},
		},
		{
			name:  "Split two bytes by 7",
			input: []byte{0b0_0110011, 0b11001100},
			split: 7,
			left:  []byte{0b1, 0b10011000},
			right: []byte{0b0110011},
		},
		{
			name:  "Split two bytes by 8",
			input: []byte{0b00110011, 0b00110011},
			split: 8,
			left:  []byte{0b00110011},
			right: []byte{0b00110011},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			left, right := splitByBits(testCase.input, testCase.split)
			require.Equal(t, left, testCase.left)
			require.Equal(t, right, testCase.right)
		})
	}
}

func Test_closestPackableAmount(t *testing.T) {
	failedTxValue, ok := new(big.Int).SetString("36677776552", 10)
	require.True(t, ok)

	tests := []struct {
		input    *big.Int
		mantissa uint8
		exp      uint8
		expected *big.Int
	}{
		{
			input:    big.NewInt(256),
			mantissa: 11,
			exp:      5,
			expected: big.NewInt(256),
		},
		{
			input:    big.NewInt(1024),
			mantissa: 11,
			exp:      5,
			expected: big.NewInt(1024),
		},
		{
			input:    big.NewInt(102400),
			mantissa: 11,
			exp:      5,
			expected: big.NewInt(102400),
		},
		{
			input:    big.NewInt(4095), // 12 bits
			mantissa: 11,
			exp:      5,
			expected: big.NewInt(4090),
		},
		{
			input:    big.NewInt(4099),
			mantissa: 11,
			exp:      5,
			expected: big.NewInt(4090),
		},
		{
			input:    failedTxValue,
			mantissa: 35,
			exp:      5,
			expected: big.NewInt(36677776550),
		},
	}

	for _, testCase := range tests {
		t.Run(fmt.Sprintf("Closest amount %s", testCase.input), func(t *testing.T) {
			res := closestPackableAmount(testCase.input, testCase.exp, testCase.mantissa)
			require.Equal(t, testCase.expected, res)
		})
	}
}

func Test_closestHigherPackableAmountRandom(t *testing.T) {

}
