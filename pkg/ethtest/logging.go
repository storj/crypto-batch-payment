package ethtest

import (
	"testing"

	"github.com/ethereum/go-ethereum/log"
)

func EnableLogging(t *testing.T) {
	fmt := log.TerminalFormat(false)
	log.Root().SetHandler(log.FuncHandler(func(r *log.Record) error {
		t.Logf(string(fmt.Format(r)))
		return nil
	}))
}
