package pipeline

import "math/big"

// bigIntRatio calculates a ratio of a big int
func bigIntRatio(gasPrice *big.Int, ratio float64) *big.Int {
	r := big.NewFloat(ratio)
	bump, _ := r.Mul(r, new(big.Float).SetInt(gasPrice)).Int(nil)
	return bump
}
