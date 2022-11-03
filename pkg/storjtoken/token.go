// Package storjtoken provides STORJ token related functionality
package storjtoken

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

var (
	// DefaultContractAddress is the address of the STORJ Token Contract
	// on the Ethereum network.
	DefaultContractAddress = common.HexToAddress("0xb64ef51c888972c908cfacf59b47c1afbc0ab8ac")

	// DefaultChainID is the chain id for Mainnet:
	// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md#list-of-chain-ids
	DefaultChainID = big.NewInt(0x1)
)

// FromUSD converts from USD to STORJ WEI, at the given price (rounded down).
func FromUSD(usd, price decimal.Decimal, decimals int32) *big.Int {
	if decimals < 8 {
		// this should never happen based on our knowledge. ETH STORJ token uses 8, Polygon uses 18
		panic("Decimals (digits of token) is less then 8. We don't support such an ERC-20 token for safety reasons. Potential overpayment!!!")
	}
	tokens, _ := usd.Shift(decimals).QuoRem(price, 0)
	// The quotient should have an exponent of zero, since it has no fractional
	// part, so returning the coefficient should represent the number of
	// tokens.
	return tokens.Coefficient()
}

func Pretty(token *big.Int, digits int32) string {
	return fmt.Sprintf("%s (%s STORJ)", token, decimal.NewFromBigInt(token, -digits).String())
}
