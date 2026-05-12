package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/zeebo/clingy"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	ok, err := clingy.Environment{}.Run(ctx, func(cmds clingy.Commands) {
		cmds.New("audit", "Audits payouts", new(cmdAudit))
		cmds.New("init", "Initializes payouts from prepayment CSVs", new(cmdInit))
		cmds.New("run", "Runs the payouts pipeline", new(cmdRun))
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed: %+v\n", err)
		return err
	}
	if !ok {
		return errors.New("usage error")
	}
	return nil
}
