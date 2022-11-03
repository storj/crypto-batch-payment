package pipeline

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBigIntRatio(t *testing.T) {
	ten := big.NewInt(10)

	ratio := bigIntRatio(ten, 0.1)
	require.Equal(t, big.NewInt(1), ratio)

	ratio = bigIntRatio(ten, 1.0)
	require.Equal(t, big.NewInt(10), ratio)

	ratio = bigIntRatio(ten, 2.0)
	require.Equal(t, big.NewInt(20), ratio)
}
