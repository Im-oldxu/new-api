package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestArchiveLegacyGatewayTablesPreservesDataAndIsIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	for _, table := range legacyGatewayTables {
		require.NoError(t, db.Exec("CREATE TABLE "+table.source+" (id INTEGER PRIMARY KEY, name TEXT)").Error)
		require.NoError(t, db.Exec("INSERT INTO "+table.source+" (id, name) VALUES (?, ?)", 1, table.source).Error)
	}

	require.NoError(t, archiveLegacyGatewayTables(db))
	for _, table := range legacyGatewayTables {
		assert.False(t, db.Migrator().HasTable(table.source))
		assert.True(t, db.Migrator().HasTable(table.archive))

		var row struct {
			Name string
		}
		require.NoError(t, db.Table(table.archive).First(&row).Error)
		assert.Equal(t, table.source, row.Name)
	}

	require.NoError(t, archiveLegacyGatewayTables(db))
}

func TestArchiveLegacyGatewayTablesRollsBackOnDestinationCollision(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec("CREATE TABLE abilities (id INTEGER PRIMARY KEY)").Error)
	require.NoError(t, db.Exec("CREATE TABLE legacy_channels (id INTEGER PRIMARY KEY)").Error)
	require.NoError(t, db.Exec("CREATE TABLE channels (id INTEGER PRIMARY KEY)").Error)

	err = archiveLegacyGatewayTables(db)
	require.ErrorContains(t, err, "destination table legacy_channels already exists")
	assert.True(t, db.Migrator().HasTable("abilities"), "the earlier rename must roll back")
	assert.False(t, db.Migrator().HasTable("legacy_abilities"))
	assert.True(t, db.Migrator().HasTable("channels"))
}
