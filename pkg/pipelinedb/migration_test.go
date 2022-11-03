package pipelinedb

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration(t *testing.T) {
	for version := 1; version < dbVersion; version++ {
		t.Run(fmt.Sprintf("migration from %d", version), func(t *testing.T) {
			testMigration(t, version)
		})
	}
}

func testMigration(t *testing.T, version int) {
	schema, err := ioutil.ReadFile(fmt.Sprintf("testdata/v%d.sql", version))
	require.NoError(t, err)

	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "test.db")

	rawDB, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	_, err = rawDB.Exec(string(schema))
	require.NoError(t, err)
	err = rawDB.Close()
	require.NoError(t, err)

	db, err := OpenDB(context.Background(), dbPath, false)
	require.NoError(t, err)

	_, err = db.Stats(context.Background())
	require.NoError(t, err)
}
