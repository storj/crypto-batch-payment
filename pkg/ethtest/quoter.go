package ethtest

import (
	"context"
	"sync"

	"github.com/zeebo/errs"

	"storj.io/crypto-batch-payment/pkg/coinmarketcap"
)

type Quoter struct {
	mu     sync.RWMutex
	quotes map[coinmarketcap.Symbol]*coinmarketcap.Quote
}

var _ coinmarketcap.Quoter = (*Quoter)(nil)

func NewQuoter() *Quoter {
	return &Quoter{
		quotes: make(map[coinmarketcap.Symbol]*coinmarketcap.Quote),
	}
}

func (quoter *Quoter) GetQuote(ctx context.Context, symbol coinmarketcap.Symbol) (*coinmarketcap.Quote, error) {
	// Take the client-wide lock and obtain the symbol cache
	quoter.mu.RLock()
	defer quoter.mu.RUnlock()

	quote, ok := quoter.quotes[symbol]
	if !ok {
		return nil, errs.New("no quote for %q", symbol)
	}

	return quote, nil
}

func (quoter *Quoter) SetQuote(symbol coinmarketcap.Symbol, quote *coinmarketcap.Quote) {
	quoter.mu.Lock()
	defer quoter.mu.Unlock()
	quoter.quotes[symbol] = quote
}
