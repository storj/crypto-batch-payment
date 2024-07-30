package config

import "storj.io/crypto-batch-payment/pkg/payer"

type Auditor interface {
	payer.Auditor
	Close()
}

type auditorWrapper struct {
	payer.Auditor
	closeFunc func()
}

func (w auditorWrapper) Close() {
	if w.closeFunc != nil {
		w.closeFunc()
	}
}

type Auditors map[payer.Type]Auditor

func (as *Auditors) Add(t payer.Type, a Auditor) {
	if *as == nil {
		*as = make(map[payer.Type]Auditor)
	}
	(*as)[t] = a
}

func (as Auditors) Close() {
	for _, a := range as {
		a.Close()
	}
}
