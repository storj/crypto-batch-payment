package payouts

import "path/filepath"

const (
	dbName = "payouts.db"
)

func DBPathFromDir(dir string) string {
	return filepath.Join(dir, dbName)
}
