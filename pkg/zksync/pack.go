package zksync

import (
	"fmt"
	"math/big"
)

// packBigInt packs the value to expBits + mantissaBits.
func packBigInt(value *big.Int, expBits uint8, mantissaBits uint8) []byte {
	maxMantissa := new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(mantissaBits)), nil)
	mantissa := value
	base := big.NewInt(10)
	exp := big.NewInt(0)
	for mantissa.Cmp(maxMantissa) > 0 {
		mantissa = new(big.Int).Div(mantissa, base)
		exp = new(big.Int).Add(exp, big.NewInt(1))
	}
	result := new(big.Int).Lsh(mantissa, uint(expBits))
	result = new(big.Int).Add(result, exp)
	raw := result.Bytes()
	size := (expBits + mantissaBits) / 8
	resultBytes := make([]byte, size)
	for i := 0; i < int(size); i++ {
		if len(raw) > i {
			resultBytes[i] = raw[len(raw)-i-1]
		}
	}
	return resultBytes
}

// unpackBigInt unpacks the packed value.
func unpackBigInt(value []byte, expBits uint8) *big.Int {
	left, right := splitByBits(value, expBits)
	exp := new(big.Int).SetBytes(right)
	mantissa := new(big.Int).SetBytes(left)
	// mantissa * 10^exp
	return new(big.Int).Mul(mantissa, new(big.Int).Exp(big.NewInt(10), exp, nil))
}

func closestPackableAmount(value *big.Int, expBits uint8, mantissaBits uint8) *big.Int {
	unpacked := packBigInt(value, expBits, mantissaBits)
	res := unpackBigInt(unpacked, expBits)

	// this should never even happen. The difference can be <9 on the last digit of a number with mantissaBits
	// keeping it here for safety
	percentageDiff := new(big.Int).Sub(new(big.Int).Div(value, res), big.NewInt(1))
	if percentageDiff.Cmp(big.NewInt(1)) > 0 {
		panic(fmt.Sprintf("closestPackableAmount is failing, too big differences between old and new values (%s %s", value.String(), res.String()))
	}
	return res
}

// splitByBits byte array by a bit number
// for example [abcdefgh pqrstuwx],5 should return with [000defgh], [pqr stuwxabc].
func splitByBits(value []byte, splitPos uint8) ([]byte, []byte) {
	split := splitPos
	shift := uint8(0)
	var left, right []byte
	for _, x := range value {
		if split >= 8 {
			// still we are at the first part
			right = append([]byte{x}, right...)
			split -= 8
		} else if split > 0 {
			// we should shift all the remaining value by this
			shift = split
			// drop the upper bytes
			val := x << (8 - split) >> (8 - split)
			right = append([]byte{val}, right...)

			split = 0

			val = x >> shift
			left = append(left, val)
		} else {
			if shift > 0 {
				left[0] += +x << (8 - shift)
			}
			left = append([]byte{x >> shift}, left...)
		}
	}
	return left, right
}
