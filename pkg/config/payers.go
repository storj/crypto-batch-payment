package config

import "storj.io/crypto-batch-payment/pkg/payer"

type Payer interface {
	payer.Payer
	Close()
}

type payerWrapper struct {
	payer.Payer
	closeFunc func()
}

func (w payerWrapper) Close() {
	if w.closeFunc != nil {
		w.closeFunc()
	}
}

type Payers map[payer.Type]Payer

func (ps *Payers) Add(t payer.Type, p Payer) {
	if *ps == nil {
		*ps = make(map[payer.Type]Payer)
	}
	(*ps)[t] = p
}

func (ps Payers) Close() {
	for _, p := range ps {
		p.Close()
	}
}
