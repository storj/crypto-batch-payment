package payouts

import "path/filepath"

const (
	dbName = "payouts.db"
)

func dbPathFromDir(dir string) string {
	return filepath.Join(dir, dbName)
}
