// Package storjtoken provides ERC-20 payout token related functionality
// (STORJ, USDC).
package storjtoken

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/zeebo/errs"

	"storj.io/crypto-batch-payment/pkg/coinmarketcap"
)

// USDC is the symbol of the USDC stablecoin, which is always priced at $1.
const USDC = "USDC"

var (
	// DefaultContractAddress is the address of the STORJ Token Contract
	// on the Ethereum network.
	DefaultContractAddress = common.HexToAddress("0xb64ef51c888972c908cfacf59b47c1afbc0ab8ac")

	// DefaultChainID is the chain id for Mainnet:
	// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md#list-of-chain-ids
	DefaultChainID = big.NewInt(0x1)
)

// GetPrice returns the USD price of the token with the given symbol. USDC is
// pinned to $1 without consulting the quoter.
func GetPrice(ctx context.Context, quoter coinmarketcap.Quoter, symbol string) (decimal.Decimal, error) {
	if symbol == USDC {
		return decimal.NewFromInt(1), nil
	}
	quote, err := quoter.GetQuote(ctx, coinmarketcap.Symbol(symbol))
	if err != nil {
		return decimal.Decimal{}, errs.Wrap(err)
	}
	return quote.Price, nil
}

// FromUSD converts from USD to token WEI, at the given price (rounded down).
func FromUSD(usd, price decimal.Decimal, decimals int32) *big.Int {
	if decimals < 6 {
		// this should never happen based on our knowledge. ETH STORJ token uses 8, Polygon uses 18, USDC uses 6
		panic("Decimals (digits of token) is less then 6. We don't support such an ERC-20 token for safety reasons. Potential overpayment!!!")
	}
	tokens, _ := usd.Shift(decimals).QuoRem(price, 0)
	// The quotient should have an exponent of zero, since it has no fractional
	// part, so returning the coefficient should represent the number of
	// tokens.
	return tokens.Coefficient()
}

// Pretty formats the raw token amount together with its real value.
func Pretty(token *big.Int, digits int32, symbol string) string {
	return fmt.Sprintf("%s (%s %s)", token, decimal.NewFromBigInt(token, -digits).String(), symbol)
}
