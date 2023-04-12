package payouts

import "path/filepath"

const (
	dbName = "payouts.db"
)

func DbPathFromDir(dir string) string {
	return filepath.Join(dir, dbName)
}
