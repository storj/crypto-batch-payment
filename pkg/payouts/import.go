package payouts

import (
	"context"
	"os"
	"path/filepath"

	"storj.io/crypto-batch-payment/pkg/pipelinedb"

	"github.com/zeebo/errs"

	"storj.io/crypto-batch-payment/pkg/csv"
)

func Import(ctx context.Context, dir string, csvPath string) error {
	dbDir, err := dbDirFromCSVPath(dir, csvPath)
	if err != nil {
		return err
	}
	dbPath := DBPathFromDir(dbDir)

	// Make sure the database does not already exist
	_, err = os.Stat(dbPath)
	switch {
	case err == nil:
		return errs.New("%q has already been imported to %q", csvPath, dbPath)
	case !os.IsNotExist(err):
		return errs.Wrap(err)
	}

	if err := importPayouts(ctx, csvPath, dbDir); err != nil {
		return err
	}

	return nil
}

func dbDirFromCSVPath(dir string, csvPath string) (string, error) {
	base := filepath.Base(csvPath)
	ext := filepath.Ext(csvPath)
	if ext == "" {
		return "", errs.New("%q must have an extension", csvPath)
	}
	name := base[:len(base)-len(filepath.Ext(base))]
	return filepath.Join(dir, name), nil
}

func importPayouts(ctx context.Context, csvPath, dir string) error {
	rows, err := csv.Load(csvPath)
	if err != nil {
		return err
	}
	payouts := PayoutsFromCSV(rows)

	// ensure the parent directory exists
	if err := os.MkdirAll(filepath.Dir(dir), 0755); err != nil {
		return errs.Wrap(err)
	}

	tmpDir, err := os.MkdirTemp(filepath.Dir(dir), "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	db, err := pipelinedb.NewDB(ctx, filepath.Join(tmpDir, dbName))
	if err != nil {
		return errs.Wrap(err)
	}

	if err := createPayoutGroups(ctx, db, payouts); err != nil {
		return err
	}

	if err := db.Close(); err != nil {
		return errs.Wrap(err)
	}

	if err := os.Rename(tmpDir, dir); err != nil {
		return errs.Wrap(err)
	}

	return nil
}

func createPayoutGroups(ctx context.Context, db *pipelinedb.DB, payouts []*pipelinedb.Payout) error {
	// Right now we only support a single transfer per transaction. Make a
	// payout group for each row.
	for i := range payouts {
		if err := db.CreatePayoutGroup(ctx, int64(i+1), payouts[i:i+1]); err != nil {
			return err
		}
	}
	return nil
}
