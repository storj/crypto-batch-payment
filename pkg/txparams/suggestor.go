package txparams

import (
	"context"
	"math/big"
)

type GasCaps struct {
	// GasFeeCap is the EIP-1559 max fee. If nil, the payer will determine this
	// value.
	GasFeeCap *big.Int

	// GasTipCap is the EIP-1559 priority fee. If nil, the payer will determine
	// this value.
	GasTipCap *big.Int
}

type Getter interface {
	GetGasCaps(ctx context.Context) (GasCaps, error)
}

type FixedGasCaps GasCaps

func (f FixedGasCaps) GetGasCaps(ctx context.Context) (GasCaps, error) {
	return GasCaps(f), nil
}
