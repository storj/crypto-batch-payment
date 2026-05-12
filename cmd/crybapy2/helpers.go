package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/manifoldco/promptui"
	"github.com/zeebo/errs"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
)

func promptConfirm(label string) error {
	_, err := (&promptui.Prompt{
		Label:     label,
		IsConfirm: true,
	}).Run()
	if err != nil {
		return errors.New("aborted")
	}
	return nil
}

type dbMap map[payer.Type]*pipelinedb.DB

func (m dbMap) Close() error {
	var g errs.Group
	for _, db := range m {
		g.Add(db.Close())
	}
	return g.Err()
}

func loadDBs(ctx context.Context) (dbs dbMap, err error) {
	defer func() {
		if err != nil {
			_ = dbs.Close()
		}
	}()

	dbPaths, err := filepath.Glob("./payout.*.db")
	if err != nil {
		return nil, fmt.Errorf("failed to determine payout database paths: %w", err)
	}
	if len(dbPaths) == 0 {
		return nil, errors.New(`no payout databases found in the current directory; did you run "init"?`)
	}

	dbs = make(dbMap)
	for _, dbPath := range dbPaths {
		var payerType payer.Type
		switch dbPath {
		case "payout.eth.db":
			payerType = payer.Eth
		case "payout.zksync-era.db":
			payerType = payer.ZkSyncEra
		default:
			return nil, fmt.Errorf("no payer type configured for payout database %q", dbPath)
		}

		db, err := pipelinedb.OpenDB(ctx, dbPath, false)
		if err != nil {
			return nil, fmt.Errorf("failed to open payout database: %w", err)
		}
		dbs[payerType] = db
	}

	return dbs, nil
}
