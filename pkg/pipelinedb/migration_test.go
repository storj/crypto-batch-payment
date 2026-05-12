package pipelinedb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
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
	schema, err := os.ReadFile(fmt.Sprintf("testdata/v%d.sql", version))
	require.NoError(t, err)

	createDB := func(t *testing.T) string {
		dbPath := filepath.Join(t.TempDir(), "test.db")
		return dbPath
	}

	doRaw := func(t *testing.T, dbPath string, fn func(*testing.T, *sql.DB)) {
		rawDB, err := sql.Open("sqlite3", dbPath)
		require.NoError(t, err)
		defer func() {
			assert.NoError(t, rawDB.Close())
		}()
		fn(t, rawDB)
	}

	doTest := func(t *testing.T, readOnly bool) {
		ctx := context.Background()
		dbPath := createDB(t)

		// Set up schema for version being migrated from
		doRaw(t, dbPath, func(t *testing.T, rawDB *sql.DB) {
			_, err = rawDB.Exec(string(schema))
			require.NoError(t, err)
		})

		// Do the migration
		db, err := OpenDB(ctx, dbPath, readOnly)
		require.NoError(t, err)

		// Try to gather stats. This isn't an exhaustive check that the
		// migration succeeded but reads from most tables.
		_, err = db.Stats(ctx, decimal.NewFromInt(0))
		assert.NoError(t, err)

		// If not readOnly, try to attempt a write to make sure the database
		// is still open in the correct mode.
		if !readOnly {
			assert.NoError(t, db.CreatePayoutGroup(ctx, 99, nil), "failed to create payout group")
		}

		require.NoError(t, db.Close())

		// Assert that the version has been updated to the latest
		doRaw(t, dbPath, func(t *testing.T, rawDB *sql.DB) {
			var gotVersion int
			require.NoError(t, rawDB.QueryRow("SELECT version FROM metadata").Scan(&gotVersion))
			assert.Equal(t, dbVersion, gotVersion)
		})
	}

	t.Run("readOnly", func(t *testing.T) {
		doTest(t, true)
	})

	t.Run("readWrite", func(t *testing.T) {
		doTest(t, false)
	})
}
