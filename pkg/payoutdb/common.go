package payoutdb

import "context"

func (db *DB) WithTx(ctx context.Context, fn func(*Tx) error) (err error) {
	tx, err := db.Open(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
		} else {
			tx.Rollback() // log this perhaps?
		}
	}()
	return fn(tx)
}
