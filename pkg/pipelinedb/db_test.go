package pipelinedb

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckpointOnClose(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	assertFiles := func(t *testing.T, want ...string) {
		t.Helper()
		assert.Equal(t, want, readDir(t, dir))
	}

	// Test case order is important. The first test case creates the database
	// and must execute first.

	t.Run("create read-write", func(t *testing.T) {
		assertFiles(t)

		db, err := NewDB(context.Background(), dbPath)
		require.NoError(t, err)
		assertFiles(t, "test.db", "test.db-shm", "test.db-wal")

		assert.NoError(t, db.Close())
		assertFiles(t, "test.db")
	})

	t.Run("open read-write", func(t *testing.T) {
		assertFiles(t, "test.db")

		db, err := OpenDB(context.Background(), dbPath, false)
		require.NoError(t, err)
		assertFiles(t, "test.db", "test.db-shm", "test.db-wal")

		assert.NoError(t, db.Close())
		assertFiles(t, "test.db")
	})

	t.Run("open read-only", func(t *testing.T) {
		assertFiles(t, "test.db")

		db, err := OpenDB(context.Background(), dbPath, true)
		require.NoError(t, err)
		assertFiles(t, "test.db", "test.db-shm", "test.db-wal")

		assert.NoError(t, db.Close())
		assertFiles(t, "test.db")
	})
}

func readDir(t *testing.T, dir string) []string {
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}
